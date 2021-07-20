//  Copyright 2021 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package daisy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	healthCheckURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/healthChecks/(?P<healthCheck>%[2]s)$`, projectRgxStr, rfc1035))
)

func (w *Workflow) healthCheckExists(project, healthCheck string) (bool, DError) {
	return w.healthCheckCache.resourceExists(func(project string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListHealthChecks(project)
	}, project, healthCheck)
}

// HealthCheck is used to create a GCE healthCheck.
type HealthCheck struct {
	compute.HealthCheck
	Resource
}

// MarshalJSON is a hacky workaround to compute.HealthCheck's implementation.
func (fir *HealthCheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(*fir)
}

func (hc *HealthCheck) populate(ctx context.Context, s *Step) DError {
	var errs DError
	hc.Name, errs = hc.Resource.populateWithGlobal(ctx, s, hc.Name)
	fmt.Printf("sbsbsbsb"+hc.Name)
	hc.Description = strOr(hc.Description, defaultDescription("HealthCheck", s.w.Name, s.w.username))
	hc.link = fmt.Sprintf("projects/%s/global/healthChecks/%s", hc.Project, hc.Name)
	return errs
}

func (hc *HealthCheck) validate(ctx context.Context, s *Step) DError {
	pre := fmt.Sprintf("cannot create health-check %q", hc.daisyName)
	errs := hc.Resource.validate(ctx, s, pre)

	// Register creation.
	errs = addErrs(errs, s.w.healthChecks.regCreate(hc.daisyName, &hc.Resource, s, false))
	return errs
}

type healthCheckConnection struct {
	connector, disconnector *Step
}

type healthCheckRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*healthCheckConnection
	testDisconnectHelper func(nName, iName string, s *Step) DError
}

func newHealthCheckRegistry(w *Workflow) *healthCheckRegistry {
	hcr := &healthCheckRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "healthCheck", urlRgx: healthCheckURLRegex}}
	hcr.baseResourceRegistry.deleteFn = hcr.deleteFn
	hcr.connections = map[string]map[string]*healthCheckConnection{}
	hcr.init()
	return hcr
}

func (hcr *healthCheckRegistry) deleteFn(res *Resource) DError {
	m := NamedSubexp(healthCheckURLRegex, res.link)
	err := hcr.w.ComputeClient.DeleteHealthCheck(m["project"], m["healthCheck"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete health check", err)
	}
	return newErr("failed to delete health check", err)
}

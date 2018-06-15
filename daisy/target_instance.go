//  Copyright 2018 Google Inc. All Rights Reserved.
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
	"sync"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	targetInstanceCache struct {
		exists map[string]map[string][]string
		mu     sync.Mutex
	}
	targetInstanceURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/TargetInstances/(?P<targetInstance>%[2]s)$`, projectRgxStr, rfc1035))
)

func targetInstanceExists(client daisyCompute.Client, project, zone, name string) (bool, dErr) {
	targetInstanceCache.mu.Lock()
	defer targetInstanceCache.mu.Unlock()
	if targetInstanceCache.exists == nil {
		targetInstanceCache.exists = map[string]map[string][]string{}
	}
	if _, ok := targetInstanceCache.exists[project]; !ok {
		targetInstanceCache.exists[project] = map[string][]string{}
	}
	if _, ok := targetInstanceCache.exists[project][zone]; !ok {
		nl, err := client.ListTargetInstances(project, zone)
		if err != nil {
			return false, errf("error listing target-instances for project %q: %v", project, err)
		}
		var targetInstances []string
		for _, ti := range nl {
			targetInstances = append(targetInstances, ti.Name)
		}
		targetInstanceCache.exists[project][zone] = targetInstances
	}
	return strIn(name, targetInstanceCache.exists[project][zone]), nil
}

// TargetInstance is used to create a GCE targetInstance.
type TargetInstance struct {
	compute.TargetInstance
	Resource
}

// MarshalJSON is a hacky workaround to compute.TargetInstance's implementation.
func (ti *TargetInstance) MarshalJSON() ([]byte, error) {
	return json.Marshal(*ti)
}

func (ti *TargetInstance) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	ti.Name, ti.Zone, errs = ti.Resource.populate(ctx, s, ti.Name, ti.Zone)

	if instanceURLRgx.MatchString(ti.Instance) {
		ti.Instance = extendPartialURL(ti.Instance, ti.Project)
	} else {
		ti.Instance = fmt.Sprintf("projects/%s/zones/%s/instances/%s", ti.Project, ti.Zone, ti.Instance)
	}

	ti.Description = strOr(ti.Description, defaultDescription("TargetInstance", s.w.Name, s.w.username))
	ti.link = fmt.Sprintf("projects/%s/zones/%s/TargetInstances/%s", ti.Project, ti.Zone, ti.Name)
	return errs
}

func (ti *TargetInstance) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create target-instance %q", ti.daisyName)
	errs := ti.Resource.validateWithZone(ctx, s, ti.Zone, pre)

	if ti.Instance == "" {
		errs = addErrs(errs, errf("%s: Instance not set", pre))
	}

	// Register creation.
	errs = addErrs(errs, s.w.targetInstances.regCreate(ti.daisyName, &ti.Resource, s, false))
	return errs
}

type targetInstanceConnection struct {
	connector, disconnector *Step
}

type targetInstanceRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*targetInstanceConnection
	testDisconnectHelper func(nName, iName string, s *Step) dErr
}

func newTargetInstanceRegistry(w *Workflow) *targetInstanceRegistry {
	tir := &targetInstanceRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "targetInstance", urlRgx: targetInstanceURLRegex}}
	tir.baseResourceRegistry.deleteFn = tir.deleteFn
	tir.connections = map[string]map[string]*targetInstanceConnection{}
	tir.init()
	return tir
}

func (tir *targetInstanceRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(targetInstanceURLRegex, res.link)
	err := tir.w.ComputeClient.DeleteTargetInstance(m["project"], m["zone"], m["targetInstance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

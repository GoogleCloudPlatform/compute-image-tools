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
	forwardingRuleCache struct {
		exists map[string]map[string][]string
		mu     sync.Mutex
	}
	forwardingRuleURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?regions/(?P<region>%[2]s)/forwardingRules/(?P<forwardingRule>%[2]s)$`, projectRgxStr, rfc1035))
)

func forwardingRuleExists(client daisyCompute.Client, project, region, name string) (bool, dErr) {
	forwardingRuleCache.mu.Lock()
	defer forwardingRuleCache.mu.Unlock()
	if forwardingRuleCache.exists == nil {
		forwardingRuleCache.exists = map[string]map[string][]string{}
	}
	if _, ok := forwardingRuleCache.exists[project]; !ok {
		forwardingRuleCache.exists[project] = map[string][]string{}
	}
	if _, ok := forwardingRuleCache.exists[project][region]; !ok {
		nl, err := client.ListForwardingRules(project, region)
		if err != nil {
			return false, errf("error listing forwarding-rules for project %q: %v", project, err)
		}
		var forwardingRules []string
		for _, fr := range nl {
			forwardingRules = append(forwardingRules, fr.Name)
		}
		forwardingRuleCache.exists[project][region] = forwardingRules
	}
	return strIn(name, forwardingRuleCache.exists[project][region]), nil
}

// ForwardingRule is used to create a GCE forwardingRule.
type ForwardingRule struct {
	compute.ForwardingRule
	Resource
}

// MarshalJSON is a hacky workaround to compute.ForwardingRule's implementation.
func (fr *ForwardingRule) MarshalJSON() ([]byte, error) {
	return json.Marshal(*fr)
}

func (fr *ForwardingRule) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	fr.Name, fr.Region, errs = fr.Resource.populateWithRegion(ctx, s, fr.Name, fr.Region)

	if targetInstanceURLRegex.MatchString(fr.Target) {
		fr.Target = extendPartialURL(fr.Target, fr.Project)
	} else {
		fr.Target = fmt.Sprintf("projects/%s/zones/%s/targetInstances/%s", fr.Project, s.w.Zone, fr.Target)
	}

	fr.Description = strOr(fr.Description, defaultDescription("ForwardingRule", s.w.Name, s.w.username))
	fr.link = fmt.Sprintf("projects/%s/regions/%s/forwardingRules/%s", fr.Project, fr.Region, fr.Name)
	return errs
}

func (fr *ForwardingRule) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create forwarding-rule %q", fr.daisyName)
	errs := fr.Resource.validateWithRegion(ctx, s, fr.Region, pre)

	if fr.IPProtocol == "" {
		errs = addErrs(errs, errf("%s: IPProtocol not set", pre))
	}
	if fr.PortRange == "" {
		errs = addErrs(errs, errf("%s: PortRange not set", pre))
	}
	if fr.Target == "" {
		errs = addErrs(errs, errf("%s: Target not set", pre))
	}

	// Register creation.
	errs = addErrs(errs, s.w.forwardingRules.regCreate(fr.daisyName, &fr.Resource, s, false))
	return errs
}

type forwardingRuleConnection struct {
	connector, disconnector *Step
}

type forwardingRuleRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*forwardingRuleConnection
	testDisconnectHelper func(nName, iName string, s *Step) dErr
}

func newForwardingRuleRegistry(w *Workflow) *forwardingRuleRegistry {
	tir := &forwardingRuleRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "forwardingRule", urlRgx: forwardingRuleURLRegex}}
	tir.baseResourceRegistry.deleteFn = tir.deleteFn
	tir.connections = map[string]map[string]*forwardingRuleConnection{}
	tir.init()
	return tir
}

func (tir *forwardingRuleRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(forwardingRuleURLRegex, res.link)
	err := tir.w.ComputeClient.DeleteForwardingRule(m["project"], m["region"], m["forwardingRule"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

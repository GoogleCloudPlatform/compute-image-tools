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
	firewallRuleCache struct {
		exists map[string][]string
		mu     sync.Mutex
	}
	firewallRuleURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/firewalls/(?P<firewallRule>%[2]s)$`, projectRgxStr, rfc1035))
)

func firewallRuleExists(client daisyCompute.Client, project, name string) (bool, dErr) {
	firewallRuleCache.mu.Lock()
	defer firewallRuleCache.mu.Unlock()
	if firewallRuleCache.exists == nil {
		firewallRuleCache.exists = map[string][]string{}
	}
	if _, ok := firewallRuleCache.exists[project]; !ok {
		nl, err := client.ListFirewallRules(project)
		if err != nil {
			return false, errf("error listing firewall-rules for project %q: %v", project, err)
		}
		var firewallRules []string
		for _, fir := range nl {
			firewallRules = append(firewallRules, fir.Name)
		}
		firewallRuleCache.exists[project] = firewallRules
	}
	return strIn(name, firewallRuleCache.exists[project]), nil
}

// FirewallRule is used to create a GCE firewallRule.
type FirewallRule struct {
	compute.Firewall
	Resource
}

// MarshalJSON is a hacky workaround to compute.Firewall's implementation.
func (fir *FirewallRule) MarshalJSON() ([]byte, error) {
	return json.Marshal(*fir)
}

func (fir *FirewallRule) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	fir.Name, errs = fir.Resource.populateWithGlobal(ctx, s, fir.Name)

	if networkURLRegex.MatchString(fir.Network) {
		fir.Network = extendPartialURL(fir.Network, fir.Project)
	}

	fir.Description = strOr(fir.Description, defaultDescription("FirewallRule", s.w.Name, s.w.username))
	fir.link = fmt.Sprintf("projects/%s/global/firewalls/%s", fir.Project, fir.Name)
	return errs
}

func (fir *FirewallRule) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create firewall-rule %q", fir.daisyName)
	errs := fir.Resource.validate(ctx, s, pre)

	if fir.Network == "" {
		errs = addErrs(errs, errf("%s: Network not set", pre))
	}

	// Register creation.
	errs = addErrs(errs, s.w.firewallRules.regCreate(fir.daisyName, &fir.Resource, s, false))
	return errs
}

type firewallRuleConnection struct {
	connector, disconnector *Step
}

type firewallRuleRegistry struct {
	baseResourceRegistry
	connections          map[string]map[string]*firewallRuleConnection
	testDisconnectHelper func(nName, iName string, s *Step) dErr
}

func newFirewallRuleRegistry(w *Workflow) *firewallRuleRegistry {
	frr := &firewallRuleRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "firewallRule", urlRgx: firewallRuleURLRegex}}
	frr.baseResourceRegistry.deleteFn = frr.deleteFn
	frr.connections = map[string]map[string]*firewallRuleConnection{}
	frr.init()
	return frr
}

func (frr *firewallRuleRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(firewallRuleURLRegex, res.link)
	err := frr.w.ComputeClient.DeleteFirewallRule(m["project"], m["firewallRule"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

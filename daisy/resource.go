//  Copyright 2017 Google Inc. All Rights Reserved.
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
	"fmt"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

// Resource is the base struct for Daisy representation structs for GCE resources.
// This base struct defines some common user-definable fields, as well as some Daisy bookkeeping fields.
type Resource struct {
	// If this is unset Workflow.Project is used.
	Project string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool `json:",omitempty"`
	// If set Daisy will use this as the resource name instead of generating a name. Mutually exclusive with ExactName.
	RealName string `json:",omitempty"`
	// If set, Daisy will use the exact name as specified by the user instead of generating a name. Mutually exclusive with RealName.
	ExactName bool `json:",omitempty"`

	// The name of the disk as known to Daisy and the Daisy user.
	daisyName string

	link        string
	deleted     bool
	stoppedByWf bool
	startedByWf bool
	deleteMx    *sync.Mutex

	creator, deleter  *Step
	createdInWorkflow bool
	users             []*Step
}

func (r *Resource) populateWithGlobal(ctx context.Context, s *Step, name string) (string, DError) {
	errs := r.populateHelper(ctx, s, name)
	return r.RealName, errs
}

func (r *Resource) populateWithZone(ctx context.Context, s *Step, name, zone string) (string, string, DError) {
	errs := r.populateHelper(ctx, s, name)
	return r.RealName, strOr(zone, s.w.Zone), errs
}

func (r *Resource) populateWithRegion(ctx context.Context, s *Step, name, region string) (string, string, DError) {
	errs := r.populateHelper(ctx, s, name)
	return r.RealName, strOr(region, getRegionFromZone(s.w.Zone)), errs
}

func (r *Resource) populateHelper(ctx context.Context, s *Step, name string) DError {
	var errs DError
	if r.ExactName && r.RealName != "" {
		errs = addErrs(errs, Errf("ExactName and RealName must be used mutually exclusively"))
	} else if r.ExactName {
		r.RealName = name
	} else if r.RealName == "" {
		r.RealName = s.w.genName(name)
	}
	r.daisyName = name
	r.Project = strOr(r.Project, s.w.Project)
	return errs
}

func (r *Resource) validate(ctx context.Context, s *Step, errPrefix string) DError {
	var errs DError

	if !checkName(r.RealName) {
		return Errf("%s: bad name: %q", errPrefix, r.RealName)
	}

	if exists, err := projectExists(s.w.ComputeClient, r.Project); err != nil {
		errs = addErrs(errs, Errf("%s: bad project lookup: %q, error: %v", errPrefix, r.Project, err))
	} else if !exists {
		errs = addErrs(errs, Errf("%s: project does not exist: %q", errPrefix, r.Project))
	}
	return errs
}

func (r *Resource) validateWithZone(ctx context.Context, s *Step, z, errPrefix string) DError {
	errs := r.validate(ctx, s, errPrefix)
	if z == "" {
		errs = addErrs(errs, Errf("%s: no zone provided in step or workflow", errPrefix))
	}
	if exists, err := zoneExists(s.w.ComputeClient, r.Project, z); err != nil {
		errs = addErrs(errs, Errf("%s: bad zone lookup: %q, error: %v", errPrefix, z, err))
	} else if !exists {
		errs = addErrs(errs, Errf("%s: zone does not exist: %q", errPrefix, z))
	}
	return errs
}

func (r *Resource) validateWithRegion(ctx context.Context, s *Step, re, errPrefix string) DError {
	errs := r.validate(ctx, s, errPrefix)
	if re == "" {
		errs = addErrs(errs, Errf("%s: no region provided in step or workflow", errPrefix))
	}
	if exists, err := regionExists(s.w.ComputeClient, r.Project, re); err != nil {
		errs = addErrs(errs, Errf("%s: bad region lookup: %q, error: %v", errPrefix, re, err))
	} else if !exists {
		errs = addErrs(errs, Errf("%s: region does not exist: %q", errPrefix, re))
	}
	return errs
}

func defaultDescription(resourceTypeName, wfName, user string) string {
	return fmt.Sprintf("%s created by Daisy in workflow %q on behalf of %s.", resourceTypeName, wfName, user)
}

func extendPartialURL(url, project string) string {
	if strings.HasPrefix(url, "projects") {
		return url
	}
	return fmt.Sprintf("projects/%s/%s", project, url)
}

func resourceExists(client compute.Client, url string) (bool, DError) {
	if !strings.HasPrefix(url, "projects/") {
		return false, Errf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	switch {
	case machineTypeURLRegex.MatchString(url):
		result := NamedSubexp(machineTypeURLRegex, url)
		return machineTypeExists(client, result["project"], result["zone"], result["machinetype"])
	case instanceURLRgx.MatchString(url):
		result := NamedSubexp(instanceURLRgx, url)
		return instanceExists(client, result["project"], result["zone"], result["instance"])
	case diskURLRgx.MatchString(url):
		result := NamedSubexp(diskURLRgx, url)
		return diskExists(client, result["project"], result["zone"], result["disk"])
	case imageURLRgx.MatchString(url):
		result := NamedSubexp(imageURLRgx, url)
		return imageExists(client, result["project"], result["family"], result["image"])
	case machineImageURLRgx.MatchString(url):
		result := NamedSubexp(machineImageURLRgx, url)
		return machineImageExists(client, result["project"], result["machineImage"])
	case networkURLRegex.MatchString(url):
		result := NamedSubexp(networkURLRegex, url)
		return networkExists(client, result["project"], result["network"])
	case subnetworkURLRegex.MatchString(url):
		result := NamedSubexp(subnetworkURLRegex, url)
		return subnetworkExists(client, result["project"], result["region"], result["subnetwork"])
	case targetInstanceURLRegex.MatchString(url):
		result := NamedSubexp(targetInstanceURLRegex, url)
		return targetInstanceExists(client, result["project"], result["zone"], result["targetInstance"])
	case forwardingRuleURLRegex.MatchString(url):
		result := NamedSubexp(forwardingRuleURLRegex, url)
		return forwardingRuleExists(client, result["project"], result["region"], result["forwardingRule"])
	case firewallRuleURLRegex.MatchString(url):
		result := NamedSubexp(firewallRuleURLRegex, url)
		return firewallRuleExists(client, result["project"], result["firewallRule"])
	}
	return false, Errf("unknown resource type: %q", url)
}

func resourceNameHelper(name string, w *Workflow, exactName bool) string {
	if !exactName {
		name = w.genName(name)
	}
	return name
}

type regionalResourceCache struct {
	exists map[string]map[string][]interface{}
	mu     sync.Mutex
}

func (rc *regionalResourceCache) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.exists = nil
}

type globalResourceCache struct {
	exists map[string][]interface{}
	mu     sync.Mutex
}

func (rc *globalResourceCache) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.exists = nil
}

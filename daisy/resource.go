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
	"reflect"
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
	if exists, err := s.w.zoneExists(r.Project, z); err != nil {
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
	if exists, err := s.w.regionExists(r.Project, re); err != nil {
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

func (w *Workflow) resourceExists(url string) (bool, DError) {
	if !strings.HasPrefix(url, "projects/") {
		return false, Errf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	switch {
	case machineTypeURLRegex.MatchString(url):
		result := NamedSubexp(machineTypeURLRegex, url)
		return w.machineTypeExists(result["project"], result["zone"], result["machinetype"])
	case instanceURLRgx.MatchString(url):
		result := NamedSubexp(instanceURLRgx, url)
		return w.instanceExists(result["project"], result["zone"], result["instance"])
	case diskURLRgx.MatchString(url):
		result := NamedSubexp(diskURLRgx, url)
		return w.diskExists(result["project"], result["zone"], result["disk"])
	case imageURLRgx.MatchString(url):
		result := NamedSubexp(imageURLRgx, url)
		return w.imageExists(result["project"], result["family"], result["image"])
	case machineImageURLRgx.MatchString(url):
		result := NamedSubexp(machineImageURLRgx, url)
		return w.machineImageExists(result["project"], result["machineImage"])
	case networkURLRegex.MatchString(url):
		result := NamedSubexp(networkURLRegex, url)
		return w.networkExists(result["project"], result["network"])
	case subnetworkURLRegex.MatchString(url):
		result := NamedSubexp(subnetworkURLRegex, url)
		return w.subnetworkExists(result["project"], result["region"], result["subnetwork"])
	case targetInstanceURLRegex.MatchString(url):
		result := NamedSubexp(targetInstanceURLRegex, url)
		return w.targetInstanceExists(result["project"], result["zone"], result["targetInstance"])
	case targetPoolURLRegex.MatchString(url):
		result := NamedSubexp(targetPoolURLRegex, url)
		return w.targetPoolExists(result["project"], result["region"], result["targetPool"])
	case forwardingRuleURLRegex.MatchString(url):
		result := NamedSubexp(forwardingRuleURLRegex, url)
		return w.forwardingRuleExists(result["project"], result["region"], result["forwardingRule"])
	case firewallRuleURLRegex.MatchString(url):
		result := NamedSubexp(firewallRuleURLRegex, url)
		return w.firewallRuleExists(result["project"], result["firewallRule"])
	case snapshotURLRgx.MatchString(url):
		result := NamedSubexp(snapshotURLRgx, url)
		return w.snapshotExists(result["project"], result["snapshot"])
	}
	return false, Errf("unknown resource type: %q", url)
}

func resourceNameHelper(name string, w *Workflow, exactName bool) string {
	if !exactName {
		name = w.genName(name)
	}
	return name
}

type twoDResourceCache struct {
	exists map[string]map[string]map[string]interface{}
	mu     sync.Mutex
}

type oneDResourceCache struct {
	exists map[string]map[string]interface{}
	mu     sync.Mutex
}

// resourceExists should only be used during validation for existing GCE
// resources and should not be relied or populated for daisy created resources.
func (c *twoDResourceCache) resourceExists(listResourceFunc func(project, regionOrZone string, opts ...compute.ListCallOption) (interface{}, error),
	project, regionOrZone, resourceName string) (bool, DError) {

	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.loadCache(listResourceFunc, project, regionOrZone, resourceName)
	if err != nil {
		return false, err
	}
	return nameInResourceMap(resourceName, c.exists[project][regionOrZone]), nil
}

func (c *twoDResourceCache) loadCache(listResourceFunc func(project string, regionOrZone string, opts ...compute.ListCallOption) (interface{}, error),
	project string, regionOrZone string, resourceName string) DError {

	if resourceName == "" {
		return Errf("must provide resource name")
	}
	if c.exists == nil {
		c.exists = map[string]map[string]map[string]interface{}{}
	}
	if _, ok := c.exists[project]; !ok {
		c.exists[project] = map[string]map[string]interface{}{}
	}
	if _, ok := c.exists[project][regionOrZone]; !ok {
		ri, err := listResourceFunc(project, regionOrZone)
		if err != nil {
			return typedErr(apiError, "error listing resource for project", err)
		}
		c.exists[project][regionOrZone] = toMap(ri)
	}
	return nil
}

// resourceExists should only be used during validation for existing GCE
// resources and should not be relied or populated for daisy created resources.
func (c *oneDResourceCache) resourceExists(listResourceFunc func(project string, opts ...compute.ListCallOption) (interface{}, error),
	project, resourceName string) (bool, DError) {

	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.loadCache(listResourceFunc, project, resourceName)
	if err != nil {
		return false, err
	}

	return nameInResourceMap(resourceName, c.exists[project]), nil
}

func (c *oneDResourceCache) loadCache(listResourceFunc func(project string, opts ...compute.ListCallOption) (interface{}, error), project string, resourceName string) DError {
	if resourceName == "" {
		return Errf("must provide resource name")
	}
	if c.exists == nil {
		c.exists = map[string]map[string]interface{}{}
	}
	if _, ok := c.exists[project]; !ok {
		ri, err := listResourceFunc(project)
		if err != nil {
			return typedErr(apiError, "error listing resource for project", err)
		}
		c.exists[project] = toMap(ri)
	}
	return nil
}

func toMap(slice interface{}) map[string]interface{} {
	s := reflect.ValueOf(slice)
	ret := make(map[string]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		r := s.Index(i).Interface()
		v := reflect.ValueOf(r)
		name := reflect.Indirect(v).FieldByName("Name").String()
		ret[name] = r
	}
	return ret
}

func nameInResourceMap(name string, m map[string]interface{}) bool {
	_, ok := m[name]
	return ok
}

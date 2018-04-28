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
	NoCleanup bool
	// If set Daisy will use this as the resource name instead of generating a name. Mutually exclusive with ExactName.
	RealName string `json:",omitempty"`
	// If set, Daisy will use the exact name as specified by the user instead of generating a name. Mutually exclusive with RealName.
	ExactName bool `json:",omitempty"`

	// The name of the disk as known to Daisy and the Daisy user.
	daisyName string

	link     string
	deleted  bool
	stopped  bool
	deleteMx *sync.Mutex

	creator, deleter *Step
	users            []*Step
}

func (r *Resource) populate(ctx context.Context, s *Step, name, zone string) (string, string, dErr) {
	var errs dErr
	if r.ExactName && r.RealName != "" {
		errs = addErrs(errs, errf("ExactName and RealName must be used mutually exclusively"))
	} else if r.ExactName {
		r.RealName = name
	} else if r.RealName == "" {
		r.RealName = s.w.genName(name)
	}
	r.daisyName = name
	r.Project = strOr(r.Project, s.w.Project)
	return r.RealName, strOr(zone, s.w.Zone), errs
}

func (r *Resource) validate(ctx context.Context, s *Step, errPrefix string) dErr {
	var errs dErr

	if !checkName(r.RealName) {
		return errf("%s: bad name: %q", errPrefix, r.RealName)
	}

	if exists, err := projectExists(s.w.ComputeClient, r.Project); err != nil {
		errs = addErrs(errs, errf("%s: bad project lookup: %q, error: %v", errPrefix, r.Project, err))
	} else if !exists {
		errs = addErrs(errs, errf("%s: project does not exist: %q", errPrefix, r.Project))
	}
	return errs
}

func (r *Resource) validateWithZone(ctx context.Context, s *Step, z, errPrefix string) dErr {
	errs := r.validate(ctx, s, errPrefix)
	if z == "" {
		errs = addErrs(errs, errf("%s: no zone provided in step or workflow", errPrefix))
	}
	if exists, err := zoneExists(s.w.ComputeClient, r.Project, z); err != nil {
		errs = addErrs(errs, errf("%s: bad zone lookup: %q, error: %v", errPrefix, z, err))
	} else if !exists {
		errs = addErrs(errs, errf("%s: zone does not exist: %q", errPrefix, z))
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

func resourceExists(client compute.Client, url string) (bool, dErr) {
	if !strings.HasPrefix(url, "projects/") {
		return false, errf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	switch {
	case machineTypeURLRegex.MatchString(url):
		result := namedSubexp(machineTypeURLRegex, url)
		return machineTypeExists(client, result["project"], result["zone"], result["machinetype"])
	case instanceURLRgx.MatchString(url):
		result := namedSubexp(instanceURLRgx, url)
		return instanceExists(client, result["project"], result["zone"], result["instance"])
	case diskURLRgx.MatchString(url):
		result := namedSubexp(diskURLRgx, url)
		return diskExists(client, result["project"], result["zone"], result["disk"])
	case imageURLRgx.MatchString(url):
		result := namedSubexp(imageURLRgx, url)
		return imageExists(client, result["project"], result["family"], result["image"])
	case networkURLRegex.MatchString(url):
		result := namedSubexp(networkURLRegex, url)
		return networkExists(client, result["project"], result["network"])
	}
	return false, errf("unknown resource type: %q", url)
}

func resourceNameHelper(name string, w *Workflow, exactName bool) string {
	if !exactName {
		name = w.genName(name)
	}
	return name
}

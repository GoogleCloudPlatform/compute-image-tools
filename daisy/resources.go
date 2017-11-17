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
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/googleapi"
)

type resource struct {
	real, link         string
	noCleanup, deleted bool
	mx                 sync.Mutex

	creator, deleter *Step
	users            []*Step
}

type baseResourceRegistry struct {
	w  *Workflow
	m  map[string]*resource
	mx sync.Mutex

	deleteFn func(res *resource) error
	typeName string
	urlRgx   *regexp.Regexp
}

func (r *baseResourceRegistry) init() {
	r.m = map[string]*resource{}
}

func (r *baseResourceRegistry) cleanup() {
	var wg sync.WaitGroup
	for name, res := range r.m {
		if res.noCleanup || res.deleted {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := r.delete(name); err != nil {
				if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code != 404 {
					fmt.Println(err)
				}
			}
		}(name)
	}
	wg.Wait()
}

func (r *baseResourceRegistry) delete(name string) error {
	res, ok := r.get(name)
	if !ok {
		return fmt.Errorf("cannot delete %q; does not exist in resource map", name)
	}
	res.mx.Lock()
	defer res.mx.Unlock()
	if res.deleted {
		return fmt.Errorf("cannot delete %q; already deleted", name)
	}
	if err := r.deleteFn(res); err != nil {
		return err
	}
	res.deleted = true
	return nil
}

func (r *baseResourceRegistry) get(name string) (*resource, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()
	res, ok := r.m[name]
	return res, ok
}

func resourceExists(client compute.Client, url string) (bool, error) {
	if !strings.HasPrefix(url, "projects/") {
		return false, fmt.Errorf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
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
	return false, fmt.Errorf("unknown resource type: %q", url)
}

func (r *baseResourceRegistry) registerCreation(name string, res *resource, s *Step, overWrite bool) error {
	// Create a resource reference, known by name. Check:
	// - no duplicates known by name
	r.mx.Lock()
	defer r.mx.Unlock()
	if res, ok := r.m[name]; ok {
		return fmt.Errorf("cannot create %s %q; already created by step %q", r.typeName, name, res.creator.name)
	}

	if !overWrite {
		if exists, err := resourceExists(r.w.ComputeClient, res.link); err != nil {
			return fmt.Errorf("cannot create %s %q; resource lookup error: %v", r.typeName, name, err)
		} else if exists {
			return fmt.Errorf("cannot create %s %q; resource already exists", r.typeName, name)
		}
	}

	res.creator = s
	r.m[name] = res
	return nil
}

func (r *baseResourceRegistry) registerDeletion(name string, s *Step) error {
	// Mark a resource reference for deletion. Check:
	// - don't dupe deletion of name.
	// - s depends on ALL registered users and creator of name.
	r.mx.Lock()
	defer r.mx.Unlock()
	var ok bool
	var res *resource
	if r.urlRgx != nil && r.urlRgx.MatchString(name) {
		var err error
		res, err = r.registerExisting(name)
		if err != nil {
			return err
		}
	} else if res, ok = r.m[name]; !ok {
		return fmt.Errorf("missing reference for %s %q", r.typeName, name)
	}

	if res.deleter != nil {
		return fmt.Errorf("cannot delete %s %q: already deleted by step %q", r.typeName, name, res.deleter.name)
	}
	us := res.users
	if res.creator != nil {
		us = append(us, res.creator)
	}
	for _, u := range us {
		if !s.nestedDepends(u) {
			return fmt.Errorf("deleting %s %q MUST transitively depend on step %q which references %q", r.typeName, name, u.name, name)
		}
	}
	res.deleter = s
	return nil
}

func (r *baseResourceRegistry) registerExisting(url string) (*resource, error) {
	if !strings.HasPrefix(url, "projects/") {
		return nil, fmt.Errorf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	if r, ok := r.m[url]; ok {
		return r, nil
	}
	if exists, err := resourceExists(r.w.ComputeClient, url); err != nil {
		return nil, err
	} else if !exists {
		return nil, typedErrorf(resourceDNEError, "%s does not exist", url)
	}

	parts := strings.Split(url, "/")
	res := &resource{real: parts[len(parts)-1], link: url, noCleanup: true}
	r.m[url] = res
	return res, nil
}

func (r *baseResourceRegistry) registerUsage(name string, s *Step) (*resource, error) {
	// Make s a user of name. Check:
	// - s depends on creator of name, if there is a creator.
	// - name doesn't have a registered deleter yet, usage must occur before deletion.
	r.mx.Lock()
	defer r.mx.Unlock()
	var ok bool
	var res *resource
	if r.urlRgx != nil && r.urlRgx.MatchString(name) {
		var err error
		res, err = r.registerExisting(name)
		if err != nil {
			return nil, err
		}
	} else if res, ok = r.m[name]; !ok {
		return nil, fmt.Errorf("missing reference for %s %q", r.typeName, name)
	}

	if res.creator != nil && !s.nestedDepends(res.creator) {
		return nil, fmt.Errorf("using %s %q MUST transitively depend on step %q which creates %q", r.typeName, name, res.creator.name, name)
	}
	if res.deleter != nil {
		return nil, fmt.Errorf("using %s %q; step %q deletes %q and MUST transitively depend on this step", r.typeName, name, res.deleter.name, name)
	}

	r.m[name].users = append(r.m[name].users, s)
	return res, nil
}

func initWorkflowResources(w *Workflow) {
	initDiskRegistry(w)
	initImageRegistry(w)
	initInstanceRegistry(w)
	initNetworkRegistry(w)
	w.addCleanupHook(resourceCleanupHook(w))
}

func shareWorkflowResources(giver, taker *Workflow) {
	disksMu.Lock()
	disks[taker] = disks[giver]
	disksMu.Unlock()
	imagesMu.Lock()
	images[taker] = images[giver]
	imagesMu.Unlock()
	instancesMu.Lock()
	instances[taker] = instances[giver]
	instancesMu.Unlock()
	networksMu.Lock()
	networks[taker] = networks[giver]
	networksMu.Unlock()
}

func resourceCleanupHook(w *Workflow) func() error {
	return func() error {
		images[w].cleanup()
		instances[w].cleanup()
		disks[w].cleanup()
		return nil
	}
}

func extendPartialURL(url, project string) string {
	if strings.HasPrefix(url, "projects") {
		return url
	}
	return fmt.Sprintf("projects/%s/%s", project, url)
}

func resourceNameHelper(name string, w *Workflow, exactName bool) string {
	if !exactName {
		name = w.genName(name)
	}
	return name
}

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

	"google.golang.org/api/googleapi"
)

type resource struct {
	real, link         string
	noCleanup, deleted bool

	creator, deleter *Step
	users            []*Step
}

type resourceMap interface {
	cleanup() error
	delete(name string) error
	get(name string) (*resource, bool)
	registerCreation(name string, r *resource, s *Step) error
	registerDeletion(name string, s *Step) error
	registerUsage(name string, s *Step) error
}

type baseResourceMap struct {
	w  *Workflow
	m  map[string]*resource
	mx sync.Mutex

	deleteFn func(r *resource) error
	typeName string
	urlRgx   *regexp.Regexp
}

func (rm *baseResourceMap) cleanup() {
	var wg sync.WaitGroup
	for name, r := range rm.m {
		if r.noCleanup || r.deleted {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := rm.delete(name); err != nil {
				if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code != 404 {
					fmt.Println(err)
				}
			}
		}(name)
	}
	wg.Wait()
}

func (rm *baseResourceMap) delete(name string) error {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if r, ok := rm.m[name]; ok {
		if r.deleted {
			return fmt.Errorf("cannot delete %q; already deleted", name)
		}
		if err := rm.deleteFn(r); err != nil {
			return err
		}
		r.deleted = true
		return nil
	} else {
		return fmt.Errorf("cannot delete %q; does not exist in resource map", name)
	}
}

func (rm *baseResourceMap) get(name string) (*resource, bool) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	r, ok := rm.m[name]
	return r, ok
}

func (rm *baseResourceMap) registerCreation(name string, r *resource, s *Step) error {
	// Create a resource reference, known by name. Check:
	// - no duplicates known by name
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m == nil {
		rm.m = map[string]*resource{}
	} else if r, ok := rm.m[name]; ok {
		return fmt.Errorf("cannot create %s %q; already created by step %q", rm.typeName, name, r.creator.name)
	}
	r.creator = s
	rm.m[name] = r
	return nil
}

func (rm *baseResourceMap) registerDeletion(name string, s *Step) error {
	// Mark a resource reference for deletion. Check:
	// - don't dupe deletion of name.
	// - s depends on ALL registered users and creator of name.
	rm.mx.Lock()
	defer rm.mx.Unlock()
	var ok bool
	var r *resource
	if rm.urlRgx != nil && rm.urlRgx.MatchString(name) {
		var err error
		r, err = rm.registerExisting(name)
		if err != nil {
			return err
		}
	} else if r, ok = rm.m[name]; !ok {
		return fmt.Errorf("missing reference for %s %q", rm.typeName, name)
	}

	if r.deleter != nil {
		return fmt.Errorf("cannot delete %s %q: already deleted by step %q", rm.typeName, name, r.deleter.name)
	}
	us := r.users
	if r.creator != nil {
		us = append(us, r.creator)
	}
	for _, u := range us {
		if !s.nestedDepends(u) {
			return fmt.Errorf("deleting %s %q MUST transitively depend on step %q which references %q", rm.typeName, name, u.name, name)
		}
	}
	r.deleter = s
	return nil
}

func (rm *baseResourceMap) registerExisting(url string) (*resource, error) {
	if !strings.HasPrefix(url, "projects/") {
		return nil, fmt.Errorf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	if rm.m == nil {
		rm.m = map[string]*resource{}
	} else if r, ok := rm.m[url]; ok {
		return r, nil
	}
	parts := strings.Split(url, "/")
	r := &resource{real: parts[len(parts)-1], link: url, noCleanup: true}
	rm.m[url] = r
	return r, nil
}

func (rm *baseResourceMap) registerUsage(name string, s *Step) (*resource, error) {
	// Make s a user of name. Check:
	// - s depends on creator of name, if there is a creator.
	// - name doesn't have a registered deleter yet, usage must occur before deletion.
	rm.mx.Lock()
	defer rm.mx.Unlock()
	var ok bool
	var r *resource
	if rm.urlRgx != nil && rm.urlRgx.MatchString(name) {
		var err error
		r, err = rm.registerExisting(name)
		if err != nil {
			return nil, err
		}
	} else if r, ok = rm.m[name]; !ok {
		return nil, fmt.Errorf("missing reference for %s %q", rm.typeName, name)
	}

	if r.creator != nil && !s.nestedDepends(r.creator) {
		return nil, fmt.Errorf("using %s %q MUST transitively depend on step %q which creates %q", rm.typeName, name, r.creator.name, name)
	}
	if r.deleter != nil {
		return nil, fmt.Errorf("using %s %q; step %q deletes %q and MUST transitively depend on this step", rm.typeName, name, r.deleter.name, name)
	}

	rm.m[name].users = append(rm.m[name].users, s)
	return r, nil
}

func initWorkflowResources(w *Workflow) {
	initDiskMap(w)
	initImageMap(w)
	initInstanceMap(w)
	w.addCleanupHook(resourceCleanupHook(w))
}

func shareWorkflowResources(giver, taker *Workflow) {
	disks[taker] = disks[giver]
	images[taker] = images[giver]
	instances[taker] = instances[giver]
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

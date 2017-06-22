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

package workflow

import (
	"fmt"
	"sync"
)

type resource struct {
	real, link         string
	noCleanup, deleted bool

	creator, deleter *Step
	users            []*Step
}

type resourceMap struct {
	typeName string
	m        map[string]*resource
	mx       sync.Mutex

	usageRegistrationHook func(name string, s *Step) error
}

func (rm *resourceMap) add(name string, r *resource) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	if rm.m == nil {
		rm.m = map[string]*resource{}
	}
	rm.m[name] = r
}

func (rm *resourceMap) del(name string) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	delete(rm.m, name)
}

func (rm *resourceMap) get(name string) (*resource, bool) {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	r, ok := rm.m[name]
	return r, ok
}

func (rm *resourceMap) registerCreation(name string, r *resource, s *Step) error {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	// Check: don't create a dupe resource.
	if rm.m == nil {
		rm.m = map[string]*resource{}
	} else if r, ok := rm.m[name]; ok {
		return fmt.Errorf("cannot create %s %q; already created by step %q", rm.typeName, name, r.creator.name)
	}
	r.creator = s
	rm.m[name] = r
	return nil
}

func (rm *resourceMap) registerDeletion(name string, s *Step) error {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	// Check: don't dupe deletion of a resource.
	// Check: delete depends on ALL registered usages and creation of resource.
	r, ok := rm.m[name]
	if ok && r.creator != nil {
		if r.deleter != nil {
			return fmt.Errorf("cannot delete %s %q: already deleted by step %q", rm.typeName, name, r.deleter.name)
		}
		for _, u := range append(r.users, r.creator) {
			if !s.depends(u) {
				return fmt.Errorf("deleting %s %q MUST transitively depend on step %q which references %q", rm.typeName, name, u.name, name)
			}
		}
	} else {
		return fmt.Errorf("missing reference for %s %q", rm.typeName, name)
	}
	r.deleter = s
	return nil
}

func (rm *resourceMap) registerUsage(name string, s *Step) error {
	rm.mx.Lock()
	defer rm.mx.Unlock()
	// Check: usage depends on creation of resource.
	// Check: there shouldn't be a deleter yet, usage occurs before deletion.
	r, ok := rm.m[name]
	if ok && r.creator != nil {
		if !s.depends(r.creator) {
			return fmt.Errorf("using %s %q MUST transitively depend on step %q which creates %q", rm.typeName, name, r.creator.name, name)
		}
		if r.deleter != nil {
			return fmt.Errorf("using %s %q; step %q deletes %q and MUST transitively depend on this step", rm.typeName, name, r.deleter.name, name)
		}
		if err := rm.usageRegistrationHook(name, s); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing reference for %s %q", rm.typeName, name)
	}
	rm.m[name].users = append(rm.m[name].users, s)
	return nil
}

func initWorkflowResources(w *Workflow) {
	initDisksMap(w)
	initImagesMap(w)
	initInstancesMap(w)
	w.addCleanupHook(resourceCleanupHook(w))
}

func shareWorkflowResources(giver, taker *Workflow) {
	disks[taker] = disks[giver]
	images[taker] = images[giver]
	instances[taker] = instances[giver]
}

func resourceCleanupHook(w *Workflow) func() error {
	return func() error {
		resourceCleanupHelper(images[w], func(r *resource) error { return deleteImage(w, r) })
		resourceCleanupHelper(instances[w], func(r *resource) error { return deleteInstance(w, r) })
		resourceCleanupHelper(disks[w], func(r *resource) error { return deleteDisk(w, r) })
		return nil
	}
}

func resourceCleanupHelper(rm *resourceMap, deleteFn func(*resource) error) {
	var wg sync.WaitGroup
	for name, r := range rm.m {
		if r.noCleanup || r.deleted {
			continue
		}
		wg.Add(1)
		go func(ref string, res *resource) {
			defer wg.Done()
			if err := deleteFn(res); err != nil {
				fmt.Println(err)
			}
		}(name, r)
	}
	wg.Wait()
}

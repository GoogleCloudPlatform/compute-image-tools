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
)

type baseResourceRegistry struct {
	w  *Workflow
	m  map[string]*Resource
	mx sync.Mutex

	deleteFn func(res *Resource) DError
	startFn  func(res *Resource) DError
	stopFn   func(res *Resource) DError
	typeName string
	urlRgx   *regexp.Regexp
}

func (r *baseResourceRegistry) init() {
	r.m = map[string]*Resource{}
}

func (r *baseResourceRegistry) cleanup() {
	var wg sync.WaitGroup
	for name, res := range r.m {
		if res.creator == nil || // placeholder resource
			(res.creator != nil && !res.createdInWorkflow) || // resource isn‘t created successfully
			(res.NoCleanup && !r.w.forceCleanup) || // resource is flagged to avoid cleanup
			res.deleted { // resource has been deleted
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := r.delete(name); err != nil && err.etype() != resourceDNEError {
				fmt.Println(err)
			}
		}(name)
	}
	wg.Wait()
}

func (r *baseResourceRegistry) delete(name string) DError {
	res, ok := r.get(name)
	if !ok {
		return Errf("cannot delete %s %q; does not exist in registry", r.typeName, name)
	}

	// TODO: find a better way for resource delete locking.
	// - move deleteMx out of Resource, it probably belongs in the registry.
	r.mx.Lock()
	if res.deleteMx == nil {
		res.deleteMx = &sync.Mutex{}
	}
	r.mx.Unlock()

	res.deleteMx.Lock()
	defer res.deleteMx.Unlock()
	if res.deleted {
		return Errf("cannot delete %q; already deleted", name)
	}
	if err := r.deleteFn(res); err != nil {
		return err
	}
	res.deleted = true
	return nil
}

func (r *baseResourceRegistry) start(name string) DError {
	res, ok := r.get(name)
	if !ok {
		return Errf("cannot start %s %q; does not exist in registry", r.typeName, name)
	}

	if res.startedByWf {
		return Errf("cannot start %q; already started", name)
	}
	if err := r.startFn(res); err != nil {
		return err
	}
	res.stoppedByWf = false
	res.startedByWf = true
	return nil
}

func (r *baseResourceRegistry) stop(name string) DError {
	res, ok := r.get(name)
	if !ok {
		return Errf("cannot stop %s %q; does not exist in registry", r.typeName, name)
	}

	if res.stoppedByWf {
		return Errf("cannot stop %q; already stopped", name)
	}
	if err := r.stopFn(res); err != nil {
		return err
	}
	res.startedByWf = false
	res.stoppedByWf = true
	return nil
}

func (r *baseResourceRegistry) get(name string) (*Resource, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()
	res, ok := r.m[name]
	return res, ok
}

// regCreate registers a Step s as the creator of a resource, res, and identifies the resource by name.
func (r *baseResourceRegistry) regCreate(name string, res *Resource, s *Step, overWrite bool) DError {
	// Check:
	// - no duplicates known by name
	r.mx.Lock()
	defer r.mx.Unlock()
	if res, ok := r.m[name]; ok {
		return Errf("cannot create %s %q; already created by step %q", r.typeName, name, res.creator.name)
	}

	if !overWrite {
		if exists, err := r.w.resourceExists(res.link); err != nil {
			return Errf("cannot create %s %q; resource lookup error: %v", r.typeName, name, err)
		} else if exists {
			return Errf("cannot create %s %q; resource already exists", r.typeName, name)
		}
	}

	res.creator = s
	r.m[name] = res
	return nil
}

// regDelete registers a Step s as the deleter of a resource.
// The name argument can be a Daisy internal name, or a fully qualified resource URL, e.g. projects/p/global/images/i.
func (r *baseResourceRegistry) regDelete(name string, s *Step) DError {
	// Check:
	// - don't dupe deletion of name.
	// - s depends on ALL registered users and creator of name.
	r.mx.Lock()
	defer r.mx.Unlock()
	var ok bool
	var res *Resource
	if r.urlRgx != nil && r.urlRgx.MatchString(name) {
		var err DError
		res, err = r.regURL(name, true)
		if err != nil {
			return err
		}
	} else if res, ok = r.m[name]; !ok {
		return Errf("missing reference for %s %q", r.typeName, name)
	}

	if res.deleter != nil {
		return Errf("cannot delete %s %q: already deleted by step %q", r.typeName, name, res.deleter.name)
	}
	us := res.users
	if res.creator != nil {
		us = append(us, res.creator)
	}
	for _, u := range us {
		if !s.nestedDepends(u) {
			return Errf("deleting %s %q MUST transitively depend on step %q which references %q", r.typeName, name, u.name, name)
		}
	}
	res.deleter = s
	return nil
}

// regURL creates a placeholder registry entry for a resource identified by a fully qualified resource URL, e.g.
// projects/p/global/images/i.
// A placeholder resource will be created in the registry. The resource will have no creator and will not auto-cleanup.
// The placeholder resource will be identified within the registry by its fully qualified resource URL.
func (r *baseResourceRegistry) regURL(url string, checkExist bool) (*Resource, DError) {
	if !strings.HasPrefix(url, "projects/") {
		return nil, Errf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	if r, ok := r.m[url]; ok {
		return r, nil
	}
	if checkExist {
		exists, err := r.w.resourceExists(url)
		if !exists {
			if err != nil {
				return nil, err
			}
			return nil, typedErrf(r.typeName+resourceDNEError, "%s does not exist", url)
		}
	}

	parts := strings.Split(url, "/")
	res := &Resource{RealName: parts[len(parts)-1], link: url, NoCleanup: true}
	r.m[url] = res
	return res, nil
}

// regUse registers a Step s as a user of a resource.
// The name argument can be a Daisy internal name, or a fully qualified resource URL, e.g. projects/p/global/images/i.
func (r *baseResourceRegistry) regUse(name string, s *Step) (*Resource, DError) {
	// Check:
	// - s depends on creator of name, if there is a creator.
	// - name doesn't have a registered deleter yet, usage must occur before deletion.
	r.mx.Lock()
	defer r.mx.Unlock()
	var ok bool
	var res *Resource
	if r.urlRgx != nil && r.urlRgx.MatchString(name) {
		var err DError
		res, err = r.regURL(name, true)
		if err != nil {
			return nil, err
		}
	} else if res, ok = r.m[name]; !ok {
		return nil, Errf("missing reference for %s %q", r.typeName, name)
	}

	if res.creator != nil && !s.nestedDepends(res.creator) {
		return nil, Errf("using %s %q MUST transitively depend on step %q which creates %q", r.typeName, name, res.creator.name, name)
	}
	if res.deleter != nil {
		return nil, Errf("using %s %q; step %q deletes %q and MUST transitively depend on this step", r.typeName, name, res.deleter.name, name)
	}

	r.m[name].users = append(r.m[name].users, s)
	return res, nil
}

// regUseDeviceName registers a Step s as a user of a disk device resource.
// "DeviceName" is only used by DetachDisks API.
// "daisyInstanceName" represents the name in daisy workflow definition, which is a shorter name
func (dr *diskRegistry) regUseDeviceName(deviceName, project, zone, instance, daisyInstanceName string, s *Step) (*Resource, bool, DError) {
	// Check:
	// deviceName either has a creator/attacher, or has been attached before the workflow's execution
	// - s depends on creator of deviceName, if there is a creator.
	// - deviceName doesn't have a registered deleter yet, usage must occur before deletion.
	dr.mx.Lock()
	defer dr.mx.Unlock()
	var isAttached bool
	var res *Resource
	var err DError

	if deviceNameURLRgx.MatchString(deviceName) {
		// check whether it's attached before the workflow's execution
		isAttached, err = isDiskAttached(dr.w.ComputeClient, deviceName, project, zone, instance)
		if err != nil {
			return nil, isAttached, err
		}
		if !isAttached {
			return nil, isAttached, Errf("device name '%v' is not attached", deviceName)
		}
		res, err = dr.regURL(deviceName, false)
		if err != nil {
			return nil, isAttached, err
		}
	} else if strings.Contains(deviceName, "/") {
		return nil, isAttached, Errf("unexpected url for %s: %q", dr.typeName, deviceName)
	} else if res, err = dr.findDiskResourceByDeviceName(deviceName, daisyInstanceName); err != nil {
		return nil, isAttached, err
	}

	if res.creator != nil && !s.nestedDepends(res.creator) {
		return nil, isAttached, Errf("using %s %q MUST transitively depend on step %q which creates %q", dr.typeName, deviceName, res.creator.name, deviceName)
	}
	if res.deleter != nil {
		return nil, isAttached, Errf("using %s %q; step %q deletes %q and MUST transitively depend on this step", dr.typeName, deviceName, res.deleter.name, deviceName)
	}

	res.users = append(res.users, s)
	return res, isAttached, nil
}

func (dr *diskRegistry) findDiskResourceByDeviceName(deviceName, instance string) (*Resource, DError) {
	attachmentMap, ok := dr.attachments[deviceName]
	if !ok {
		return nil, Errf("missing registered disk attachment for device name '%v'", deviceName)
	}
	attachment, ok := attachmentMap[instance]
	if !ok {
		return nil, Errf("missing registered disk attachment for instance '%v'", instance)
	}
	return dr.m[attachment.diskName], nil
}

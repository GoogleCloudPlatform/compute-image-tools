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

	deleteFn func(res *Resource) dErr
	stopFn   func(res *Resource) dErr
	typeName string
	urlRgx   *regexp.Regexp
}

func (r *baseResourceRegistry) init() {
	r.m = map[string]*Resource{}
}

func (r *baseResourceRegistry) cleanup() {
	var wg sync.WaitGroup
	for name, res := range r.m {
		if res.NoCleanup || res.deleted {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := r.delete(name); err != nil && err.Type() != resourceDNEError {
				fmt.Println(err)
			}
		}(name)
	}
	wg.Wait()
}

func (r *baseResourceRegistry) delete(name string) dErr {
	res, ok := r.get(name)
	if !ok {
		return errf("cannot delete %s %q; does not exist in registry", r.typeName, name)
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
		return errf("cannot delete %q; already deleted", name)
	}
	if err := r.deleteFn(res); err != nil {
		return err
	}
	res.deleted = true
	return nil
}

func (r *baseResourceRegistry) stop(name string) dErr {
	res, ok := r.get(name)
	if !ok {
		return errf("cannot stop %s %q; does not exist in registry", r.typeName, name)
	}

	if res.stopped {
		return errf("cannot stop %q; already stopped", name)
	}
	if err := r.stopFn(res); err != nil {
		return err
	}
	res.stopped = true
	return nil
}

func (r *baseResourceRegistry) get(name string) (*Resource, bool) {
	r.mx.Lock()
	defer r.mx.Unlock()
	res, ok := r.m[name]
	return res, ok
}

// regCreate registers a Step s as the creator of a resource, res, and identifies the resource by name.
func (r *baseResourceRegistry) regCreate(name string, res *Resource, s *Step, overWrite bool) dErr {
	// Check:
	// - no duplicates known by name
	r.mx.Lock()
	defer r.mx.Unlock()
	if res, ok := r.m[name]; ok {
		return errf("cannot create %s %q; already created by step %q", r.typeName, name, res.creator.name)
	}

	if !overWrite {
		if exists, err := resourceExists(r.w.ComputeClient, res.link); err != nil {
			return errf("cannot create %s %q; resource lookup error: %v", r.typeName, name, err)
		} else if exists {
			return errf("cannot create %s %q; resource already exists", r.typeName, name)
		}
	}

	res.creator = s
	r.m[name] = res
	return nil
}

// regDelete registers a Step s as the deleter of a resource.
// The name argument can be a Daisy internal name, or a fully qualified resource URL, e.g. projects/p/global/images/i.
func (r *baseResourceRegistry) regDelete(name string, s *Step) dErr {
	// Check:
	// - don't dupe deletion of name.
	// - s depends on ALL registered users and creator of name.
	r.mx.Lock()
	defer r.mx.Unlock()
	var ok bool
	var res *Resource
	if r.urlRgx != nil && r.urlRgx.MatchString(name) {
		var err dErr
		res, err = r.regURL(name)
		if err != nil {
			return err
		}
	} else if res, ok = r.m[name]; !ok {
		return errf("missing reference for %s %q", r.typeName, name)
	}

	if res.deleter != nil {
		return errf("cannot delete %s %q: already deleted by step %q", r.typeName, name, res.deleter.name)
	}
	us := res.users
	if res.creator != nil {
		us = append(us, res.creator)
	}
	for _, u := range us {
		if !s.nestedDepends(u) {
			return errf("deleting %s %q MUST transitively depend on step %q which references %q", r.typeName, name, u.name, name)
		}
	}
	res.deleter = s
	return nil
}

// regURL creates a placeholder registry entry for a resource identified by a fully qualified resource URL, e.g.
// projects/p/global/images/i.
// A placeholder resource will be created in the registry. The resource will have no creator and will not auto-cleanup.
// The placeholder resource will be identified within the registry by its fully qualified resource URL.
func (r *baseResourceRegistry) regURL(url string) (*Resource, dErr) {
	if !strings.HasPrefix(url, "projects/") {
		return nil, errf("partial GCE resource URL %q needs leading \"projects/PROJECT/\"", url)
	}
	if r, ok := r.m[url]; ok {
		return r, nil
	}
	exists, err := resourceExists(r.w.ComputeClient, url)
	if !exists {
		if err != nil {
			return nil, err
		}
		return nil, typedErrf(resourceDNEError, "%s does not exist", url)
	}

	parts := strings.Split(url, "/")
	res := &Resource{RealName: parts[len(parts)-1], link: url, NoCleanup: true}
	r.m[url] = res
	return res, err
}

// regUse registers a Step s as a user of a resource.
// The name argument can be a Daisy internal name, or a fully qualified resource URL, e.g. projects/p/global/images/i.
func (r *baseResourceRegistry) regUse(name string, s *Step) (*Resource, dErr) {
	// Check:
	// - s depends on creator of name, if there is a creator.
	// - name doesn't have a registered deleter yet, usage must occur before deletion.
	r.mx.Lock()
	defer r.mx.Unlock()
	var ok bool
	var res *Resource
	if r.urlRgx != nil && r.urlRgx.MatchString(name) {
		var err dErr
		res, err = r.regURL(name)
		if err != nil {
			return nil, err
		}
	} else if res, ok = r.m[name]; !ok {
		return nil, errf("missing reference for %s %q", r.typeName, name)
	}

	if res.creator != nil && !s.nestedDepends(res.creator) {
		return nil, errf("using %s %q MUST transitively depend on step %q which creates %q", r.typeName, name, res.creator.name, name)
	}
	if res.deleter != nil {
		return nil, errf("using %s %q; step %q deletes %q and MUST transitively depend on this step", r.typeName, name, res.deleter.name, name)
	}

	r.m[name].users = append(r.m[name].users, s)
	return res, nil
}

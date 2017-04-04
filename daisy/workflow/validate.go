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
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

const defaultTimeout = "10m"
const defaultTimeoutParsed = time.Duration(10 * time.Minute)

type nameSet map[*Workflow]*[]string

var (
	// The steps DAG is inspected in DAG order.
	// Given a sane workflow, these will be populated as they would occur
	// as the workflow runs. This allows us to check, ahead of time, if
	// resources exist.
	// If a workflow isn't sane, we can't make guarantees about whether
	// a resource will exist or not. Example:
	//
	//   Step foo creates a disk, step bar uses that disk.
	//   If the workflow is sane, bar should transitively depend on foo.
	//   If the workflow isn't sane, and bar does not depend on foo, the
	//   disk created by foo may or may not exist when bar is run.
	//
	// As with that example, foo may or may not be validated before bar.
	// If foo is validated first, the disk will be listed here.
	// If not, the disk will not be listed here.
	//
	// So, on sane workflows, reference checking will be fine.
	// On insane workflows, validation may or may not catch the disk
	// reference discrepancy.
	validatedDisks             = nameSet{}
	validatedDiskDeletions     = nameSet{}
	validatedImages            = nameSet{}
	validatedInstances         = nameSet{}
	validatedInstanceDeletions = nameSet{}
	nameSetsMx                 = sync.Mutex{}
)

var rfc1035Rgx = regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")

func (n nameSet) add(w *Workflow, s string) error {
	nameSetsMx.Lock()
	defer nameSetsMx.Unlock()
	ss := n.getNames(w)

	// Name checking first.
	if containsString(s, *ss) {
		return fmt.Errorf("workflow %q has duplicate references for %q", w.Name, s)
	}
	if !checkName(s) {
		return fmt.Errorf("bad name %q", s)
	}

	*ss = append(*ss, s)
	return nil
}

func (n nameSet) getNames(w *Workflow) *[]string {
	if ss, ok := n[w]; ok {
		return ss
	}
	n[w] = &[]string{}
	return n[w]
}

func (n nameSet) has(w *Workflow, s string) bool {
	nameSetsMx.Lock()
	defer nameSetsMx.Unlock()
	return containsString(s, *n.getNames(w))
}

func checkName(s string) bool {
	return len(s) < 64 && rfc1035Rgx.MatchString(s)
}

func diskValid(w *Workflow, d string) bool {
	if !validatedDisks.has(w, d) {
		return false
	}
	if validatedDiskDeletions.has(w, d) {
		return false
	}
	return true
}

func imageValid(w *Workflow, i string) bool {
	// TODO(crunkleton): better checking for resource names pointing to GCE resources.
	return validatedImages.has(w, i) || strings.HasPrefix(i, "projects/")
}

func instanceValid(w *Workflow, i string) bool {
	if !validatedInstances.has(w, i) {
		return false
	}
	if validatedInstanceDeletions.has(w, i) {
		return false
	}
	return true
}

func projectExists(p string) bool {
	// TODO(crunkleton)
	return true
}

func sourceExists(s string) bool {
	// TODO(crunkleton)
	return true
}

func (w *Workflow) validate() error {
	if w.Name == "" {
		return errors.New("must provide workflow field 'name'")
	}
	if w.Project == "" {
		return errors.New("must provide workflow field 'project'")
	}
	if w.Zone == "" {
		return errors.New("must provide workflow field 'zone'")
	}
	if w.GCSPath == "" {
		return errors.New("must provide workflow field 'gcs_path'")
	}
	if len(w.Steps) == 0 {
		return errors.New("must provide at least one step in workflow field 'steps'")
	}
	for name, step := range w.Steps {
		if name == "" {
			return fmt.Errorf("no name defined for step %q", name)
		}
		if step.Timeout == "" {
			return fmt.Errorf("no timeout defined for step %q", name)
		}
	}

	// Check for unsubstituted vars.
	if err := w.validateVarsSubbed(); err != nil {
		return err
	}

	return w.validateDAG()
}

// Step through the step DAG, calling each step's validate().
func (w *Workflow) validateDAG() error {
	// Sanitation.
	for s, deps := range w.Dependencies {
		seen := map[string]bool{}
		var clean []string
		for _, dep := range deps {
			// Check for missing dependencies.
			if _, ok := w.Steps[dep]; !ok {
				return fmt.Errorf("missing reference for dependency %s", dep)
			}
			// Remove duplicate dependencies.
			if !seen[dep] {
				seen[dep] = true
				clean = append(clean, dep)
			}
		}
		w.Dependencies[s] = clean
	}

	// Check for cycles.
	for s := range w.Steps {
		if w.stepDepends(s, s) {
			return fmt.Errorf("cyclic dependency on step %s", s)
		}
	}
	return w.traverseDAG(func(s step) error { return s.validate(w) })
}

func (w *Workflow) validateVarsSubbed() error {
	unsubbedVarRgx := regexp.MustCompile(`\$\{[^}]+}`)
	return traverseDataStructure(reflect.ValueOf(w).Elem(), func(v reflect.Value) error {
		switch v.Interface().(type) {
		case string:
			if unsubbedVarRgx.MatchString(v.String()) {
				return fmt.Errorf("Unresolved var found in %q", v.String())
			}
		}
		return nil
	})
}

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
)

type nameSet map[*Workflow][]string

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
	validatedImageDeletions    = nameSet{}
	nameSetsMx                 = sync.Mutex{}
	rfc1035                    = "[a-z]([-a-z0-9]*[a-z0-9])?"
	rfc1035Rgx                 = regexp.MustCompile(fmt.Sprintf("^%s$", rfc1035))
)

func (n nameSet) add(w *Workflow, s string) error {
	nameSetsMx.Lock()
	defer nameSetsMx.Unlock()
	ss := n.getNames(w)

	// Name checking first.
	if containsString(s, ss) {
		return fmt.Errorf("workflow %q has duplicate references for %q", w.Name, s)
	}
	if !checkName(s) {
		return fmt.Errorf("bad name %q", s)
	}

	n[w] = append(ss, s)
	return nil
}

func (n nameSet) getNames(w *Workflow) []string {
	if ss, ok := n[w]; ok {
		return ss
	}
	n[w] = []string{}
	return n[w]
}

func (n nameSet) has(w *Workflow, s string) bool {
	nameSetsMx.Lock()
	defer nameSetsMx.Unlock()
	return containsString(s, n.getNames(w))
}

func checkName(s string) bool {
	return len(s) < 64 && rfc1035Rgx.MatchString(s)
}

func diskValid(w *Workflow, d string) bool {
	if validatedDiskDeletions.has(w, d) {
		return false
	}
	if diskURLRegex.MatchString(d) {
		return true
	}
	if !validatedDisks.has(w, d) {
		return false
	}
	return true
}

func imageValid(w *Workflow, i string) bool {
	if validatedImageDeletions.has(w, i) {
		return false
	}
	if imageURLRegex.MatchString(i) {
		return true
	}
	if !validatedImages.has(w, i) {
		return false
	}
	return true
}

func instanceValid(w *Workflow, i string) bool {
	if validatedInstanceDeletions.has(w, i) {
		return false
	}
	if instanceURLRegex.MatchString(i) {
		return true
	}
	if !validatedInstances.has(w, i) {
		return false
	}
	return true
}

func projectExists(p string) bool {
	// TODO(crunkleton)
	return true
}

func (w *Workflow) validateRequiredFields() error {
	if w.Name == "" {
		return errors.New("must provide workflow field 'Name'")
	}
	if !rfc1035Rgx.MatchString(strings.ToLower(w.Name)) {
		return errors.New("workflow field 'Name' must start with a letter and only contain letters, numbers, and hyphens")
	}
	if w.Project == "" {
		return errors.New("must provide workflow field 'Project'")
	}
	if w.Zone == "" {
		return errors.New("must provide workflow field 'Zone'")
	}
	if w.GCSPath == "" {
		return errors.New("must provide workflow field 'GCSPath'")
	}
	if len(w.Steps) == 0 {
		return errors.New("must provide at least one step in workflow field 'Steps'")
	}
	for name := range w.Steps {
		if name == "" {
			return fmt.Errorf("no name defined for Step %q", name)
		}
	}
	return nil
}

func (w *Workflow) validate() error {
	if err := w.validateRequiredFields(); err != nil {
		return err
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
	for _, s := range w.Steps {
		if s.depends(s) {
			return fmt.Errorf("cyclic dependency on step %v", s)
		}
	}
	return w.traverseDAG(func(s *Step) error { return s.validate(w) })
}

func (w *Workflow) validateVarsSubbed() error {
	unsubbedVarRgx := regexp.MustCompile(`\$\{([^}]+)}`)
	return traverseData(reflect.ValueOf(w).Elem(), func(v reflect.Value) error {
		switch v.Interface().(type) {
		case string:
			if match := unsubbedVarRgx.FindStringSubmatch(v.String()); match != nil {
				if containsString(match[1], w.RequiredVars) {
					return fmt.Errorf("Unresolved required var %q found in %q", match[0], v.String())
				}
				return fmt.Errorf("Unresolved var %q found in %q", match[0], v.String())
			}
		}
		return nil
	})
}

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
	"regexp"
	"strings"
)

const defaultTimeout = "10m"

type nameSet []string

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
	diskNames             nameSet
	diskNamesToDelete     nameSet
	imageNames            nameSet
	instanceNames         nameSet
	instanceNamesToDelete nameSet
)

var rfc1035Rgx = regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")

func (n *nameSet) add(s string) error {
	// Name checking first.
	if n.has(s) {
		return fmt.Errorf("nameSet %s already has %q", *n, s)
	}
	if !checkName(s) {
		return fmt.Errorf("bad name %q", s)
	}

	*n = append(*n, s)
	return nil
}

func (n *nameSet) has(s string) bool {
	return containsString(s, *n)
}

func checkName(s string) bool {
	return len(s) < 64 && rfc1035Rgx.MatchString(s)
}

func diskExists(d string) bool {
	if !diskNames.has(d) {
		return false
	}
	if diskNamesToDelete.has(d) {
		return false
	}
	return true
}

func imageExists(i string) bool {
	// TODO(crunkleton): better checking for resource names pointing to GCE resources.
	return imageNames.has(i) || strings.HasPrefix(i, "projects/")
}

func instanceExists(i string) bool {
	if !instanceNames.has(i) {
		return false
	}
	if instanceNamesToDelete.has(i) {
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
		return fmt.Errorf("must provide workflow field 'project'")
	}
	if w.Zone == "" {
		return fmt.Errorf("must provide workflow field 'zone'")
	}
	if w.Bucket == "" {
		return fmt.Errorf("must provide workflow field 'bucket'")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("must provide at least one step in workflow field 'steps'")
	}
	for name, step := range w.Steps {
		if name == "" {
			return fmt.Errorf("no name defined for step %q", name)
		}
		if step.Timeout == "" {
			return fmt.Errorf("no timeout defined for step %q", name)
		}
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
	return w.traverseDAG(func(s step) error { return s.validate() })
}

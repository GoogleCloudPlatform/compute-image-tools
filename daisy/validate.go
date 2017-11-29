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
	"regexp"
	"strings"
)

var (
	rfc1035       = "[a-z]([-a-z0-9]*[a-z0-9])?"
	projectRgxStr = "[a-z]([-.:a-z0-9]*[a-z0-9])?"
	rfc1035Rgx    = regexp.MustCompile(fmt.Sprintf("^%s$", rfc1035))
)

func checkName(s string) bool {
	return len(s) < 64 && rfc1035Rgx.MatchString(s)
}

func (w *Workflow) validateRequiredFields() dErr {
	if w.Name == "" {
		return errf("must provide workflow field 'Name'")
	}
	if !rfc1035Rgx.MatchString(strings.ToLower(w.Name)) {
		return errf("workflow field 'Name' must start with a letter and only contain letters, numbers, and hyphens")
	}
	if w.Project == "" {
		return errf("must provide workflow field 'Project'")
	}
	if exists, err := projectExists(w.ComputeClient, w.Project); err != nil {
		return errf("bad project lookup: %q, error: %v", w.Project, err)
	} else if !exists {
		return errf("project does not exist: %q", w.Project)
	}
	if w.Zone == "" {
		return errf("must provide workflow field 'Zone'")
	}
	if exists, err := zoneExists(w.ComputeClient, w.Project, w.Zone); err != nil {
		return errf("bad zone lookup: %q, error: %v", w.Zone, err)
	} else if !exists {
		return errf("zone does not exist: %q", w.Zone)
	}
	if len(w.Steps) == 0 {
		return errf("must provide at least one step in workflow field 'Steps'")
	}
	for name := range w.Steps {
		if name == "" {
			return errf("no name defined for Step %q", name)
		}
	}
	return nil
}

func (w *Workflow) validate(ctx context.Context) dErr {
	// Check for unsubstituted wfVar.
	if err := w.validateVarsSubbed(); err != nil {
		return err
	}

	return w.validateDAG(ctx)
}

// Step through the step DAG, calling each step's validate().
func (w *Workflow) validateDAG(ctx context.Context) dErr {
	// Sanitation.
	for s, deps := range w.Dependencies {
		// Check for missing steps.
		if _, ok := w.Steps[s]; !ok {
			return errf("Dependencies reference non existent step %q: %q:%q", s, s, deps)
		}
		seen := map[string]bool{}
		var clean []string
		for _, dep := range deps {
			// Check for missing dependencies.
			if _, ok := w.Steps[dep]; !ok {
				return errf("Dependencies reference non existent step %q: %q:%q", dep, s, deps)
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
			return errf("cyclic dependency on step %v", s)
		}
	}
	return w.traverseDAG(func(s *Step) dErr { return s.validate(ctx) })
}

func (w *Workflow) validateVarsSubbed() dErr {
	unsubbedVarRgx := regexp.MustCompile(`\$\{([^}]+)}`)
	return traverseData(reflect.ValueOf(w).Elem(), func(v reflect.Value) dErr {
		switch v.Interface().(type) {
		case string:
			if match := unsubbedVarRgx.FindStringSubmatch(v.String()); match != nil {
				return errf("Unresolved var %q found in %q", match[0], v.String())
			}
		}
		return nil
	})
}

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
	"reflect"
	"strings"
	"path/filepath"
)

// IncludeWorkflow defines a Daisy workflow injection step. This step will
// 'include' the workflow found the path given into the parent workflow. Unlike
// a Subworkflow the included workflow will exist in the same namespace
// as the parent and have access to all its resources.
type IncludeWorkflow struct {
	Path string
	Vars map[string]string `json:",omitempty"`
	w    *Workflow
}

func (i *IncludeWorkflow) populate(s *Step) error {
	i.w.GCSPath = i.w.parent.GCSPath
	i.w.Name = s.name
	i.w.Project = i.w.parent.Project
	i.w.Zone = i.w.parent.Zone

	for k, v := range i.Vars {
		i.w.AddVar(k, v)
	}
	if err := i.w.populateVars(); err != nil {
		return err
	}

	var replacements []string
	for k, v := range i.w.parent.autovars {
		if k == "NAME" {
			v = s.name
		}
		if k == "WFDIR" {
			v = i.w.workflowDir
		}
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v)
	}
	for k, v := range i.w.vars {
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v.Value)
	}
	substitute(reflect.ValueOf(i.w).Elem(), strings.NewReplacer(replacements...))

	for name, st := range i.w.Steps {
		st.name = name
		st.w = s.w
		if err := s.w.populateStep(st); err != nil {
			return err
		}
	}

	// Copy Sources up to parent resolving relative paths as we go.
	for k, v := range i.w.Sources {
		if _, ok := s.w.Sources[k]; ok {
			return fmt.Errorf("source %q already exists in parent workflow", k)
		}
		if i.w.parent.Sources == nil {
			i.w.parent.Sources = map[string]string{}
		}

		if _, _, err := splitGCSPath(v); err != nil && !filepath.IsAbs(v) {
			v = filepath.Join(i.w.workflowDir, v)
		}
		i.w.parent.Sources[k] = v
	}
	return nil
}

func (i *IncludeWorkflow) validate(s *Step) error {
	return i.w.validate()
}

func (i *IncludeWorkflow) run(s *Step) error {
	return i.w.run()
}

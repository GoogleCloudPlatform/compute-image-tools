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
	"path/filepath"
	"reflect"
	"strings"
)

// IncludeWorkflow defines a Daisy workflow injection step. This step will
// 'include' the workflow found the path given into the parent workflow. Unlike
// a Subworkflow the included workflow will exist in the same namespace
// as the parent and have access to all its resources.
type IncludeWorkflow struct {
	Path     string
	Vars     map[string]string `json:",omitempty"`
	Workflow *Workflow
}

func (i *IncludeWorkflow) populate(ctx context.Context, s *Step) dErr {
	if i.Path != "" {
		var err error
		if i.Workflow, err = s.w.NewIncludedWorkflowFromFile(i.Path); err != nil {
			return newErr(err)
		}
	}

	if i.Workflow == nil {
		return errf("IncludeWorkflow %q does not have a workflow", s.name)
	}

	i.Workflow.id = s.w.id
	i.Workflow.username = s.w.username
	i.Workflow.ComputeClient = s.w.ComputeClient
	i.Workflow.StorageClient = s.w.StorageClient
	i.Workflow.GCSPath = s.w.GCSPath
	i.Workflow.Name = s.name
	i.Workflow.Project = s.w.Project
	i.Workflow.Zone = s.w.Zone
	i.Workflow.autovars = s.w.autovars
	i.Workflow.bucket = s.w.bucket
	i.Workflow.scratchPath = s.w.scratchPath
	i.Workflow.sourcesPath = s.w.sourcesPath
	i.Workflow.logsPath = s.w.logsPath
	i.Workflow.outsPath = s.w.outsPath
	i.Workflow.gcsLogWriter = s.w.gcsLogWriter
	i.Workflow.gcsLogging = s.w.gcsLogging

	for k, v := range i.Vars {
		i.Workflow.AddVar(k, v)
	}

	var replacements []string
	for k, v := range i.Workflow.autovars {
		if k == "NAME" {
			v = s.name
		}
		if k == "WFDIR" {
			v = i.Workflow.workflowDir
		}
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v)
	}
	substitute(reflect.ValueOf(i.Workflow).Elem(), strings.NewReplacer(replacements...))
	for k, v := range i.Workflow.Vars {
		replacements = append(replacements, fmt.Sprintf("${%s}", k), v.Value)
	}
	substitute(reflect.ValueOf(i.Workflow).Elem(), strings.NewReplacer(replacements...))

	i.Workflow.populateLogger(ctx)

	for name, st := range i.Workflow.Steps {
		st.name = name
		st.w = i.Workflow
		if err := st.w.populateStep(ctx, st); err != nil {
			return err
		}
	}

	// Copy Sources up to parent resolving relative paths as we go.
	for k, v := range i.Workflow.Sources {
		if v == "" {
			continue
		}
		if _, ok := s.w.Sources[k]; ok {
			return errf("source %q already exists in workflow", k)
		}
		if s.w.Sources == nil {
			s.w.Sources = map[string]string{}
		}

		if _, _, err := splitGCSPath(v); err != nil && !filepath.IsAbs(v) {
			v = filepath.Join(i.Workflow.workflowDir, v)
		}
		s.w.Sources[k] = v
	}

	return nil
}

func (i *IncludeWorkflow) validate(ctx context.Context, s *Step) dErr {
	return i.Workflow.validate(ctx)
}

func (i *IncludeWorkflow) run(ctx context.Context, s *Step) dErr {
	return i.Workflow.run(ctx)
}

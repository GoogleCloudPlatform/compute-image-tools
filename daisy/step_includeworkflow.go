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
	Workflow *Workflow         `json:",omitempty"`
}

func (i *IncludeWorkflow) populate(ctx context.Context, s *Step) DError {
	if i.Path != "" && i.Workflow == nil {
		var err error
		if i.Workflow, err = s.w.NewIncludedWorkflowFromFile(i.Path); err != nil {
			return newErr("failed to parse duration for step includeworkflow", err)
		}
	} else {
		if i.Workflow == nil {
			return Errf(fmt.Sprintf("IncludeWorkflow %q does not have a workflow", s.name))
		}
		s.w.includeWorkflow(i.Workflow)
	}

	i.Workflow.id = i.Workflow.parent.id
	i.Workflow.username = i.Workflow.parent.username
	i.Workflow.ComputeClient = i.Workflow.parent.ComputeClient
	i.Workflow.StorageClient = i.Workflow.parent.StorageClient
	i.Workflow.cloudLoggingClient = i.Workflow.parent.cloudLoggingClient
	i.Workflow.GCSPath = i.Workflow.parent.GCSPath
	i.Workflow.Name = i.Workflow.parent.Name
	i.Workflow.Project = i.Workflow.parent.Project
	i.Workflow.Zone = i.Workflow.parent.Zone
	i.Workflow.DefaultTimeout = i.Workflow.parent.DefaultTimeout
	i.Workflow.autovars = i.Workflow.parent.autovars
	i.Workflow.bucket = i.Workflow.parent.bucket
	i.Workflow.scratchPath = i.Workflow.parent.scratchPath
	i.Workflow.sourcesPath = i.Workflow.parent.sourcesPath
	i.Workflow.logsPath = i.Workflow.parent.logsPath
	i.Workflow.outsPath = i.Workflow.parent.outsPath
	i.Workflow.externalLogging = i.Workflow.parent.externalLogging
	i.Workflow.Logger = i.Workflow.parent.Logger
	i.Workflow.Name = s.name
	i.Workflow.DefaultTimeout = s.Timeout

	var errs DError
Loop:
	for k, v := range i.Vars {
		for wv := range i.Workflow.Vars {
			if k == wv {
				i.Workflow.AddVar(k, v)
				continue Loop
			}
		}
		errs = addErrs(errs, Errf("unknown workflow Var %q passed to IncludeWorkflow %q", k, s.name))
	}
	if errs != nil {
		return errs
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

	// We do this here, and not in validate, as embedded startup scripts could
	// have what we think are daisy variables.
	if err := i.Workflow.validateVarsSubbed(); err != nil {
		return err
	}

	if err := i.Workflow.substituteSourceVars(ctx, reflect.ValueOf(i.Workflow).Elem()); err != nil {
		return err
	}

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
			return Errf("source %q already exists in workflow", k)
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

func (i *IncludeWorkflow) validate(ctx context.Context, s *Step) DError {
	return i.Workflow.validate(ctx)
}

func (i *IncludeWorkflow) run(ctx context.Context, s *Step) DError {
	return i.Workflow.run(ctx)
}

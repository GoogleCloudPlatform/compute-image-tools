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
	"testing"

	"cloud.google.com/go/storage"
)

func TestCopyGCSObjectsPopulate(t *testing.T) {
	if err := (&CopyGCSObjects{}).populate(&Step{}); err != nil {
		t.Error("not implemented, err should be nil")
	}
}

func TestCopyGCSObjectsRun(t *testing.T) {
	w := testWorkflow()
	s := &Step{w: w}
	w.Steps = map[string]*Step{
		"copy": {CopyGCSObjects: &CopyGCSObjects{{Source: "", Destination: ""}}},
	}

	ws := &CopyGCSObjects{
		{Source: "gs://bucket", Destination: "gs://bucket"},
		{Source: "gs://bucket/object", Destination: "gs://bucket/object"},
		{Source: "gs://bucket/object", Destination: "gs://bucket/object", ACLRules: []storage.ACLRule{{Entity: "allUsers", Role: "OWNER"}}},
	}
	if err := ws.run(s); err != nil {
		t.Errorf("error running CopyGCSObjects.run(): %v", err)
	}

	ws = &CopyGCSObjects{
		{Source: "gs://bucket", Destination: ""},
		{Source: "", Destination: "gs://bucket"},
	}
	if err := ws.run(s); err == nil {
		t.Error("expected error")
	}
}

func TestCopyGCSObjectsValidate(t *testing.T) {
	if err := (&CopyGCSObjects{}).validate(&Step{}); err != nil {
		t.Error("not implemented, err should be nil")
	}
}

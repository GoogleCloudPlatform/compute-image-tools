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

import "testing"

func TestAttachDisksRun(t *testing.T) {}

func TestAttachDisksValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"d-foo"}}
	validatedInstances = nameSet{w: {"i-foo"}}

	// Test normal, good case.
	ad := AttachDisks{AttachDisk{"d-foo", "i-foo"}}
	if err := ad.validate(s); err != nil {
		t.Fatal("validation should not have failed")
	}

	// Test disk DNE.
	ad = AttachDisks{AttachDisk{"d-dne", "i-foo"}}
	if err := ad.validate(s); err == nil {
		t.Fatal("validation should have failed")
	}

	// Test instance DNE.
	ad = AttachDisks{AttachDisk{"d-foo", "i-dne"}}
	if err := ad.validate(s); err == nil {
		t.Fatal("validation should have failed")
	}

	// Test both DNE.
	ad = AttachDisks{AttachDisk{"d-dne", "i-dne"}}
	if err := ad.validate(s); err == nil {
		t.Fatal("validation should have failed")
	}
}

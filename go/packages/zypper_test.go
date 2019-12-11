//  Copyright 2019 Google Inc. All Rights Reserved.
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

package packages

import (
	"errors"
	"testing"
)

func TestZypperInstalls(t *testing.T) {
	run = getMockRun([]byte("TestZypperInstalls"), nil)
	if err := InstallZypperPackages(pkgs); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestZypperInstallsReturnError(t *testing.T) {
	run = getMockRun([]byte("TestZypperInstallsReturnError"), errors.New("Could not find package"))
	if err := InstallZypperPackages(pkgs); err == nil {
		t.Errorf("did not get expected error")
	}
}

func TestRemoveZypper(t *testing.T) {
	run = getMockRun([]byte("TestRemoveZypper"), nil)
	if err := RemoveZypperPackages(pkgs); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveZypperReturnError(t *testing.T) {
	run = getMockRun([]byte("TestRemoveZypperReturnError"), errors.New("Could not find package"))
	if err := RemoveZypperPackages(pkgs); err == nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstallZypperUpdates(t *testing.T) {
	run = getMockRun([]byte("TestInstallZypperUpdates"), nil)
	if err := InstallZypperUpdates(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstallZypperUpdatesReturnError(t *testing.T) {
	run = getMockRun([]byte(""), errors.New("Could not find package"))
	if err := InstallZypperUpdates(); err == nil {
		t.Errorf("did not get expected error")
	}
}

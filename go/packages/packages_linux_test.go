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

package packages

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"
)

func getMockRun(content []byte, err error) runFunc {
	return func(cmd *exec.Cmd) ([]byte, error) {
		return content, err
	}
}

func getMockPackages() []string {
	return []string{"pkg1", "pkg2"}
}

func TestZypperInstalls(t *testing.T) {
	out := "installed successful output"
	run = getMockRun([]byte(out), nil)
	pkgs := getMockPackages()
	actual := InstallZypperPackages(pkgs)
	if actual != nil {
		t.Errorf("unexpected error: %v", actual)
	}
}

func TestZypperInstallsReturnError(t *testing.T) {
	out := ""
	run = getMockRun([]byte(out), errors.New("Could not find package"))
	pkgs := getMockPackages()
	actual := InstallZypperPackages(pkgs)
	if actual == nil {
		t.Errorf("unexpected error: %v", actual)
	}
}

func TestInstalledZypperContainsTwoPackages(t *testing.T) {
	out, err := helperLoadBytes("installedZypperData")
	if err != nil {
		t.Fatal("could not load test data")
	}
	run = getMockRun(out, nil)
	pkgs, err := installedZypper(run)

	if err != nil {
		t.Fatal("could not execxute test")
	}

	if len(pkgs) != 2 {
		t.Errorf("unexpected number of packages")
	}

	fmt.Printf("%s\n%s", pkgs[0], pkgs[1])

	if pkgs[0].Name != "GeoIP-data" || pkgs[1].Name != "SUSEConnect" {
		t.Errorf("Unexpected output")
	}
}

func TestInstalledZypperNoPackagesInstalled(t *testing.T) {
	out, err := helperLoadBytes("installedZypperDataNoZypperPackage")
	run = getMockRun(out, nil)
	pkgs, err := installedZypper(run)

	if pkgs != nil && err != nil {
		t.Errorf("Unexpected output")
	}
}

func TestInstalledZypperOneInvalidPackageEntry(t *testing.T) {
	out, err := helperLoadBytes("installedZypperPackageOneValidEntry")
	if err != nil {
		t.Fatal("could not load test data")
	}
	run = getMockRun(out, nil)
	pkgs, err := installedZypper(run)

	if len(pkgs) != 1 {
		t.Errorf("unexpected number of packages")
	}

	fmt.Printf("%s", pkgs[0])

	if pkgs[0].Name != "GeoIP-data" {
		t.Errorf("Unexpected output")
	}
}

func TestRemoveZypper(t *testing.T) {
	out := "removed successful output"
	run = getMockRun([]byte(out), nil)
	pkgs := getMockPackages()
	actual := RemoveZypperPackages(pkgs)
	if actual != nil {
		t.Errorf("unexpected error: %v", actual)
	}
}

func TestRemoveZypperReturnError(t *testing.T) {
	out := ""
	run = getMockRun([]byte(out), errors.New("Could not find package"))
	pkgs := getMockPackages()
	actual := RemoveZypperPackages(pkgs)
	if actual == nil {
		t.Errorf("unexpected error: %v", actual)
	}
}

func TestInstallZypperUpdates(t *testing.T) {
	out := "removed successful output"
	run = getMockRun([]byte(out), nil)
	actual := InstallZypperUpdates()
	if actual != nil {
		t.Errorf("unexpected error: %v", actual)
	}
}

func TestInstallZypperUpdatesReturnError(t *testing.T) {
	out := ""
	run = getMockRun([]byte(out), errors.New("Could not find package"))
	actual := InstallZypperUpdates()
	if actual == nil {
		t.Errorf("unexpected error: %v", actual)
	}
}

// TODO: move this to a common helper package
func helperLoadBytes(name string) ([]byte, error) {
	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

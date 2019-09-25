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

package guestpolicies

import (
	"fmt"
	"path"
	"strings"
	"time"

	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"

	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha2"
)

const (
	packageInstalled    = "osconfig_tests/pkg_installed"
	packageNotInstalled = "osconfig_tests/pkg_not_installed"
	osconfigTestRepo    = "osconfig-agent-test-repository"
	yumTestRepoBaseURL  = "https://packages.cloud.google.com/yum/repos/osconfig-agent-test-repository"
	aptTestRepoBaseURL  = "http://packages.cloud.google.com/apt"
	gooTestRepoURL      = "https://packages.cloud.google.com/yuck/repos/osconfig-agent-test-repository"
	aptRaptureGpgKey    = "https://packages.cloud.google.com/apt/doc/apt-key.gpg"
)

var (
	yumRaptureGpgKeys = []string{"https://packages.cloud.google.com/yum/doc/yum-key.gpg", "https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg"}
)

func buildPkgInstallTestSetup(name, image, pkgManager, key string) *guestPolicyTestSetup {
	assertTimeout := 60 * time.Second
	testName := packageInstallFunction
	packageName := "cowsay"
	machineType := "n1-standard-2"
	if pkgManager == "googet" {
		machineType = "n1-standard-4"
	}
	if pkgManager == "zypper" {
		packageName = "xeyes"
	}
	if strings.Contains(image, "rhel-8") {
		packageName = "xorg-x11-apps"
	}

	instanceName := fmt.Sprintf("%s-%s-%s-%s", path.Base(name), testName, key, utils.RandString(3))
	gp := &osconfigpb.GuestPolicy{
		Packages:   osconfigserver.BuildPackagePolicy([]string{packageName}, nil, nil),
		Assignment: &osconfigpb.Assignment{InstanceNamePrefixes: []string{instanceName}},
	}
	ss := getStartupScript(name, pkgManager, packageName)
	return newGuestPolicyTestSetup(image, instanceName, testName, packageInstalled, machineType, gp, ss, assertTimeout)
}

func addPackageInstallTest(key string) []*guestPolicyTestSetup {
	var pkgTestSetup []*guestPolicyTestSetup
	for name, image := range utils.HeadAptImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallTestSetup(name, image, "apt", key))
	}
	for name, image := range utils.HeadELImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallTestSetup(name, image, "yum", key))
	}
	for name, image := range utils.HeadSUSEImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallTestSetup(name, image, "zypper", key))
	}
	for name, image := range utils.HeadWindowsImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallTestSetup(name, image, "googet", key))
	}
	return pkgTestSetup
}

func buildPkgUpdateTestSetup(name, image, pkgManager, key string) *guestPolicyTestSetup {
	assertTimeout := 240 * time.Second
	testName := packageUpdateFunction
	packageName := "cowsay"
	machineType := "n1-standard-2"
	if pkgManager == "googet" {
		machineType = "n1-standard-4"
	}
	instanceName := fmt.Sprintf("%s-%s-%s-%s", path.Base(name), testName, key, utils.RandString(3))
	gp := &osconfigpb.GuestPolicy{
		Packages:   osconfigserver.BuildPackagePolicy(nil, nil, []string{packageName}),
		Assignment: &osconfigpb.Assignment{InstanceNamePrefixes: []string{instanceName}},
	}
	ss := getUpdateStartupScript(name, pkgManager, packageName)
	return newGuestPolicyTestSetup(image, instanceName, testName, packageNotInstalled, machineType, gp, ss, assertTimeout)
}

func addPackageUpdateTest(key string) []*guestPolicyTestSetup {
	var pkgTestSetup []*guestPolicyTestSetup
	for name, image := range utils.HeadAptImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgUpdateTestSetup(name, image, "apt", key))
	}
	for name, image := range utils.HeadELImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgUpdateTestSetup(name, image, "yum", key))
	}
	for name, image := range utils.HeadSUSEImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgUpdateTestSetup(name, image, "zypper", key))
	}
	for name, image := range utils.HeadWindowsImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgUpdateTestSetup(name, image, "googet", key))
	}
	return pkgTestSetup
}

func buildPkgDoesNotUpdateTestSetup(name, image, pkgManager, key string) *guestPolicyTestSetup {
	assertTimeout := 240 * time.Second
	testName := packageNoUpdateFunction
	packageName := "cowsay"
	machineType := "n1-standard-2"
	if pkgManager == "googet" {
		machineType = "n1-standard-4"
	}

	instanceName := fmt.Sprintf("%s-%s-%s-%s", path.Base(name), testName, key, utils.RandString(3))
	gp := &osconfigpb.GuestPolicy{
		Packages:   osconfigserver.BuildPackagePolicy([]string{packageName}, nil, nil),
		Assignment: &osconfigpb.Assignment{InstanceNamePrefixes: []string{instanceName}},
	}
	ss := getUpdateStartupScript(name, pkgManager, packageName)
	return newGuestPolicyTestSetup(image, instanceName, testName, packageInstalled, machineType, gp, ss, assertTimeout)
}

func addPackageDoesNotUpdateTest(key string) []*guestPolicyTestSetup {
	var pkgTestSetup []*guestPolicyTestSetup
	for name, image := range utils.HeadAptImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgDoesNotUpdateTestSetup(name, image, "apt", key))
	}
	for name, image := range utils.HeadELImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgDoesNotUpdateTestSetup(name, image, "yum", key))
	}
	for name, image := range utils.HeadSUSEImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgDoesNotUpdateTestSetup(name, image, "zypper", key))
	}
	for name, image := range utils.HeadWindowsImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgDoesNotUpdateTestSetup(name, image, "googet", key))
	}
	return pkgTestSetup
}

func buildPkgRemoveTestSetup(name, image, pkgManager, key string) *guestPolicyTestSetup {
	assertTimeout := 180 * time.Second
	testName := packageRemovalFunction
	packageName := "vim"
	machineType := "n1-standard-2"
	if pkgManager == "googet" {
		packageName = "certgen"
		machineType = "n1-standard-4"
	}

	instanceName := fmt.Sprintf("%s-%s-%s-%s", path.Base(name), testName, key, utils.RandString(3))
	gp := &osconfigpb.GuestPolicy{
		Packages:   osconfigserver.BuildPackagePolicy(nil, []string{packageName}, nil),
		Assignment: &osconfigpb.Assignment{InstanceNamePrefixes: []string{instanceName}},
	}
	ss := getStartupScript(name, pkgManager, packageName)
	return newGuestPolicyTestSetup(image, instanceName, testName, packageNotInstalled, machineType, gp, ss, assertTimeout)
}

func addPackageRemovalTest(key string) []*guestPolicyTestSetup {
	var pkgTestSetup []*guestPolicyTestSetup
	for name, image := range utils.HeadAptImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgRemoveTestSetup(name, image, "apt", key))
	}
	for name, image := range utils.HeadELImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgRemoveTestSetup(name, image, "yum", key))
	}
	for name, image := range utils.HeadSUSEImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgRemoveTestSetup(name, image, "zypper", key))
	}
	for name, image := range utils.HeadWindowsImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgRemoveTestSetup(name, image, "googet", key))
	}
	return pkgTestSetup
}

func buildPkgInstallFromNewRepoTestSetup(name, image, pkgManager, key string) *guestPolicyTestSetup {
	assertTimeout := 60 * time.Second
	packageName := "osconfig-agent-test"
	testName := packageInstallFromNewRepoFunction
	machineType := "n1-standard-2"
	if pkgManager == "googet" {
		machineType = "n1-standard-4"
	}

	instanceName := fmt.Sprintf("%s-%s-%s-%s", path.Base(name), testName, key, utils.RandString(3))
	gp := &osconfigpb.GuestPolicy{
		// Test that upgrade also installs.
		Packages:   osconfigserver.BuildPackagePolicy(nil, nil, []string{packageName}),
		Assignment: &osconfigpb.Assignment{InstanceNamePrefixes: []string{instanceName}},
		PackageRepositories: []*osconfigpb.PackageRepository{
			&osconfigpb.PackageRepository{Repository: osconfigserver.BuildAptRepository(osconfigpb.AptRepository_DEB, aptTestRepoBaseURL, osconfigTestRepo, aptRaptureGpgKey, []string{"main"})},
			&osconfigpb.PackageRepository{Repository: osconfigserver.BuildYumRepository(osconfigTestRepo, "Google OSConfig Agent Test Repository", yumTestRepoBaseURL, yumRaptureGpgKeys)},
			&osconfigpb.PackageRepository{Repository: osconfigserver.BuildZypperRepository(osconfigTestRepo, "Google OSConfig Agent Test Repository", yumTestRepoBaseURL, yumRaptureGpgKeys)},
			&osconfigpb.PackageRepository{Repository: osconfigserver.BuildGooRepository("Google OSConfig Agent Test Repository", gooTestRepoURL)},
		},
	}
	ss := getStartupScript(name, pkgManager, packageName)
	return newGuestPolicyTestSetup(image, instanceName, testName, packageInstalled, machineType, gp, ss, assertTimeout)
}

func addPackageInstallFromNewRepoTest(key string) []*guestPolicyTestSetup {
	var pkgTestSetup []*guestPolicyTestSetup
	for name, image := range utils.HeadAptImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallFromNewRepoTestSetup(name, image, "apt", key))
	}
	for name, image := range utils.HeadELImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallFromNewRepoTestSetup(name, image, "yum", key))
	}
	for name, image := range utils.HeadSUSEImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallFromNewRepoTestSetup(name, image, "zypper", key))
	}
	for name, image := range utils.HeadWindowsImages {
		pkgTestSetup = append(pkgTestSetup, buildPkgInstallFromNewRepoTestSetup(name, image, "googet", key))
	}
	return pkgTestSetup
}

func generateAllTestSetup(testProjectConfig *testconfig.Project) []*guestPolicyTestSetup {
	key := utils.RandString(3)

	pkgTestSetup := []*guestPolicyTestSetup{}
	pkgTestSetup = append(pkgTestSetup, addPackageInstallTest(key)...)
	pkgTestSetup = append(pkgTestSetup, addPackageRemovalTest(key)...)
	pkgTestSetup = append(pkgTestSetup, addPackageInstallFromNewRepoTest(key)...)
	pkgTestSetup = append(pkgTestSetup, addPackageUpdateTest(key)...)
	pkgTestSetup = append(pkgTestSetup, addPackageDoesNotUpdateTest(key)...)
	return pkgTestSetup
}

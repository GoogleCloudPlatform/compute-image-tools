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

package packagemanagement

import (
	"fmt"
	"path/filepath"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	api "google.golang.org/api/compute/v1"
)

var (
	pkgManagers = []string{"apt"}
)

// vf is the the vertificationFunction that is used in each testSetup during assertion of test case.
var vf = func(inst *compute.Instance, vfString string, port int64, interval, timeout time.Duration) error {
	return inst.WaitForSerialOutput(vfString, port, interval, timeout)
}

func addCreateOsConfigTest(pkgTestSetup []*packageManagementTestSetup) []*packageManagementTestSetup {
	testName := "createosconfigtest"
	desc := "test osconfig creation"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig

		switch pkgManager {
		case "apt":
			pkg := osconfigserver.BuildPackage("cowsay")
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
			break
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}
		setup := packageManagementTestSetup{
			image:      debianImage,
			name:       fmt.Sprintf("%s-%s", filepath.Base(debianImage), testName),
			osconfig:   oc,
			assignment: nil,
			fname:      testName,
			vf:         vf,
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}
func addPackageInstallTest(pkgTestSetup []*packageManagementTestSetup) []*packageManagementTestSetup {
	testName := "packageinstalltest"
	desc := "test package installation"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var assign *osconfigpb.Assignment
		var instaneName, ss, vs string

		switch pkgManager {
		case "apt":
			instaneName = fmt.Sprintf("%s-%s", filepath.Base(debianImage), testName)
			pkg := osconfigserver.BuildPackage("cowsay")
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instaneName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			ss = fmt.Sprintf("%s\nwhile true; do /usr/bin/dpkg-query -s cowsay | sudo tee /dev/ttyS0; sleep 5; done", utils.InstallOSConfigDeb)
			vs = fmt.Sprintf("install ok installed")
			break
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}
		setup := packageManagementTestSetup{
			image:      debianImage,
			name:       instaneName,
			osconfig:   oc,
			assignment: assign,
			fname:      testName,
			vf:         vf,
			vstring:    vs,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &ss,
			},
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}

func addPackageRemovalTest(pkgTestSetup []*packageManagementTestSetup) []*packageManagementTestSetup {
	testName := "packageremovaltest"
	desc := "test package removal"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var assign *osconfigpb.Assignment
		var instaneName, ss, vs string

		switch pkgManager {
		case "apt":
			instaneName = fmt.Sprintf("%s-%s", filepath.Base(debianImage), testName)
			pkg := osconfigserver.BuildPackage("cowsay")
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(nil, pkgs, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instaneName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			ss = fmt.Sprintf("%s\nsudo apt-get -y install cowsay\nwhile true; do /usr/bin/dpkg-query -s cowsay | sudo tee /dev/ttyS0; sleep 5; done", utils.InstallOSConfigDeb)
			vs = fmt.Sprintf("package 'cowsay' is not installed")
			break
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}
		setup := packageManagementTestSetup{
			image:      debianImage,
			name:       instaneName,
			osconfig:   oc,
			assignment: assign,
			fname:      testName,
			vf:         vf,
			vstring:    vs,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &ss,
			},
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}

func addPackageInstallRemovalTest(pkgTestSetup []*packageManagementTestSetup) []*packageManagementTestSetup {
	testName := "packageinstallremovaltest"
	desc := "test package removal takes precedence over package installation"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var assign *osconfigpb.Assignment
		var instaneName, ss, vs string

		switch pkgManager {
		case "apt":
			instaneName = fmt.Sprintf("%s-%s", filepath.Base(debianImage), testName)
			pkg := osconfigserver.BuildPackage("cowsay")
			installPkg := []*osconfigpb.Package{pkg}
			pkg = osconfigserver.BuildPackage("cowsay")
			removePkg := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instaneName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			ss = fmt.Sprintf("%s\nwhile true; do /usr/bin/dpkg-query -s wget | sudo tee /dev/ttyS0; sleep 5; done", utils.InstallOSConfigDeb)
			vs = fmt.Sprintf("package 'cowsay' is not installed")
			break
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}
		setup := packageManagementTestSetup{
			image:      debianImage,
			name:       instaneName,
			osconfig:   oc,
			assignment: assign,
			fname:      testName,
			vf:         vf,
			vstring:    vs,
			startup: &api.MetadataItems{
				Key:   "startup-script",
				Value: &ss,
			},
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}

func generateAllTestSetup() []*packageManagementTestSetup {
	pkgTestSetup := []*packageManagementTestSetup{}
	pkgTestSetup = addCreateOsConfigTest(pkgTestSetup)
	pkgTestSetup = addPackageInstallTest(pkgTestSetup)
	pkgTestSetup = addPackageRemovalTest(pkgTestSetup)
	pkgTestSetup = addPackageInstallRemovalTest(pkgTestSetup)
	return pkgTestSetup
}

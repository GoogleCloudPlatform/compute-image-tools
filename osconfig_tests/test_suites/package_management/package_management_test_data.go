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
	"path"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	api "google.golang.org/api/compute/v1"
)

const (
	packageInstalledString    = "package is installed"
	packageNotInstalledString = "package is not installed"
)

var (
	pkgManagers = []string{"apt", "yum"}
)

// vf is the the vertificationFunction that is used in each testSetup during assertion of test case.
var vf = func(inst *compute.Instance, vfString string, port int64, interval, timeout time.Duration) error {
	return inst.WaitForSerialOutput(vfString, port, interval, timeout)
}

func addCreateOsConfigTest(pkgTestSetup []*packageManagementTestSetup) []*packageManagementTestSetup {
	testName := "createosconfigtest"
	desc := "test osconfig creation"
	packageName := "cowsay"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var image string

		switch pkgManager {
		case "apt":
			pkg := osconfigserver.BuildPackage(packageName)
			image = debianImage
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
		case "yum":
			image = centosImage
			pkg := osconfigserver.BuildPackage(packageName)
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}
		setup := packageManagementTestSetup{
			image:      image,
			name:       fmt.Sprintf("%s-%s", path.Base(image), testName),
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
	packageName := "cowsay"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var assign *osconfigpb.Assignment
		var instanceName, image, ss, vs string

		switch pkgManager {
		case "apt":
			image = debianImage
			instanceName = fmt.Sprintf("%s-%s", path.Base(image), testName)
			pkg := osconfigserver.BuildPackage(packageName)
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			vs = fmt.Sprintf(packageInstalledString)
			break
		case "yum":
			image = centosImage
			instanceName = fmt.Sprintf("%s-%s", path.Base(image), testName)
			pkg := osconfigserver.BuildPackage(packageName)
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			vs = fmt.Sprintf(packageInstalledString)
			break
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}

		ss = getPackageInstallStartupScript(pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:      image,
			name:       instanceName,
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
	packageName := "cowsay"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var assign *osconfigpb.Assignment
		var instanceName, image, ss, vs string

		switch pkgManager {
		case "apt":
			image = debianImage
			instanceName = fmt.Sprintf("%s-%s", path.Base(debianImage), testName)
			pkg := osconfigserver.BuildPackage(packageName)
			pkgs := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(nil, pkgs, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			vs = fmt.Sprintf(packageNotInstalledString)
		case "yum":
			image = centosImage
			instanceName = fmt.Sprintf("%s-%s", path.Base(image), testName)
			pkg := osconfigserver.BuildPackage(packageName)
			removePkg := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			vs = fmt.Sprintf(packageNotInstalledString)
			break
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}

		ss = getPackageRemovalStartupScript(pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:      image,
			name:       instanceName,
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
	packageName := "cowsay"
	for _, pkgManager := range pkgManagers {
		var oc *osconfigpb.OsConfig
		var assign *osconfigpb.Assignment
		var instanceName, image, ss, vs string

		switch pkgManager {
		case "apt":
			pkg := osconfigserver.BuildPackage(packageName)
			image = debianImage
			instanceName = fmt.Sprintf("%s-%s", path.Base(image), testName)
			installPkg := []*osconfigpb.Package{pkg}
			pkg = osconfigserver.BuildPackage(packageName)
			removePkg := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			vs = fmt.Sprintf(packageNotInstalledString)
		case "yum":
			image = centosImage
			instanceName = fmt.Sprintf("%s-%s", path.Base(image), testName)
			pkg := osconfigserver.BuildPackage(packageName)
			installPkg := []*osconfigpb.Package{pkg}
			pkg = osconfigserver.BuildPackage(packageName)
			removePkg := []*osconfigpb.Package{pkg}
			oc = osconfigserver.BuildOsConfig(testName, desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			assign = osconfigserver.BuildAssignment(testName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectID, oc.Name)})
			vs = fmt.Sprintf(packageNotInstalledString)
		default:
			panic(fmt.Sprintf("non existent package manager: %s", pkgManager))
		}

		ss = getPackageInstallRemovalStartupScript(pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:      image,
			name:       instanceName,
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

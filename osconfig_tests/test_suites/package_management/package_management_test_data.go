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
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	api "google.golang.org/api/compute/v1"
)

type platformPkgManagerTuple struct {
	platform   string
	pkgManager string
}

const (
	packageInstalledString    = "package is installed"
	packageNotInstalledString = "package is not installed"
)

var (
	tuples = []platformPkgManagerTuple{{"debian", "apt"}, {"centos", "yum"}, {"rhel", "yum"}}
)

// vf is the the vertificationFunction that is used in each testSetup during assertion of test case.
var vf = func(inst *compute.Instance, vfString string, port int64, interval, timeout time.Duration) error {
	return inst.WaitForSerialOutput(vfString, port, interval, timeout)
}

func addCreateOsConfigTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "createosconfigtest"
	desc := "test osconfig creation"
	packageName := "cowsay"
	for _, tuple := range tuples {
		var oc *osconfigpb.OsConfig
		var image string

		switch tuple.platform {
		case "debian":
			image = debianImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
		case "centos":
			image = centosImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
		case "rhel":
			image = rhelImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
		default:
			panic(fmt.Sprintf("non existent platform: %s", tuple.platform))
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
func addPackageInstallTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "packageinstalltest"
	desc := "test package installation"
	packageName := "cowsay"
	for _, tuple := range tuples {
		var oc *osconfigpb.OsConfig
		var image, vs string

		switch tuple.platform {
		case "debian":
			image = debianImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "centos":
			image = centosImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "rhel":
			image = rhelImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		default:
			panic(fmt.Sprintf("non existent platform: %s", tuple.platform))
		}

		instanceName := fmt.Sprintf("%s-%s", path.Base(image), testName)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageInstallStartupScript(tuple.pkgManager, packageName)
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

func addPackageRemovalTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "packageremovaltest"
	desc := "test package removal"
	packageName := "cowsay"
	for _, tuple := range tuples {
		var oc *osconfigpb.OsConfig
		var image, vs string

		switch tuple.platform {
		case "debian":
			image = debianImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(nil, pkgs, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "centos":
			image = centosImage
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "rhel":
			image = rhelImage
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		default:
			panic(fmt.Sprintf("non existent platform: %s", tuple.platform))
		}

		instanceName := fmt.Sprintf("%s-%s", path.Base(image), testName)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageRemovalStartupScript(tuple.pkgManager, packageName)
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

func addPackageInstallRemovalTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "packageinstallremovaltest"
	desc := "test package removal takes precedence over package installation"
	packageName := "cowsay"
	for _, tuple := range tuples {
		var oc *osconfigpb.OsConfig
		var image, vs string

		switch tuple.platform {
		case "debian":
			image = debianImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "centos":
			image = centosImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "rhel":
			image = rhelImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		default:
			panic(fmt.Sprintf("non existent platform: %s", tuple.platform))
		}

		instanceName := fmt.Sprintf("%s-%s", path.Base(image), testName)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s", path.Base(image), testName), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageInstallRemovalStartupScript(tuple.pkgManager, packageName)
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

func generateAllTestSetup(testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	pkgTestSetup := []*packageManagementTestSetup{}
	pkgTestSetup = addCreateOsConfigTest(pkgTestSetup, testProjectConfig)
	pkgTestSetup = addPackageInstallTest(pkgTestSetup, testProjectConfig)
	pkgTestSetup = addPackageRemovalTest(pkgTestSetup, testProjectConfig)
	pkgTestSetup = addPackageInstallRemovalTest(pkgTestSetup, testProjectConfig)
	return pkgTestSetup
}

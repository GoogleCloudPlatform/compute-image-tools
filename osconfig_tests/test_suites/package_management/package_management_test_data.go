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

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	osconfigserver "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	testconfig "github.com/GoogleCloudPlatform/compute-image-tools/test_common/test_config"
	"github.com/GoogleCloudPlatform/guest-logging-go/logger"
	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	api "google.golang.org/api/compute/v1"
)

type platformPkgManagerTuple struct {
	platform   string
	pkgManager string
}

const (
	packageInstalledString    = "package is installed"
	packageNotInstalledString = "package is not installed"
	osconfigTestRepo          = "osconfig-agent-test-repository"
	yumTestRepoBaseURL        = "https://packages.cloud.google.com/yum/repos/osconfig-agent-test-repository"
	aptTestRepoBaseURL        = "http://packages.cloud.google.com/apt"
	gooTestRepoURL            = "https://packages.cloud.google.com/yuck/repos/osconfig-agent-test-repository"
	aptRaptureGpgKey          = "https://packages.cloud.google.com/apt/doc/apt-key.gpg"
)

var (
	platformPkgManagers = []platformPkgManagerTuple{{"debian", "apt"}, {"centos", "yum"}, {"rhel", "yum"}, {"windows", "googet"}}
	yumRaptureGpgKeys   = []string{"https://packages.cloud.google.com/yum/doc/yum-key.gpg", "https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg"}
)

// vf is the the vertificationFunction that is used in each testSetup during assertion of test case.
var vf = func(inst *compute.Instance, vfString string, port int64, interval, timeout time.Duration) error {
	return inst.WaitForSerialOutput(vfString, port, interval, timeout)
}

func addPackageInstallTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	const testName = packageInstallFunction
	desc := "test package installation"
	packageName := "cowsay"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 60 * time.Second

		switch tuple.platform {
		case "debian":
			for _, image := range debianImages {
				pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "centos":
			for _, image := range centosImages {
				pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "rhel":
			for _, image := range rhelImages {
				pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "windows":
			for _, image := range windowsImages {
				pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, nil, osconfigserver.BuildGooPackageConfig(pkgs, nil, nil), nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}
	}
	return pkgTestSetup
}

func addPackageRemovalTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	const testName = packageRemovalFunction
	desc := "test package removal"
	packageName := "cowsay"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 600 * time.Second

		switch tuple.platform {
		case "debian":
			for _, image := range debianImages {
				pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, osconfigserver.BuildAptPackageConfig(nil, pkgs, nil), nil, nil, nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "centos":
			for _, image := range centosImages {
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "rhel":
			for _, image := range rhelImages {
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "windows":
			for _, image := range windowsImages {
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, nil, osconfigserver.BuildGooPackageConfig(nil, removePkg, nil), nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}
	}
	return pkgTestSetup
}

func addPackageInstallRemovalTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	const testName = packageInstallRemovalFunction
	desc := "test package removal takes precedence over package installation"
	packageName := "cowsay"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 60 * time.Second

		switch tuple.platform {
		case "debian":
			for _, image := range debianImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "centos":
			for _, image := range centosImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, removePkg, nil), nil, nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "rhel":
			for _, image := range rhelImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, removePkg, nil), nil, nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "windows":
			for _, image := range windowsImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, nil, osconfigserver.BuildGooPackageConfig(installPkg, removePkg, nil), nil, nil)
				vs = fmt.Sprintf(packageNotInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallRemovalStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}

		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}
	}
	return pkgTestSetup
}

func addPackageInstallFromNewRepoTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	const testName = packageInstallFromNewRepoFunction
	desc := "test package installation from new package"
	packageName := "osconfig-agent-test"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 60 * time.Second

		switch tuple.platform {
		case "debian":
			for _, image := range debianImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				repos := []*osconfigpb.AptRepository{osconfigserver.BuildAptRepository(osconfigpb.AptRepository_DEB, aptTestRepoBaseURL, osconfigTestRepo, aptRaptureGpgKey, []string{"main"})}
				oc = osconfigserver.BuildOsConfig(instanceName, desc, osconfigserver.BuildAptPackageConfig(installPkg, nil, repos), nil, nil, nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallFromNewRepoTestStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout,
					vf)
			}
		case "centos":
			for _, image := range centosImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				repos := []*osconfigpb.YumRepository{osconfigserver.BuildYumRepository(osconfigTestRepo, "Google OSConfig Agent Test Repository", yumTestRepoBaseURL, yumRaptureGpgKeys)}
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, nil, repos), nil, nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallFromNewRepoTestStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "rhel":
			for _, image := range rhelImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				repos := []*osconfigpb.YumRepository{osconfigserver.BuildYumRepository(osconfigTestRepo, "Google OSConfig Agent Test Repository", yumTestRepoBaseURL, yumRaptureGpgKeys)}
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, nil, repos), nil, nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallFromNewRepoTestStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		case "windows":
			for _, image := range windowsImages {
				installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
				instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
				repos := []*osconfigpb.GooRepository{osconfigserver.BuildGooRepository("Google OSConfig Agent Test Repository", gooTestRepoURL)}
				oc = osconfigserver.BuildOsConfig(instanceName, desc, nil, nil, osconfigserver.BuildGooPackageConfig(installPkg, nil, repos), nil, nil)
				vs = fmt.Sprintf(packageInstalledString)
				assign := osconfigserver.BuildAssignment(instanceName, desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
				ss := getPackageInstallFromNewRepoTestStartupScript(path.Base(image), tuple.pkgManager, packageName)
				pkgTestSetup = createAndAppendSetup(pkgTestSetup, image, instanceName, testName, vs, oc, assign, ss, assertTimeout, vf)
			}
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}
	}
	return pkgTestSetup
}

func createAndAppendSetup(pkgTestSetup []*packageManagementTestSetup, image, name string, fname packageMangementTestFunctionName, vs string, oc *osconfigpb.OsConfig, assignment *osconfigpb.Assignment, startup *api.MetadataItems, assertTimeout time.Duration, vf func(*compute.Instance, string, int64, time.Duration, time.Duration) error) []*packageManagementTestSetup {
	var setup *packageManagementTestSetup
	newPackageManagementTestSetup(&setup, image, name, fname, vs, oc, assignment, startup, assertTimeout, vf)
	pkgTestSetup = append(pkgTestSetup, setup)
	return pkgTestSetup
}

func generateAllTestSetup(testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	pkgTestSetup := []*packageManagementTestSetup{}
	pkgTestSetup = addPackageInstallTest(pkgTestSetup, testProjectConfig)
	pkgTestSetup = addPackageRemovalTest(pkgTestSetup, testProjectConfig)
	pkgTestSetup = addPackageInstallRemovalTest(pkgTestSetup, testProjectConfig)
	pkgTestSetup = addPackageInstallFromNewRepoTest(pkgTestSetup, testProjectConfig)
	return pkgTestSetup
}

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
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/osconfig_server"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/test_config"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
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

func addCreateOsConfigTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "createosconfigtest"
	desc := "test osconfig creation"
	packageName := "cowsay"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var image string
		uniqueSuffix := utils.RandString(5)

		switch tuple.platform {
		case "debian":
			image = debianImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
		case "centos":
			image = centosImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
		case "rhel":
			image = rhelImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
		case "windows":
			image = windowsImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, nil, osconfigserver.BuildGooPackageConfig(pkgs, nil, nil), nil, nil)
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
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
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var image, vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 60 * time.Second

		switch tuple.platform {
		case "debian":
			image = debianImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildAptPackageConfig(pkgs, nil, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "centos":
			image = centosImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "rhel":
			image = rhelImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(pkgs, nil, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "windows":
			image = windowsImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, nil, osconfigserver.BuildGooPackageConfig(pkgs, nil, nil), nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
			assertTimeout = 1200 * time.Second
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}

		instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageInstallStartupScript(tuple.pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:      image,
			name:       instanceName,
			osconfig:   oc,
			assignment: assign,
			fname:      testName,
			vf:         vf,
			vstring:    vs,
			startup:    ss,
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}

func addPackageRemovalTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "packageremovaltest"
	desc := "test package removal"
	packageName := "cowsay"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var image, vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 600 * time.Second

		switch tuple.platform {
		case "debian":
			image = debianImage
			pkgs := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildAptPackageConfig(nil, pkgs, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "centos":
			image = centosImage
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "rhel":
			image = rhelImage
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(nil, removePkg, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "windows":
			image = windowsImage
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, nil, osconfigserver.BuildGooPackageConfig(nil, removePkg, nil), nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}

		instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageRemovalStartupScript(tuple.pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:         image,
			name:          instanceName,
			osconfig:      oc,
			assignment:    assign,
			fname:         testName,
			vf:            vf,
			assertTimeout: assertTimeout,
			vstring:       vs,
			startup:       ss,
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}

func addPackageInstallRemovalTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "packageinstallremovaltest"
	desc := "test package removal takes precedence over package installation"
	packageName := "cowsay"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var image, vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 60 * time.Second

		switch tuple.platform {
		case "debian":
			image = debianImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildAptPackageConfig(installPkg, removePkg, nil), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "centos":
			image = centosImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, removePkg, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "rhel":
			image = rhelImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, removePkg, nil), nil, nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)
		case "windows":
			image = windowsImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			removePkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, nil, osconfigserver.BuildGooPackageConfig(installPkg, removePkg, nil), nil, nil)
			vs = fmt.Sprintf(packageNotInstalledString)

		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}

		instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageInstallRemovalStartupScript(tuple.pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:      image,
			name:       instanceName,
			osconfig:   oc,
			assignment: assign,
			fname:      testName,
			vf:         vf,
			vstring:    vs,
			startup:    ss,
		}
		pkgTestSetup = append(pkgTestSetup, &setup)
	}
	return pkgTestSetup
}

func addPackageInstallFromNewRepoTest(pkgTestSetup []*packageManagementTestSetup, testProjectConfig *testconfig.Project) []*packageManagementTestSetup {
	testName := "packageinstallfromnewrepotest"
	desc := "test package installation from new package"
	packageName := "osconfig-agent-test"
	for _, tuple := range platformPkgManagers {
		var oc *osconfigpb.OsConfig
		var image, vs string
		uniqueSuffix := utils.RandString(5)
		assertTimeout := 60 * time.Second

		switch tuple.platform {
		case "debian":
			image = debianImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			repos := []*osconfigpb.AptRepository{osconfigserver.BuildAptRepository(osconfigpb.AptRepository_DEB, aptTestRepoBaseURL, osconfigTestRepo, aptRaptureGpgKey, []string{"main"})}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildAptPackageConfig(installPkg, nil, repos), nil, nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "centos":
			image = centosImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			repos := []*osconfigpb.YumRepository{osconfigserver.BuildYumRepository(osconfigTestRepo, "Google OSConfig Agent Test Repository", yumTestRepoBaseURL, yumRaptureGpgKeys)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, nil, repos), nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "rhel":
			image = rhelImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			repos := []*osconfigpb.YumRepository{osconfigserver.BuildYumRepository(osconfigTestRepo, "Google OSConfig Agent Test Repository", yumTestRepoBaseURL, yumRaptureGpgKeys)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, osconfigserver.BuildYumPackageConfig(installPkg, nil, repos), nil, nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
		case "windows":
			image = windowsImage
			installPkg := []*osconfigpb.Package{osconfigserver.BuildPackage(packageName)}
			repos := []*osconfigpb.GooRepository{osconfigserver.BuildGooRepository("Google OSConfig Agent Test Repository", gooTestRepoURL)}
			oc = osconfigserver.BuildOsConfig(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, nil, nil, osconfigserver.BuildGooPackageConfig(installPkg, nil, repos), nil, nil)
			vs = fmt.Sprintf(packageInstalledString)
			assertTimeout = 1200 * time.Second
		default:
			logger.Errorf(fmt.Sprintf("non existent platform: %s", tuple.platform))
			continue
		}

		instanceName := fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix)
		assign := osconfigserver.BuildAssignment(fmt.Sprintf("%s-%s-%s", path.Base(image), testName, uniqueSuffix), desc, osconfigserver.BuildInstanceFilterExpression(instanceName), []string{fmt.Sprintf("projects/%s/osConfigs/%s", testProjectConfig.TestProjectID, oc.Name)})
		ss := getPackageInstallFromNewRepoTestStartupScript(tuple.pkgManager, packageName)
		setup := packageManagementTestSetup{
			image:         image,
			name:          instanceName,
			osconfig:      oc,
			assignment:    assign,
			fname:         testName,
			vf:            vf,
			assertTimeout: assertTimeout,
			vstring:       vs,
			startup:       ss,
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
	pkgTestSetup = addPackageInstallFromNewRepoTest(pkgTestSetup, testProjectConfig)
	return pkgTestSetup
}

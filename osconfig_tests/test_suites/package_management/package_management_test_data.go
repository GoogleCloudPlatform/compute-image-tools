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

const (
	PackageInstallTestOsConfig = &osconfigpb.OsConfig{
		Name:        "packageinstalltest",
		Description: "test osconfig to test package installation",
		Apt: &osconfigpb.AptPackageConfig{
			PackageInstalls: []*osconfigpb.Package{
				&osconfigpb.Package{
					Name: "cowsay",
				},
			},
		},
	}

	PackageInstallTestAssignment = &osconfigpb.Assignment{
		Name:        "packageinstalltest",
		Description: "test assignment to test package installation",
		OsConfigs: []*string{
			"projects/281997379984/osConfigs/packageinstalltest",
		},
		Expression: "instance.Name=\"osconfig-test-debian-9-packageinstalltest\"",
	}

	PackageRemovalTestOsConfig = &osconfigpb.OsConfig{
		Name:        "packageremovaltest",
		Description: "test osconfig to test package removal",
		Apt: &osconfigpb.AptPackageConfig{
			PackageRemovals: []*osconfigpb.Package{
				&osconfigpb.Package{
					Name: "cowsay",
				},
			},
		},
	}

	PackageRemovalTestAssignment = &osconfigpb.Assignment{
		Name:        "packageremovaltest",
		Description: "test assignment to test package removal",
		OsConfigs: []*string{
			"projects/281997379984/osConfigs/packageremovaltest",
		},
		Expression: "instance.Name=\"osconfig-test-debian-9-packageremovaltest\"",
	}

	PackageInstalRemovalTestOsConfig = &osconfigpb.OsConfig{
		Name:        "packageinstallremovaltest",
		Description: "test osconfig to test package removal takes precedence over installation",
		Apt: &osconfigpb.AptPackageConfig{
			PackageInstalls: []*osconfigpb.Package{
				&osconfigpb.Package{
					Name: "cowsay",
				},
			},
			PackageRemovals: []*osconfigpb.Package{
				&osconfigpb.Package{
					Name: "cowsay",
				},
			},
		},
	}

	PackageInstallRemovalTestAssignment = &osconfigpb.Assignment{
		Name:        "packageinstallremovaltest",
		Description: "test assignment to test package install removal test",
		OsConfigs: []*string{
			"projects/281997379984/osConfigs/packageinstallremovaltest",
		},
		Expression: "instance.Name=\"osconfig-test-debian-9-packageinstallremovaltest\"",
	}
)

//  Copyright 2018 Google Inc. All Rights Reserved.
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

package main

import (
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/package_library"
)

func runPackageConfig(res *osconfigpb.LookupConfigsResponse) {
	if res.Apt != nil && packages.AptExists {
		aptRepositories(res.Apt.Repositories)
		aptInstalls(res.Apt.PackageInstalls)
		aptRemovals(res.Apt.PackageRemovals)
	}

	if res.Yum != nil && packages.YumExists {
		yumRepositories(res.Yum.Repositories)
		yumInstalls(res.Yum.PackageInstalls)
		yumRemovals(res.Yum.PackageRemovals)
	}
}

func aptRepositories(repos []*osconfigpb.AptRepository) {}

func aptInstalls(pkgs []*osconfigpb.Package) {}

func aptRemovals(pkgs []*osconfigpb.Package) {}

func yumRepositories(repos []*osconfigpb.YumRepository) {}

func yumInstalls(pkgs []*osconfigpb.Package) {}

func yumRemovals(pkgs []*osconfigpb.Package) {}

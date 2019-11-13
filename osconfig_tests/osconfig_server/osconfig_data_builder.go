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

package osconfigserver

import (
	"fmt"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/_internal/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha2"
)

// BuildPackagePolicy creates an package policy.
func BuildPackagePolicy(installs, removes, upgrades []string) []*osconfigpb.Package {
	var pkgs []*osconfigpb.Package
	for _, p := range installs {
		pkgs = append(pkgs, &osconfigpb.Package{
			DesiredState: osconfigpb.DesiredState_INSTALLED,
			Name:         p,
		})
	}
	for _, p := range removes {
		pkgs = append(pkgs, &osconfigpb.Package{
			DesiredState: osconfigpb.DesiredState_REMOVED,
			Name:         p,
		})
	}
	for _, p := range upgrades {
		pkgs = append(pkgs, &osconfigpb.Package{
			DesiredState: osconfigpb.DesiredState_UPDATED,
			Name:         p,
		})
	}

	return pkgs
}

// BuildAptRepository create an apt repository object
func BuildAptRepository(archiveType osconfigpb.AptRepository_ArchiveType, uri, distribution, keyuri string, components []string) *osconfigpb.PackageRepository_Apt {
	return &osconfigpb.PackageRepository_Apt{
		Apt: &osconfigpb.AptRepository{
			ArchiveType:  archiveType,
			Uri:          uri,
			Distribution: distribution,
			Components:   components,
			GpgKey:       keyuri,
		},
	}
}

// BuildYumRepository create an yum repository object
func BuildYumRepository(id, name, baseURL string, gpgkeys []string) *osconfigpb.PackageRepository_Yum {
	return &osconfigpb.PackageRepository_Yum{
		Yum: &osconfigpb.YumRepository{
			Id:          id,
			DisplayName: name,
			BaseUrl:     baseURL,
			GpgKeys:     gpgkeys,
		},
	}
}

// BuildZypperRepository create an zypper repository object
func BuildZypperRepository(id, name, baseURL string, gpgkeys []string) *osconfigpb.PackageRepository_Zypper {
	return &osconfigpb.PackageRepository_Zypper{
		Zypper: &osconfigpb.ZypperRepository{
			Id:          id,
			DisplayName: name,
			BaseUrl:     baseURL,
			GpgKeys:     gpgkeys,
		},
	}
}

// BuildGooRepository create an googet repository object
func BuildGooRepository(name, url string) *osconfigpb.PackageRepository_Goo {
	return &osconfigpb.PackageRepository_Goo{
		Goo: &osconfigpb.GooRepository{
			Name: name,
			Url:  url,
		},
	}
}

// BuildInstanceFilterExpression creates an instance filter expression to
// be used by Assignment
func BuildInstanceFilterExpression(instance string) string {
	return fmt.Sprintf("instance.name==\"%s\"", instance)
}

// BuildPackage creates an os config package
func BuildPackage(name string) *osconfigpb.Package {
	return &osconfigpb.Package{
		Name: name,
	}
}

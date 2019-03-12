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

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

// BuildOsConfig creates a struct of OsConfig
func BuildOsConfig(name, description string, aptconfig *osconfigpb.AptPackageConfig, yumconfig *osconfigpb.YumPackageConfig, gooconfig *osconfigpb.GooPackageConfig, wuconfig *osconfigpb.WindowsUpdateConfig, zypperconfig *osconfigpb.ZypperPackageConfig) *osconfigpb.OsConfig {
	return &osconfigpb.OsConfig{
		Name:          name,
		Description:   description,
		Apt:           aptconfig,
		Yum:           yumconfig,
		Goo:           gooconfig,
		WindowsUpdate: wuconfig,
		Zypper:        zypperconfig,
	}
}

// BuildAssignment creates a struct of Assignment
func BuildAssignment(name, description, expression string, osconfigs []string) *osconfigpb.Assignment {
	return &osconfigpb.Assignment{
		Name:        name,
		Description: description,
		Expression:  expression,
		OsConfigs:   osconfigs,
	}
}

// BuildAptPackageConfig creates an Apt package config
func BuildAptPackageConfig(installs, removes []*osconfigpb.Package, repos []*osconfigpb.AptRepository) *osconfigpb.AptPackageConfig {
	return &osconfigpb.AptPackageConfig{
		PackageInstalls: installs,
		PackageRemovals: removes,
		Repositories:    repos,
	}
}

// BuildYumPackageConfig creates a yum package config
func BuildYumPackageConfig(installs, removes []*osconfigpb.Package, repos []*osconfigpb.YumRepository) *osconfigpb.YumPackageConfig {
	return &osconfigpb.YumPackageConfig{
		PackageInstalls: installs,
		PackageRemovals: removes,
		Repositories:    repos,
	}
}

// BuildGooPackageConfig create a goo package config
func BuildGooPackageConfig(installs, removes []*osconfigpb.Package, repos []*osconfigpb.GooRepository) *osconfigpb.GooPackageConfig {
	return &osconfigpb.GooPackageConfig{
		PackageInstalls: installs,
		PackageRemovals: removes,
		Repositories:    repos,
	}
}

// BuildZypperPackageConfig create a zypper package config
func BuildZypperPackageConfig(installs, removes []*osconfigpb.Package, repos []*osconfigpb.ZypperRepository) *osconfigpb.ZypperPackageConfig {
	return &osconfigpb.ZypperPackageConfig{
		PackageInstalls: installs,
		PackageRemovals: removes,
		Repositories:    repos,
	}
}

func BuildAptRepository(archiveType osconfigpb.AptRepository_ArchiveType, uri, distribution, keyuri string, components []string) *osconfigpb.AptRepository {
	return &osconfigpb.AptRepository{
		ArchiveType:  archiveType,
		Uri:          uri,
		Distribution: distribution,
		Components:   components,
		KeyUri:       keyuri,
	}
}

func BuildYumRepository(id, name, baseUrl string, gpgkeys []string) *osconfigpb.YumRepository {
	return &osconfigpb.YumRepository{
		Id:          id,
		DisplayName: name,
		BaseUrl:     baseUrl,
		GpgKeys:     gpgkeys,
	}
}

func BuildZypperRepository(id, name, baseUrl string, gpgkeys []string) *osconfigpb.ZypperRepository {
	return &osconfigpb.ZypperRepository{
		Id:          id,
		DisplayName: name,
		BaseUrl:     baseUrl,
		GpgKeys:     gpgkeys,
	}
}

func BuildGooRepository(name, url string) *osconfigpb.GooRepository {
	return &osconfigpb.GooRepository{
		Name: name,
		Url:  url,
	}
}

func BuildWindowsUpdateConfig(uri string) *osconfigpb.WindowsUpdateConfig {
	return &osconfigpb.WindowsUpdateConfig{
		WindowsUpdateServerUri: uri,
	}
}

// BuildWUPackageConfig create a window update config
func BuildWUPackageConfig(wusu string) *osconfigpb.WindowsUpdateConfig {
	return &osconfigpb.WindowsUpdateConfig{
		WindowsUpdateServerUri: wusu,
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

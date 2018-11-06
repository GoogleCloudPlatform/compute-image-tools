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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/compute-image-tools/go/packages"
)

const zypperRepoFile = "/etc/zypp/repos.d/google_osconfig.repo"

func setOsConfig(res *osconfigpb.LookupConfigsResponse) error {
	var errs []string
	if res.Apt != nil && packages.AptExists {
		if err := aptRepositories(res.Apt.Repositories); err != nil {
			errs = append(errs, fmt.Sprintf("error writing apt repo file: %v", err))
		}
		if err := aptInstalls(res.Apt.PackageInstalls); err != nil {
			errs = append(errs, fmt.Sprintf("error installing apt packages: %v", err))
		}
		if err := aptRemovals(res.Apt.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error removing apt packages: %v", err))
		}
	}

	if res.Yum != nil && packages.YumExists {
		if err := yumRepositories(res.Yum.Repositories); err != nil {
			errs = append(errs, fmt.Sprintf("error writing yum repo file: %v", err))
		}
		if err := yumInstalls(res.Yum.PackageInstalls); err != nil {
			errs = append(errs, fmt.Sprintf("error installing yum packages: %v", err))
		}
		if err := yumRemovals(res.Yum.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error removing yum packages: %v", err))
		}
	}

	if res.Zypper != nil && packages.ZypperExists {
		if err := zypperRepositories(res.Zypper.Repositories); err != nil {
			errs = append(errs, fmt.Sprintf("error writing zypper repo file: %v", err))
		}
		if err := zypperInstalls(res.Zypper.PackageInstalls); err != nil {
			errs = append(errs, fmt.Sprintf("error installing zypper packages: %v", err))
		}
		if err := zypperRemovals(res.Zypper.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error removing zypper packages: %v", err))
		}
	}

	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

func aptRepositories(repos []*osconfigpb.AptRepository) error { return nil }

func aptInstalls(pkgs []*osconfigpb.Package) error { return nil }

func aptRemovals(pkgs []*osconfigpb.Package) error { return nil }

func yumRepositories(repos []*osconfigpb.YumRepository) error { return nil }

func yumInstalls(pkgs []*osconfigpb.Package) error { return nil }

func yumRemovals(pkgs []*osconfigpb.Package) error { return nil }

func zypperRepositories(repos []*osconfigpb.ZypperRepository) error {
	/*
		# Repo file managed by Google OSConfig agent
		- id: repo1-name
		  display_name: repo1
		  url: https://repo1-url
		- id: repo2-name
		  display_name: repo2-name
		  url: https://repo2-url
	*/
	var buf bytes.Buffer
	buf.WriteString("# Repo file managed by Google OSConfig agent\n")
	for _, repo := range repos {
		buf.WriteString(fmt.Sprintf("\n- id: %s\n", repo.Id))
		if repo.DisplayName == "" {
			buf.WriteString(fmt.Sprintf("\n- display_name: %s\n", repo.Id))
		} else {
			buf.WriteString(fmt.Sprintf("\n- display_name: %s\n", repo.DisplayName))
		}
		buf.WriteString(fmt.Sprintf("  url: %s\n", repo.BaseUrl))
	}

	return ioutil.WriteFile(zypperRepoFile, buf.Bytes(), 0600)
}

func zypperInstalls(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.InstallZypperPackages(packages.Run, names)
}

func zypperRemovals(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.RemoveZypperPackages(packages.Run, names)
}

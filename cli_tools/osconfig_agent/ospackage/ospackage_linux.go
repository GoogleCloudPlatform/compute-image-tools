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

package ospackage

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

// SetConfig applies the configurations specified in the service.
func SetConfig(res *osconfigpb.LookupConfigsResponse) error {
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

func aptRepositories(repos []*osconfigpb.AptRepository) error {
	/*
		# Repo file managed by Google OSConfig agent
		deb http://repo1-url/ repo1 main
		deb http://repo1-url/ repo2 main contrib non-free
	*/
	var buf bytes.Buffer
	buf.WriteString("# Repo file managed by Google OSConfig agent\n")
	for _, repo := range repos {
		line := fmt.Sprintf("\ndeb %s %s", repo.Uri, repo.Distribution)
		for _, c := range repo.Components {
			line = fmt.Sprintf("%s %s", line, c)
		}
		buf.WriteString(line + "\n")
	}

	return ioutil.WriteFile(config.YumRepoFilePath(), buf.Bytes(), 0600)
}

func aptInstalls(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.InstallAptPackages(names)
}

func aptRemovals(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.RemoveAptPackages(names)
}

func yumRepositories(repos []*osconfigpb.YumRepository) error {
	/*
		# Repo file managed by Google OSConfig agent
		[repo1]
		name: repo1-name
		baseurl: https://repo1-url
		enabled=1
		gpgcheck=1
		repo_gpgcheck=1
		gpgkey=http://repo1-url/gpg
		[repo2]
		display_name: repo2-name
		baseurl: https://repo2-url
		enabled=1
		gpgcheck=1
		repo_gpgcheck=1
	*/
	var buf bytes.Buffer
	buf.WriteString("# Repo file managed by Google OSConfig agent\n")
	for _, repo := range repos {
		buf.WriteString(fmt.Sprintf("\n[%s]\n", repo.Id))
		if repo.DisplayName == "" {
			buf.WriteString(fmt.Sprintf("name: %s\n", repo.Id))
		} else {
			buf.WriteString(fmt.Sprintf("name: %s\n", repo.DisplayName))
		}
		buf.WriteString(fmt.Sprintf("baseurl: %s\n", repo.BaseUrl))
		buf.WriteString("enabled=1\ngpgcheck=1\nrepo_gpgcheck=1\n")
		if len(repo.GpgKeys) > 0 {
			buf.WriteString(fmt.Sprintf("gpgkey=%s\n", repo.GpgKeys[0]))
			for _, k := range repo.GpgKeys[1:] {
				buf.WriteString(fmt.Sprintf("       %s\n", k))
			}
		}
	}

	return ioutil.WriteFile(config.YumRepoFilePath(), buf.Bytes(), 0600)
}

func yumInstalls(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.InstallYumPackages(names)
}

func yumRemovals(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.RemoveYumPackages(names)
}

// TODO: Write repo_gpgcheck, pkg_gpgcheck, gpgkeys, type
func zypperRepositories(repos []*osconfigpb.ZypperRepository) error {
	/*
		# Repo file managed by Google OSConfig agent
		[repo1]
		name: repo1-name
		baseurl: https://repo1-url
		enabled=1
		gpgcheck=1
		repo_gpgcheck=1
		gpgkey=http://repo1-url/gpg
		[repo2]
		display_name: repo2-name
		baseurl: https://repo2-url
		enabled=1
		gpgcheck=1
		repo_gpgcheck=1
	*/
	var buf bytes.Buffer
	buf.WriteString("# Repo file managed by Google OSConfig agent\n")
	for _, repo := range repos {
		buf.WriteString(fmt.Sprintf("\n[%s]\n", repo.Id))
		if repo.DisplayName == "" {
			buf.WriteString(fmt.Sprintf("name: %s\n", repo.Id))
		} else {
			buf.WriteString(fmt.Sprintf("name: %s\n", repo.DisplayName))
		}
		buf.WriteString(fmt.Sprintf("baseurl: %s\n", repo.BaseUrl))
		buf.WriteString("enabled=1\ngpgcheck=1\nrepo_gpgcheck=1\n")
		if len(repo.GpgKeys) > 0 {
			buf.WriteString(fmt.Sprintf("gpgkey=%s\n", repo.GpgKeys[0]))
			for _, k := range repo.GpgKeys[1:] {
				buf.WriteString(fmt.Sprintf("       %s\n", k))
			}
		}
	}

	return ioutil.WriteFile(config.ZypperRepoFilePath(), buf.Bytes(), 0600)
}

func zypperInstalls(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.InstallZypperPackages(names)
}

func zypperRemovals(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.RemoveZypperPackages(names)
}

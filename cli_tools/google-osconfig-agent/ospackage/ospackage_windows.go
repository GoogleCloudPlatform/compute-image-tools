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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

// SetConfig applies the configurations specified in the service.
func SetConfig(res *osconfigpb.LookupConfigsResponse) error {
	var errs []string
	if res.Goo != nil && packages.GooGetExists {
		if err := gooRepositories(res.Goo.Repositories); err != nil {
			errs = append(errs, fmt.Sprintf("error writing GooGet repo file: %v", err))
		}
		if err := gooInstalls(res.Goo.PackageInstalls); err != nil {
			errs = append(errs, fmt.Sprintf("error installing GooGet packages: %v", err))
		}
		if err := gooRemovals(res.Goo.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error removing GooGet packages: %v", err))
		}
	}

	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

func gooRepositories(repos []*osconfigpb.GooRepository) error {
	/*
		# Repo file managed by Google OSConfig agent

		- name: repo1-name
		  url: https://repo1-url
		- name: repo1-name
		  url: https://repo2-url
	*/
	var buf bytes.Buffer
	buf.WriteString("# Repo file managed by Google OSConfig agent\n")
	for _, repo := range repos {
		buf.WriteString(fmt.Sprintf("\n- name: %s\n", repo.Name))
		buf.WriteString(fmt.Sprintf("  url: %s\n", repo.Url))
	}

	return ioutil.WriteFile(config.GoogetRepoFilePath(), buf.Bytes(), 0600)
}

func gooInstalls(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.InstallGooGetPackages(names)
}

func gooRemovals(pkgs []*osconfigpb.Package) error {
	if pkgs == nil {
		return nil
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return packages.RemoveGooGetPackages(names)
}

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
	"fmt"
	"io/ioutil"
	"log"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/internal/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/package_library"
)

func runPackageConfig(res *osconfig.LookupConfigsResponse) {
	if res.Goo != nil && packages.GooGetExists {
		gooRepositories(res.Goo.Repositories)
		gooInstalls(res.Goo.PackageInstalls)
		gooInstalls(res.Goo.PackageRemovals)
	}
}

func gooRepositories(repos []*osconfig.GooRepository) {
	/*
		Repo file managed by Google OSConfig agent

		- name: repo1-name
		  url: https://repo1-url
		- name: repo1-name
		  url: https://repo2-url
	*/
	repoContents := "Repo file managed by Google OSConfig agent"
	for _, repo := range repos {
		repoContents = fmt.Sprintf("%s\n- name: %s\n  url: %s", repoContents, repo.Name, repo.Url)
	}
	if err := ioutil.WriteFile("C:/ProgramData/GooGet/repos/google_osconfig.repo", []byte(repoContents), 0600); err != nil {
		log.Printf("Error writing repo file: %v", err)
	}
}

func gooInstalls(pkgs []*osconfig.Package) {
	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	if err := packages.InstallGooGetPackages(names); err != nil {
		log.Printf("Error installing packages: %v", err)
	}
}

func gooRemovals(pkgs []*osconfig.Package) {
	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	if err := packages.RemoveGooGetPackages(names); err != nil {
		log.Printf("Error removing packages: %v", err)
	}
}

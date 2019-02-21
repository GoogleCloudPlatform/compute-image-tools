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

// +build !public

package ospackage

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/inventory"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

func googetRepositories(repos []*osconfigpb.GooRepository, repoFile string) error {
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

	return writeIfChanged(buf.Bytes(), repoFile)
}

func googetChanges(packageInstalls, packageRemovals []*osconfigpb.Package) error {
	var errs []string

	inv := inventory.GetInventory()
	changes := getNecessaryChanges(inv.InstalledPackages.Apt, inv.PackageUpdates.Apt, packageInstalls, packageRemovals)

	if changes.packagesToInstall != nil {
		logger.Infof("Installing packages %s", changes.packagesToInstall)
		if err := packages.InstallGooGetPackages(changes.packagesToInstall); err != nil {
			logger.Errorf("Error installing googet packages: %v", err)
			errs = append(errs, fmt.Sprintf("error installing googet packages: %v", err))
		}
	}

	if changes.packagesToUpgrade != nil {
		logger.Infof("Upgrading packages %s", changes.packagesToUpgrade)
		if err := packages.InstallGooGetPackages(changes.packagesToUpgrade); err != nil {
			logger.Errorf("Error upgrading googet packages: %v", err)
			errs = append(errs, fmt.Sprintf("error upgrading googet packages: %v", err))
		}
	}

	if changes.packagesToRemove != nil {
		logger.Infof("Removing packages %s", changes.packagesToRemove)
		if err := packages.RemoveGooGetPackages(changes.packagesToRemove); err != nil {
			logger.Errorf("Error removing googet packages: %v", err)
			errs = append(errs, fmt.Sprintf("error removing googet packages: %v", err))
		}
	}

	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

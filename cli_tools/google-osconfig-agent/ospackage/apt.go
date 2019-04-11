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

package ospackage

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

var debArchiveTypeMap = map[osconfigpb.AptRepository_ArchiveType]string{
	osconfigpb.AptRepository_DEB:     "deb",
	osconfigpb.AptRepository_DEB_SRC: "deb-src",
}

func aptRepositories(repos []*osconfigpb.AptRepository, repoFile string) error {
	/*
		# Repo file managed by Google OSConfig agent
		deb http://repo1-url/ repo1 main
		deb http://repo1-url/ repo2 main contrib non-free
	*/
	var buf bytes.Buffer
	buf.WriteString("# Repo file managed by Google OSConfig agent\n")
	for _, repo := range repos {
		archiveType, ok := debArchiveTypeMap[repo.ArchiveType]
		if !ok {
			archiveType = "deb"
		}
		line := fmt.Sprintf("\n%s %s %s", archiveType, repo.Uri, repo.Distribution)
		for _, c := range repo.Components {
			line = fmt.Sprintf("%s %s", line, c)
		}
		buf.WriteString(line + "\n")
	}

	return writeIfChanged(buf.Bytes(), repoFile)
}

func aptChanges(packageInstalls, packageRemovals []*osconfigpb.Package) error {
	var errs []string

	installed, err := packages.InstalledDebPackages()
	if err != nil {
		return err
	}
	updates, err := packages.AptUpdates()
	if err != nil {
		return err
	}
	changes := getNecessaryChanges(installed, updates, packageInstalls, packageRemovals)

	if changes.packagesToInstall != nil {
		logger.Infof("Installing packages %s", changes.packagesToInstall)
		if err := packages.InstallAptPackages(changes.packagesToInstall); err != nil {
			logger.Errorf("Error installing apt packages: %v", err)
			errs = append(errs, fmt.Sprintf("error installing apt packages: %v", err))
		}
	}

	if changes.packagesToUpgrade != nil {
		logger.Infof("Upgrading packages %s", changes.packagesToUpgrade)
		if err := packages.InstallAptPackages(changes.packagesToUpgrade); err != nil {
			logger.Errorf("Error upgrading apt packages: %v", err)
			errs = append(errs, fmt.Sprintf("error upgrading apt packages: %v", err))
		}
	}

	if changes.packagesToRemove != nil {
		logger.Infof("Removing packages %s", changes.packagesToRemove)
		if err := packages.RemoveAptPackages(changes.packagesToRemove); err != nil {
			logger.Errorf("Error removing apt packages: %v", err)
			errs = append(errs, fmt.Sprintf("error removing apt packages: %v", err))
		}
	}

	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

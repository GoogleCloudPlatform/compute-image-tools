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

// Package ospackage configures OS packages based on osconfig API response.
package ospackage

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

// SetConfig applies the configurations specified in the service.
func SetConfig(res *osconfigpb.LookupConfigsResponse) error {
	var errs []string

	if res.Goo != nil && packages.GooGetExists {
		if err := googetRepositories(res.Goo.Repositories, config.GoogetRepoFilePath()); err != nil {
			logger.Errorf("Error writing googet repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing googet repo file: %v", err))
		}
		if err := googetChanges(res.Goo.PackageInstalls, res.Goo.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing googet changes: %v", err))
		}
	}

	if res.Apt != nil && packages.AptExists {
		if err := aptRepositories(res.Apt.Repositories, config.AptRepoFilePath()); err != nil {
			logger.Errorf("Error writing apt repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing apt repo file: %v", err))
		}
		if err := aptChanges(res.Apt.PackageInstalls, res.Apt.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing apt changes: %v", err))
		}
	}

	if res.Yum != nil && packages.YumExists {
		if err := yumRepositories(res.Yum.Repositories, config.YumRepoFilePath()); err != nil {
			logger.Errorf("Error writing yum repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing yum repo file: %v", err))
		}
		if err := yumChanges(res.Yum.PackageInstalls, res.Yum.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing yum changes: %v", err))
		}
	}

	if res.Zypper != nil && packages.ZypperExists {
		if err := zypperRepositories(res.Zypper.Repositories, config.ZypperRepoFilePath()); err != nil {
			logger.Errorf("Error writing zypper repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing zypper repo file: %v", err))
		}
		if err := zypperChanges(res.Zypper.PackageInstalls, res.Zypper.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing zypper changes: %v", err))
		}
	}

	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

func checksum(r io.Reader) hash.Hash {
	hash := sha256.New()
	io.Copy(hash, r)
	return hash
}

func writeIfChanged(content []byte, path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(content)
	h1 := checksum(reader)
	h2 := checksum(file)
	if bytes.Equal(h1.Sum(nil), h2.Sum(nil)) {
		file.Close()
		return nil
	}

	logger.Infof("Writing repo file %s with updated contents", path)
	if _, err := file.WriteAt(content, 0); err != nil {
		file.Close()
		return err
	}

	return file.Close()
}

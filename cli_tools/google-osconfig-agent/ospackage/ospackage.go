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
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/api/option"
	"google.golang.org/grpc/status"
)

var dump = &pretty.Config{IncludeUnexported: true}

func run(ctx context.Context, res string) {
	client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
	if err != nil {
		logger.Errorf("osconfig.NewClient Error: %v", err)
		return
	}

	resp, err := lookupConfigs(ctx, client, res)
	if err != nil {
		logger.Errorf("LookupConfigs error: %v", err)
		return
	}

	// We don't check the error from ospackage.SetConfig as all errors are already logged.
	setConfig(resp)
}

// Run looks up osconfigs and applies them using tasker.Enqueue.
func Run(ctx context.Context, res string) {
	tasker.Enqueue("Run OSPackage", func() { run(ctx, res) })
}

func lookupConfigs(ctx context.Context, client *osconfig.Client, resource string) (*osconfigpb.LookupConfigsResponse, error) {
	info, err := osinfo.GetDistributionInfo()
	if err != nil {
		return nil, err
	}

	req := &osconfigpb.LookupConfigsRequest{
		Resource: resource,
		OsInfo: &osconfigpb.LookupConfigsRequest_OsInfo{
			OsLongName:     info.LongName,
			OsShortName:    info.ShortName,
			OsVersion:      info.Version,
			OsKernel:       info.Kernel,
			OsArchitecture: info.Architecture,
		},
		ConfigTypes: []osconfigpb.LookupConfigsRequest_ConfigType{
			osconfigpb.LookupConfigsRequest_GOO,
			osconfigpb.LookupConfigsRequest_WINDOWS_UPDATE,
			osconfigpb.LookupConfigsRequest_APT,
			osconfigpb.LookupConfigsRequest_YUM,
			osconfigpb.LookupConfigsRequest_ZYPPER,
		},
	}
	logger.Debugf("LookupConfigs request:\n%s\n\n", dump.Sprint(req))

	res, err := client.LookupConfigs(ctx, req)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, fmt.Errorf("code: %q, message: %q, details: %q", s.Code(), s.Message(), s.Details())
		}
		return nil, err
	}
	logger.Debugf("LookupConfigs response:\n%s\n\n", dump.Sprint(res))

	return res, nil
}

func setConfig(res *osconfigpb.LookupConfigsResponse) error {
	var errs []string
	if res.Goo != nil && packages.GooGetExists {
		if _, err := os.Stat(config.GoogetRepoFilePath()); os.IsNotExist(err) {
			logger.Debugf("Repo file does not exist, will create one...")
			if err := os.MkdirAll(filepath.Dir(config.GoogetRepoFilePath()), 07550); err != nil {
				logger.Errorf("Error creating repo file: %v", err)
				errs = append(errs, fmt.Sprintf("error creating googet repo file: %v", err))
			}
		}
		if err := googetRepositories(res.Goo.Repositories, config.GoogetRepoFilePath()); err != nil {
			logger.Errorf("Error writing googet repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing googet repo file: %v", err))
		}
		if err := googetChanges(res.Goo.PackageInstalls, res.Goo.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing googet changes: %v", err))
		}
	}

	if res.Apt != nil && packages.AptExists {
		if _, err := os.Stat(config.AptRepoFilePath()); os.IsNotExist(err) {
			logger.Debugf("Repo file does not exist, will create one...")
			if err := os.MkdirAll(filepath.Dir(config.AptRepoFilePath()), 07550); err != nil {
				logger.Errorf("Error creating repo file: %v", err)
				errs = append(errs, fmt.Sprintf("error creating apt repo file: %v", err))
			}
		}
		if err := aptRepositories(res.Apt.Repositories, config.AptRepoFilePath()); err != nil {
			logger.Errorf("Error writing apt repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing apt repo file: %v", err))
		}
		if err := aptChanges(res.Apt.PackageInstalls, res.Apt.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing apt changes: %v", err))
		}
	}

	if res.Yum != nil && packages.YumExists {
		if _, err := os.Stat(config.YumRepoFilePath()); os.IsNotExist(err) {
			logger.Debugf("Repo file does not exist, will create one...")
			if err := os.MkdirAll(filepath.Dir(config.YumRepoFilePath()), 07550); err != nil {
				logger.Errorf("Error creating repo file: %v", err)
				errs = append(errs, fmt.Sprintf("error creating yum repo file: %v", err))
			}
		}
		if err := yumRepositories(res.Yum.Repositories, config.YumRepoFilePath()); err != nil {
			logger.Errorf("Error writing yum repo file: %v", err)
			errs = append(errs, fmt.Sprintf("error writing yum repo file: %v", err))
		}
		if err := yumChanges(res.Yum.PackageInstalls, res.Yum.PackageRemovals); err != nil {
			errs = append(errs, fmt.Sprintf("error performing yum changes: %v", err))
		}
	}

	if res.Zypper != nil && packages.ZypperExists {
		if _, err := os.Stat(config.ZypperRepoFilePath()); os.IsNotExist(err) {
			logger.Debugf("Repo file does not exist, will create one...")
			if err := os.MkdirAll(filepath.Dir(config.ZypperRepoFilePath()), 07550); err != nil {
				logger.Errorf("Error creating repo file: %v", err)
				errs = append(errs, fmt.Sprintf("error creating zypper repo file: %v", err))
			}
		}
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

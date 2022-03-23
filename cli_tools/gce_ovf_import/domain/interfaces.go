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

package domain

import (
	"context"

	"github.com/vmware/govmomi/ovf"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

// To rebuild mocks, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package ovfdomainmocks -source $GOFILE -destination mocks/mock_interfaces.go

// OvfDescriptorValidatorInterface represents OVF descriptor validator
type OvfDescriptorValidatorInterface interface {
	ValidateOvfPackage(ovfDescriptor *ovf.Envelope, ovfGcsPath string) (*ovf.Envelope, error)
}

// OvfDescriptorLoaderInterface represents a loader for OVF descriptors
type OvfDescriptorLoaderInterface interface {
	Load(ovfGcsPath string) (*ovf.Envelope, error)
}

// MachineTypeProviderInterface is responsible for providing GCE machine type
type MachineTypeProviderInterface interface {
	GetMachineType() (string, error)
}

// DiskImporterInterface imports a GCE PD from a virtual disk file (e.g. vmdk).
type DiskImporterInterface interface {
	Import(context.Context, importer.ImageImportRequest, logging.Logger) (string, error)
}

// MultiDiskImporterInterface imports multiple disks files simultaneously, returning
// a list of disks resource URIs. The ordering of the returned list matches the ordering
// of fileURIs.
//
// If an import fails, all running imports will be cancelled, and disk from finished disks
// will be deleted.
type MultiDiskImporterInterface interface {
	Import(ctx context.Context, params *OVFImportParams, fileURIs []string) (disks []domain.Disk, err error)
}

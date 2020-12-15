//  Copyright 2020 Google Inc. All Rights Reserved.
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

package ovfexportdomain

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"google.golang.org/api/compute/v1"
)

// InstanceDisksExporter exports disks of an instance
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_instance_disks_exporter.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain InstanceDisksExporter
type InstanceDisksExporter interface {
	Export(*compute.Instance, *OVFExportArgs) ([]*ExportedDisk, error)
	TraceLogs() []string
	Cancel(reason string) bool
}

// InstanceExportCleaner cleans a Compute Engine instance after export into OVF
// by re-attaching disks and starting it up if it was already started.
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_instance_export_cleaner.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain InstanceExportCleaner
type InstanceExportCleaner interface {
	Clean(instance *compute.Instance, params *OVFExportArgs) error
	TraceLogs() []string
	Cancel(reason string) bool
}

// InstanceExportPreparer prepares a Compute Engine instance for export into OVF
// by shutting it down and detaching disks
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_instance_export_preparer.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain InstanceExportPreparer
type InstanceExportPreparer interface {
	Prepare(*compute.Instance, *OVFExportArgs) error
	TraceLogs() []string
	Cancel(reason string) bool
}

// OvfManifestGenerator generates a manifest file for all files under a Cloud
// Storage path and writes it to a file under the same path
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_ovf_manifest_generator.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain OvfManifestGenerator
type OvfManifestGenerator interface {
	GenerateAndWriteToGCS(gcsPath, manifestFileName string) error
	Cancel(reason string) bool
}

// OvfDescriptorGenerator is responsible for generating OVF descriptor based on
// GCE instance being exported.
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_ovf_descriptor_generator.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain OvfDescriptorGenerator
type OvfDescriptorGenerator interface {
	GenerateAndWriteOVFDescriptor(instance *compute.Instance, exportedDisks []*ExportedDisk, bucketName, gcsDirectoryPath string, diskInspectionResult *pb.InspectionResults) error
	Cancel(reason string) bool
}

// OvfExportParamValidator validate params for OVF export
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_ovf_export_param_validator.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain OvfExportParamValidator
type OvfExportParamValidator interface {
	ValidateAndParseParams(*OVFExportArgs) error
}

// OvfExportParamPopulator populates params for OVF export
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -destination ../../mocks/mock_ovf_export_param_populator.go github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain OvfExportParamPopulator
type OvfExportParamPopulator interface {
	Populate(*OVFExportArgs) error
}

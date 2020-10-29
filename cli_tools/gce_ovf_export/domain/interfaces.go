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
	commondisk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"google.golang.org/api/compute/v1"
)

// InstanceDisksExporter exports disks of an instance
type InstanceDisksExporter interface {
	Export(*compute.Instance, *OVFExportParams) ([]*ExportedDisk, error)
	TraceLogs() []string
	Cancel(reason string) bool
}

// InstanceExportCleaner cleans a Compute Engine instance after export into OVF
// by re-attaching disks and starting it up if it was already started.
type InstanceExportCleaner interface {
	Clean(*compute.Instance, *OVFExportParams) error
	TraceLogs() []string
	Cancel(reason string) bool
}

// InstanceExportPreparer prepares a Compute Engine instance for export into OVF
// by shutting it down and detaching disks
type InstanceExportPreparer interface {
	Prepare(*compute.Instance, *OVFExportParams) error
	TraceLogs() []string
	Cancel(reason string) bool
}

// OvfManifestGenerator generates a manifest file for all files under a Cloud
// Storage path and writes it to a file under the same path
type OvfManifestGenerator interface {
	GenerateAndWriteToGCS(gcsPath, manifestFileName string) error
	Cancel(reason string) bool
}

// OvfDescriptorGenerator is responsible for generating OVF descriptor based on
//GCE instance being exported.
type OvfDescriptorGenerator interface {
	GenerateAndWriteOVFDescriptor(instance *compute.Instance, exportedDisks []*ExportedDisk, bucketName, gcsDirectoryPath string, diskInspectionResult *commondisk.InspectionResult) error
	Cancel(reason string) bool
}

// OvfExportParamValidator validate params for OVF export
type OvfExportParamValidator interface {
	ValidateAndParseParams(params *OVFExportParams) error
}

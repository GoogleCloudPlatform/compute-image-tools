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

package ovfutils

import (
	"bytes"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/vmware/govmomi/ovf"
)

// OvfDescriptorLoader is responsible for loading OVF descriptor from a GCS directory path.
type OvfDescriptorLoader struct {
	storageClient commondomain.StorageClientInterface
	validator     domain.OvfDescriptorValidatorInterface
}

// NewOvfDescriptorLoader creates new OvfDescriptorLoader
func NewOvfDescriptorLoader(sc commondomain.StorageClientInterface) *OvfDescriptorLoader {
	return &OvfDescriptorLoader{
		storageClient: sc,
		validator:     NewOvfValidator(sc)}
}

// Load finds and loads OVF descriptor from a GCS directory path.
// ovfGcsPath is a path to OVF directory, not a path to OVF descriptor file itself.
func (l *OvfDescriptorLoader) Load(ovfGcsPath string) (*ovf.Envelope, error) {
	ovfDescriptorGcsReference, err := l.storageClient.FindGcsFile(ovfGcsPath, ".ovf")
	if err != nil {
		return nil, err
	}
	descriptorContent, err := l.storageClient.GetGcsFileContent(ovfDescriptorGcsReference)
	if err != nil {
		return nil, err
	}
	descriptorReader := bytes.NewReader(descriptorContent)
	ovfDescriptor, err := ovf.Unmarshal(descriptorReader)
	if err != nil {
		return nil, err
	}

	return l.validator.ValidateOvfPackage(ovfDescriptor, ovfGcsPath)
}

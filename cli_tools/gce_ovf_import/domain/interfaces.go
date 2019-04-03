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

import "github.com/GoogleCloudPlatform/compute-image-tools/third_party/govmomi/ovf"

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

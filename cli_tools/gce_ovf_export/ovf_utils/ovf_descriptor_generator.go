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

package ovfutils

import (
	"bytes"
	"encoding/xml"

	"github.com/vmware/govmomi/ovf"
)

// OvfDescriptorGenerator is responsible for generating OVF descriptor based on GCE instance being exported
type OvfDescriptorGenerator struct {
}

// Load finds and loads OVF descriptor from a GCS directory path.
// ovfGcsPath is a path to OVF directory, not a path to OVF descriptor file itself.
func (g *OvfDescriptorGenerator) Generate(exportedDisksGCSPaths []string) (*ovf.Envelope, error) {
	descriptor := &ovf.Envelope{}

	//TODO

	return descriptor, nil
}

func Marshal(descriptor *ovf.Envelope) (string, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	err := enc.Encode(&descriptor)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

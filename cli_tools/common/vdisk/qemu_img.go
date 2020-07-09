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

package vdisk

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/files"
)

// Format is an enum type for representing VM disk encoding formats.
type Format string

const (
	// FormatUnknown means that qemu-img could not determine the file's format.
	FormatUnknown Format = "unknown"
	// FormatVMDK means that the file uses the vmdk file format.
	FormatVMDK Format = "vmdk"
	// FormatVHDX means that the file uses the vhdx file format.
	FormatVHDX Format = "vhdx"
	// FormatVPC means that the file uses the vpc file format.
	FormatVPC Format = "vpc"
	// FormatVDI means that the file uses the vdi file format.
	FormatVDI Format = "vdi"
	// FormatQCOW2 means that the file uses the qcow2 file format.
	FormatQCOW2 Format = "qcow2"
	// FormatRAW means that the file uses the raw file format.
	FormatRAW Format = "raw"
)

var formats = map[string]Format{
	"vmdk":  FormatVMDK,
	"vhdx":  FormatVHDX,
	"vpc":   FormatVPC,
	"vdi":   FormatVDI,
	"qcow2": FormatQCOW2,
	"raw":   FormatRAW,
}

// ImageInfo includes metadata returned by `qemu-img info`.
type ImageInfo struct {
	Format           Format
	ActualSizeBytes  int64
	VirtualSizeBytes int64
}

// InfoClient runs `qemu-img info` and returns the results.
type InfoClient interface {
	GetInfo(ctx context.Context, filename string) (ImageInfo, error)
}

// NewInfoClient returns a new instance of InfoClient.
func NewInfoClient() InfoClient {
	return defaultInfoClient{}
}

type defaultInfoClient struct{}

func (client defaultInfoClient) GetInfo(ctx context.Context, filename string) (info ImageInfo, err error) {
	if !files.Exists(filename) {
		return info, fmt.Errorf("file %q not found", filename)
	}
	cmd := exec.CommandContext(ctx, "qemu-img", "info", "--output=json", filename)
	out, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf("inspection failure: %v, stderr: %s", err,
			err.(*exec.ExitError).Stderr)
	}
	jsonTemplate := struct {
		Filename         string `json:"filename"`
		Format           string `json:"format"`
		ActualSizeBytes  int64  `json:"actual-size"`
		VirtualSizeBytes int64  `json:"virtual-size"`
	}{}
	if err = json.Unmarshal(out, &jsonTemplate); err != nil {
		return info, fmt.Errorf("failed to inspect %q: %w", filename, err)
	}
	return ImageInfo{
		Format:           lookupFileFormat(jsonTemplate.Format),
		ActualSizeBytes:  jsonTemplate.ActualSizeBytes,
		VirtualSizeBytes: jsonTemplate.VirtualSizeBytes,
	}, nil
}

func lookupFileFormat(s string) Format {
	format := formats[strings.ToLower(s)]
	if format != "" {
		return format
	}
	return FormatUnknown
}

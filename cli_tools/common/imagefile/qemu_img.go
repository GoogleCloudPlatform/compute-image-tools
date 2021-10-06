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

package imagefile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/files"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// FormatUnknown means that qemu-img could not determine the file's format.
const FormatUnknown string = "unknown"

// The output of `qemu-img --help` contains this list.
var qemuImgFormats = strings.Split("blkdebug blklogwrites blkreplay blkverify bochs cloop "+
	"copy-on-read dmg file ftp ftps gluster host_cdrom host_device http "+
	"https iscsi iser luks nbd nfs null-aio null-co nvme parallels qcow "+
	"qcow2 qed quorum raw rbd replication sheepdog ssh throttle vdi vhdx vmdk vpc vvfat", " ")

// ImageInfo includes metadata returned by `qemu-img info`.
type ImageInfo struct {
	Format           string
	ActualSizeBytes  int64
	VirtualSizeBytes int64
	Checksum         string
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

type fileInfoJsonTemplate struct {
Filename         string `json:"filename"`
Format           string `json:"format"`
ActualSizeBytes  int64  `json:"actual-size"`
VirtualSizeBytes int64  `json:"virtual-size"`
}

func (client defaultInfoClient) GetInfo(ctx context.Context, filename string) (info ImageInfo, err error) {
	if !files.Exists(filename) {
		err = fmt.Errorf("file %q not found", filename)
		return
	}

	// To reduce the runtime permissions used on the inflation worker, we pre-allocate
	// disks sufficient to hold the disk file and the inflated disk. The scratch disk gets
	// a padding factor to account for filesystem overhead. This info is required,
	// so we have to terminate the import if it's failed to fetch file info.
	jsonTemplate, err := client.getFileInfo(ctx, filename)
	if err != nil {
		err = daisy.Errf("Failed to inspect file %v", filename)
		return
	}

	// To ensure disk.insert API produced expected disk content from a VMDK file, we need
	// to calculate the checksum from qemu output for comparison. This check is required,
	// so we have to terminate the import if it's failed to calculate the checksum.
	checksum, err := client.getFileChecksum(ctx, filename, info.VirtualSizeBytes)
	if err != nil {
		err = daisy.Errf("Failed to calculate file '%v' checksum by qemu: %v", filename, err)
		return
	}

	info.Format = lookupFileFormat(jsonTemplate.Format)
	info.ActualSizeBytes = jsonTemplate.ActualSizeBytes
	info.VirtualSizeBytes = jsonTemplate.VirtualSizeBytes
	info.Checksum = checksum
	return
}

func (client defaultInfoClient) getFileInfo(ctx context.Context, filename string) (*fileInfoJsonTemplate, error) {
	cmd := exec.CommandContext(ctx, "qemu-img", "info", "--output=json", filename)
	out, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, daisy.Errf("inspection failure: %w, stderr: %s", err,
					exitError.Stderr)
		}
		return nil, daisy.Errf("inspection failure: %w", err)
	}
	jsonTemplate := fileInfoJsonTemplate{}
	if err = json.Unmarshal(out, &jsonTemplate); err != nil {
		return nil, daisy.Errf("failed to inspect %q: %w", filename, err)
	}
	return &jsonTemplate, err
}

func (client defaultInfoClient) getFileChecksum(ctx context.Context, filename string, virtualSizeBytes int64) (string, error) {
	// Check size = 200000*512 = 100MB
	checkCount := int64(200000)
	blockSize := int64(512)
	blockCount := virtualSizeBytes / blockSize
	skips := []int64{0, int64(2000000) - checkCount, int64(20000000) - checkCount, blockCount - checkCount}
	checksum := ""

	for _, skip := range skips {
		cmd := exec.CommandContext(ctx, "qemu-img",
			fmt.Sprintf("dd if=%v of=out1 bs=%v count=%v skip=%v", filename, blockSize, skip+checkCount, skip))
		out, err := cmd.Output()
		if err != nil {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return "", daisy.Errf("inspection for checksum failure: %w, stderr: %s", err,
					exitError.Stderr)
			}
			return "nil", daisy.Errf("inspection for checksum failure: %w", err)
		}

		if checksum != "" {
			checksum += "-"
		}
		checksum += string(out)
	}
	return checksum, nil
}


func lookupFileFormat(s string) string {
	lower := strings.ToLower(s)
	for _, format := range qemuImgFormats {
		if format == lower {
			return format
		}
	}
	return FormatUnknown
}

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
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/files"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
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
	// This checksum is calculated from the partial disk content extracted by QEMU.
	Checksum string
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

	jsonTemplate, err := client.getFileInfo(ctx, filename)
	if err != nil {
		err = daisy.Errf("Failed to inspect file %v: %v", filename, err)
		return
	}
	info.Format = lookupFileFormat(jsonTemplate.Format)
	info.ActualSizeBytes = jsonTemplate.ActualSizeBytes
	info.VirtualSizeBytes = jsonTemplate.VirtualSizeBytes

	checksum, err := client.getFileChecksum(ctx, filename, info.VirtualSizeBytes)
	if err != nil {
		err = daisy.Errf("Failed to calculate file '%v' checksum by qemu: %v", filename, err)
		return
	}

	info.Checksum = checksum
	return
}

func (client defaultInfoClient) getFileInfo(ctx context.Context, filename string) (*fileInfoJsonTemplate, error) {
	cmd := exec.CommandContext(ctx, "qemu-img", "info", "--output=json", filename)
	out, err := cmd.Output()
	err = constructCmdErr(string(out), err, "inspection failure")
	if err != nil {
		return nil, err
	}

	jsonTemplate := fileInfoJsonTemplate{}
	if err = json.Unmarshal(out, &jsonTemplate); err != nil {
		return nil, daisy.Errf("failed to inspect %q: %w", filename, err)
	}
	return &jsonTemplate, err
}

func (client defaultInfoClient) getFileChecksum(ctx context.Context, filename string, virtualSizeBytes int64) (checksum string, err error) {
	// We calculate 4 chunks' checksum. Each of them is 100MB: 0~100MB, 0.9GB~1GB, 9.9GB~10GB, the last 100MB.
	// It is align with what we did for "daisy_workflows/image_import/import_image.sh" so that we can compare them.
	// Each block size is 512 Bytes. So, we need to check 20000 blocks: 200000 * 512 Bytes = 100MB
	// "skips" is also the start point of each chunks.
	checkBlockCount := int64(200000)
	blockSize := int64(512)
	totalBlockCount := virtualSizeBytes / blockSize
	skips := []int64{0, int64(2000000) - checkBlockCount, int64(20000000) - checkBlockCount, totalBlockCount - checkBlockCount}
	tmpOutFilePrefix := "out" + pathutils.RandString(5)
	for i, skip := range skips {
		tmpOutFileName := fmt.Sprintf("%v%v", tmpOutFilePrefix, i)
		defer os.Remove(tmpOutFileName)

		if skip < 0 {
			skip = 0
		}

		// Write 100MB data to a file.
		cmd := exec.CommandContext(ctx, "qemu-img", "dd", fmt.Sprintf("if=%v", filename),
			fmt.Sprintf("of=%v", tmpOutFileName), fmt.Sprintf("bs=%v", blockSize),
			fmt.Sprintf("count=%v", skip+checkBlockCount), fmt.Sprintf("skip=%v", skip))
		var out []byte
		out, err = cmd.Output()
		err = constructCmdErr(string(out), err, "inspection for checksum failure")
		if err != nil {
			return
		}

		// Calculate checksum for the 100MB file.
		f, fileErr := os.Open(tmpOutFileName)
		if err != nil {
			err = daisy.Errf("Failed to open file '%v' for QEMU md5 checksum calculation: %v", tmpOutFileName, fileErr)
			return
		}
		defer f.Close()
		h := md5.New()
		if _, md5Err := io.Copy(h, f); err != nil {
			err = daisy.Errf("Failed to copy data from file '%v' for QEMU md5 checksum calculation: %v", tmpOutFileName, md5Err)
			return
		}
		newChecksum := fmt.Sprintf("%x", h.Sum(nil))

		if checksum != "" {
			checksum += "-"
		}
		checksum += newChecksum
	}
	return
}

func constructCmdErr(out string, err error, errorFormat string) error {
	if err == nil {
		return nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return daisy.Errf("%v: '%w', stderr: '%s', out: '%s'", errorFormat, err, exitError.Stderr, out)
	}
	return daisy.Errf("%v: '%w', out: '%s'", errorFormat, err, out)
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

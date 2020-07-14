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
	"io/ioutil"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
)

const bytesPerMB = int64(1024 * 1024)

func TestGetInfo_FormatDetection(t *testing.T) {
	skipIfQemuImgNotInstalled(t)

	cases := []struct {
		filename              string
		size                  string
		format                string
		expectedVirtualSizeGB int64
	}{
		{
			filename:              "test.vmdk",
			size:                  "50G",
			format:                "vmdk",
			expectedVirtualSizeGB: 50 * bytesPerGB,
		},

		{
			filename:              "test.vhdx",
			size:                  "500M",
			format:                "vhdx",
			expectedVirtualSizeGB: 500 * bytesPerMB,
		},

		{
			filename:              "test.vpc",
			size:                  "10M",
			format:                "vpc",
			expectedVirtualSizeGB: 10 * bytesPerMB,
		},

		{
			filename:              "test.vdi",
			size:                  "1G",
			format:                "vdi",
			expectedVirtualSizeGB: bytesPerGB,
		},

		{
			filename:              "test.qcow2",
			size:                  "2G",
			format:                "qcow2",
			expectedVirtualSizeGB: 2 * bytesPerGB,
		},

		{
			filename:              "test.raw",
			size:                  "4G",
			format:                "raw",
			expectedVirtualSizeGB: 4 * bytesPerGB,
		},
	}

	client := NewInfoClient()

	for _, tt := range cases {
		t.Run(tt.filename, func(t *testing.T) {
			// 1. Create temp dir
			dir, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			absPath := path.Join(dir, tt.filename)

			// 2. Create image in temp dir
			cmd := exec.Command("qemu-img", "create", absPath, "-f", tt.format, tt.size)
			_, err = cmd.Output()
			assert.NoError(t, err)

			// 3. Run inspection, and verify results
			imageInfo, err := client.GetInfo(context.Background(), absPath)
			assert.NoError(t, err)
			assert.Equal(t, tt.format, imageInfo.Format)
			// Testing to the nearest GB, since that's what the GCP APIs use, and
			// because some image formats have additional overhead such that
			// the virtual size doesn't match the requested size in qemu-img create.
			assert.Equal(t, tt.expectedVirtualSizeGB/bytesPerGB, imageInfo.VirtualSizeBytes/bytesPerGB)
		})
	}
}

func TestGetInfo_ReturnErrorWhenImageNotFound(t *testing.T) {
	skipIfQemuImgNotInstalled(t)
	client := NewInfoClient()
	_, err := client.GetInfo(context.Background(), "/zz/garbage")
	assert.EqualError(t, err, "file \"/zz/garbage\" not found")
}

func TestGetInfo_PropagateQemuImgError(t *testing.T) {
	skipIfQemuImgNotInstalled(t)
	client := NewInfoClient()
	_, err := client.GetInfo(context.Background(), "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "qemu-img: Could not open '/tmp': A regular file")
}

func TestGetInfo_InspectionClassifiesCompressedAsRaw(t *testing.T) {
	skipIfQemuImgNotInstalled(t)
	tempFile, err := ioutil.TempFile("", "")
	assert.NoError(t, err)

	_, err = tempFile.WriteString(test.CreateCompressedFile())
	assert.NoError(t, err)
	assert.NoError(t, tempFile.Close())

	client := NewInfoClient()
	info, err := client.GetInfo(context.Background(), tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "raw", info.Format)
}

func TestLookupFileFormat_ReturnsUnknown_WhenFormatNotFound(t *testing.T) {
	assert.Equal(t, "unknown", lookupFileFormat("not-found"))
}

func TestLookupFileFormat_PerformsCaseInsensitiveSearch(t *testing.T) {
	assert.Equal(t, "vmdk", lookupFileFormat("VmDK"))
}

func skipIfQemuImgNotInstalled(t *testing.T) {
	cmd := exec.Command("qemu-img", "--version")
	_, err := cmd.Output()
	if err != nil {
		t.Skipf("Skipping since qemu-img is not installed %v", err)
	}
}

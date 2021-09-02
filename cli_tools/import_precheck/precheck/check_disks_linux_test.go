//  Copyright 2017 Google Inc. All Rights Reserved.
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

package precheck

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/mount"

	"github.com/stretchr/testify/assert"
)

type disksCheckTest struct {
	name           string
	mountInfo      mount.InspectionResults
	inspectError   error
	byteTrailer    []byte
	expectAllLogs  []string
	expectedStatus Result
}

func TestDisksCheck_Inspector(t *testing.T) {
	for _, tc := range []disksCheckTest{
		{
			name: "pass if boot device is non virtual",
			mountInfo: mount.InspectionResults{
				BlockDevicePath:        "/dev/sda1",
				BlockDeviceIsVirtual:   false,
				UnderlyingBlockDevices: []string{"/dev/sda"},
			},
			expectAllLogs: []string{
				"INFO: root filesystem mounted on /dev/sda1",
			},
			expectedStatus: Passed,
		}, {
			name: "pass if boot device is virtual with one underlying device",
			mountInfo: mount.InspectionResults{
				BlockDevicePath:        "/dev/mapper/vg-lv",
				BlockDeviceIsVirtual:   true,
				UnderlyingBlockDevices: []string{"/dev/sda"},
			},
			expectAllLogs: []string{
				"INFO: root filesystem mounted on /dev/mapper/vg-lv",
			},
			expectedStatus: Passed,
		}, {
			name: "fail if boot device is virtual with multiple underlying devices",
			mountInfo: mount.InspectionResults{
				BlockDevicePath:        "/dev/mapper/vg-lv",
				BlockDeviceIsVirtual:   true,
				UnderlyingBlockDevices: []string{"/dev/sda", "/dev/sdb"},
			},
			expectAllLogs: []string{
				"FATAL: root filesystem spans multiple block devices (/dev/sda, /dev/sdb). Typically this occurs when an LVM " +
					"logical volume spans multiple block devices. Image import only supports single block device.",
			},
			expectedStatus: Failed,
		}, {
			name:         "fail if inspect fails",
			inspectError: errors.New("failed to find root device"),
			expectAllLogs: []string{
				"WARN: Failed to inspect the boot disk. Prior to importing, verify that the boot disk " +
					"contains the root filesystem, and that the root filesystem isn't virtualized " +
					"over multiple disks (using LVM, for example).",
			},
			expectedStatus: Unknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.byteTrailer = []byte{'G', 'R', 'U', 'B', 0x55, 0xAA}
			runDisksCheck(t, tc)
		})
	}
}

func TestDisksCheck_MBR(t *testing.T) {
	for _, tc := range []disksCheckTest{
		{
			name: "happy case",
			expectAllLogs: []string{
				"INFO: boot disk has an MBR partition table",
				"INFO: GRUB found in MBR",
			},
			expectedStatus: Passed,
			byteTrailer:    []byte{'G', 'R', 'U', 'B', 0x55, 0xAA},
		}, {
			name: "fail if GRUB not in first 512 bytes",
			expectAllLogs: []string{
				"INFO: boot disk has an MBR partition table",
				"FATAL: GRUB not detected in MBR",
			},
			expectedStatus: Failed,
			byteTrailer:    []byte{0x55, 0xAA},
		}, {
			name: "fail if 0x55 not at 510",
			expectAllLogs: []string{
				"FATAL: boot disk does not have an MBR partition table",
				"FATAL: GRUB not detected in MBR",
			},
			expectedStatus: Failed,
			byteTrailer:    []byte{0xAA},
		}, {
			name: "fail if 0xAA not at 511",
			expectAllLogs: []string{
				"FATAL: boot disk does not have an MBR partition table",
				"FATAL: GRUB not detected in MBR",
			},
			expectedStatus: Failed,
			byteTrailer:    []byte{0x55},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.mountInfo = mount.InspectionResults{
				BlockDevicePath:        "/dev/mapper/vg-lv",
				BlockDeviceIsVirtual:   true,
				UnderlyingBlockDevices: []string{"/dev/sda"},
			}
			runDisksCheck(t, tc)
		})
	}
}

func runDisksCheck(t *testing.T, tc disksCheckTest) {
	t.Helper()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMountInspector := mount.NewMockInspector(ctrl)
	mockMountInspector.EXPECT().Inspect("/").Return(
		tc.mountInfo, tc.inspectError)
	report, err := (&DisksCheck{
		getMBROverride: func(devName string) ([]byte, error) {
			assert.Len(t, tc.mountInfo.UnderlyingBlockDevices, 1, "Only check grub when there's a single block device")
			assert.Equal(t, tc.mountInfo.UnderlyingBlockDevices[0], devName)
			bytes := make([]byte, 512)
			for i := range tc.byteTrailer {
				bytes[len(bytes)-len(tc.byteTrailer)+i] = tc.byteTrailer[i]
			}
			return bytes, nil
		},
		inspector: mockMountInspector,
	}).Run()
	if err != nil {
		t.Fatal(err)
	}
	for _, expectedLog := range tc.expectAllLogs {
		assert.Contains(t, report.logs, expectedLog)
	}

	assert.Equal(t, tc.expectedStatus.String(), report.result.String())
}

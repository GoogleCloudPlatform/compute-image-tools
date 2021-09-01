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
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readMounts(t *testing.T) {
	for _, tc := range []struct {
		lsblkOutputFile         string
		expectedPhysicalDevices map[string][]string
		expectedListMounts      []string
	}{
		{
			lsblkOutputFile:    "lsblk-one-partition.json",
			expectedListMounts: []string{"/"},
			expectedPhysicalDevices: map[string][]string{
				"/": []string{"sda"},
			},
		}, {
			lsblkOutputFile:    "lsblk-multi-disk-lvm.json",
			expectedListMounts: []string{"/", "/boot", "/mnt/tmp"},
			expectedPhysicalDevices: map[string][]string{
				"/":        []string{"sda", "sdb"},
				"/boot":    []string{"sda"},
				"/mnt/tmp": []string{"sdc"},
			},
		}, {
			lsblkOutputFile:    "lsblk-efi.json",
			expectedListMounts: []string{"/", "/boot/efi"},
			expectedPhysicalDevices: map[string][]string{
				"/":         []string{"sda"},
				"/boot/efi": []string{"sda"},
			},
		}, {
			lsblkOutputFile:    "lsblk-single-disk-lvm.json",
			expectedListMounts: []string{"/", "/boot", "/boot/efi", "[SWAP]"},
			expectedPhysicalDevices: map[string][]string{
				"/":         []string{"sda"},
				"/boot":     []string{"sda"},
				"/boot/efi": []string{"sda"},
				"[SWAP]":    []string{"sda"},
			},
		},
	} {
		t.Run(tc.lsblkOutputFile, func(t *testing.T) {
			actual, err := (&DisksCheck{
				lsblkOverride: func() ([]byte, error) {
					return ioutil.ReadFile("testdata/" + tc.lsblkOutputFile)
				},
			}).readMounts()
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.expectedListMounts, actual.listMountedDirectories())
			for mountDir, expectedPhysicalDevices := range tc.expectedPhysicalDevices {
				assert.Equal(t, expectedPhysicalDevices, actual.listPhysicalDevicesForMount(mountDir))
			}
		})
	}
}

func Test_run(t *testing.T) {
	for _, tc := range []struct {
		lsblkOutputFile string
		expectAllLogs   []string
		expectToPass    bool
	}{
		{
			lsblkOutputFile: "lsblk-one-partition.json",
			expectAllLogs: []string{
				"INFO: root filesystem found on device: sda",
			},
			expectToPass: true,
		},
		{
			lsblkOutputFile: "lsblk-multi-disk-lvm.json",
			expectAllLogs: []string{
				"FATAL: root filesystem spans multiple block devices ([sda sdb]). Typically this occurs when an LVM " +
					"logical volume spans multiple block devices. Image import only supports single block device.",
			},
			expectToPass: false,
		}, {
			lsblkOutputFile: "lsblk-efi.json",
			expectAllLogs: []string{
				"INFO: root filesystem found on device: sda",
			},
			expectToPass: true,
		}, {
			lsblkOutputFile: "lsblk-single-disk-lvm.json",
			expectAllLogs: []string{
				"INFO: root filesystem found on device: sda",
			},
			expectToPass: true,
		},
	} {
		t.Run(tc.lsblkOutputFile, func(t *testing.T) {
			report, err := (&DisksCheck{
				lsblkOverride: func() ([]byte, error) {
					return ioutil.ReadFile("testdata/" + tc.lsblkOutputFile)
				},
				getMBROverride: func(devName string) ([]byte, error) {
					bytes := make([]byte, 512)
					copy(bytes, "GRUB")
					bytes[510] = 0x55
					bytes[511] = 0xAA
					return bytes, nil
				},
			}).Run()
			if err != nil {
				t.Fatal(err)
			}

			for _, expectedLog := range tc.expectAllLogs {
				assert.Contains(t, report.logs, expectedLog)
			}

			if tc.expectToPass {
				assert.False(t, report.failed, "Expected check to pass")
			} else {
				assert.True(t, report.failed, "Expected check to fail")
			}
		})
	}
}

//  Copyright 2021 Google Inc. All Rights Reserved.
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

package mount

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/shell"
)

// To rebuild mocks, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mount -source $GOFILE -destination mock_mount_inspector.go

// Inspector inspects a mount directory to return:
//   - The underlying block device.
//   - The block device's type.
//   - If the block device is virtual, the number of block devices
//     that comprise it.
type Inspector interface {
	Inspect(dir string) (InspectionResults, error)
}

// InspectionResults contains information about the mountpoint of a directory.
type InspectionResults struct {
	BlockDevicePath        string
	BlockDeviceIsVirtual   bool
	UnderlyingBlockDevices []string
}

// NewMountInspector returns a new inspector that uses command-line utilities.
func NewMountInspector() Inspector {
	return &defaultMountInspector{shell.NewShellExecutor()}
}

type defaultMountInspector struct {
	shellExecutor shell.Executor
}

// Inspect returns the mount information for dir.
func (mi *defaultMountInspector) Inspect(dir string) (mountInfo InspectionResults, err error) {
	if mountInfo.BlockDevicePath, err = mi.getDeviceForMount(dir); err != nil {
		return InspectionResults{}, fmt.Errorf("unable to find mount information for `%s`: %w", dir, err)
	}
	if mountInfo.BlockDeviceIsVirtual, err = mi.isDeviceVirtual(mountInfo.BlockDevicePath); err != nil {
		return InspectionResults{}, fmt.Errorf("unable to find the type of device `%s`: %w", mountInfo.BlockDevicePath, err)
	}
	if mountInfo.UnderlyingBlockDevices, err = mi.getPhysicalDisks(mountInfo.BlockDevicePath); err != nil {
		return InspectionResults{}, fmt.Errorf("unable to find the physical disks for the block device `%s`: %w",
			mountInfo.BlockDevicePath, err)
	}
	return mountInfo, nil
}

// getDeviceForMount returns the path of the block device that is mounted for dir.
func (mi *defaultMountInspector) getDeviceForMount(dir string) (string, error) {
	stdout, err := mi.shellExecutor.Exec("findmnt", "--noheadings", "--output=SOURCE", dir)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout), nil
}

// isDeviceVirtual returns whether the block device is LVM
func (mi *defaultMountInspector) isDeviceVirtual(device string) (bool, error) {
	stdout, err := mi.shellExecutor.Exec("lsblk", "--noheadings", "--output=TYPE", device)
	return strings.TrimSpace(strings.ToLower(stdout)) == "lvm", err
}

// getPhysicalDisks returns the list of physical disks that are used for the blockDevice.
// For example:
//  1. blockDevice is an MBR-style partition:
//     blockDevice: /dev/sdb1
//     getPhysicalDisks:  [/dev/sdb]
//  2. blockDevice is an LVM logical volume that is spread across three disks:
//     blockDevice:  /dev/mapper/vg-lv
//     getPhysicalDisks:  [/dev/sda, /dev/sdb, /dev/sdc]
func (mi *defaultMountInspector) getPhysicalDisks(blockDevice string) (disksForDevice []string, err error) {
	disks, err := mi.getAllPhysicalDisks()
	if err != nil {
		return nil, err
	}

	for _, disk := range disks {
		blkDevices, err := mi.blockDevicesOnDisk(disk)
		if err != nil {
			return nil, err
		}
		for _, blkDevice := range blkDevices {
			if blkDevice == blockDevice {
				disksForDevice = append(disksForDevice, disk)
				break
			}
		}
	}
	return disksForDevice, nil
}

// getAllPhysicalDisks returns the paths of all physical disks on the system.
func (mi *defaultMountInspector) getAllPhysicalDisks() (allDisks []string, err error) {
	return mi.shellExecutor.ExecLines(
		"lsblk", "--noheadings", "--paths", "--list", "--nodeps", "--output=NAME")
}

// blockDevicesOnDisk returns the paths of all block devices contained on a disk.
func (mi *defaultMountInspector) blockDevicesOnDisk(disk string) ([]string, error) {
	return mi.shellExecutor.ExecLines(
		"lsblk", "--noheadings", "--paths", "--list", "--output=NAME", disk)
}

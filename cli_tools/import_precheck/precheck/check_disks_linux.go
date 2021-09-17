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
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/mount"
)

// DisksCheck performs disk configuration checking:
// - finding the root filesystem partition
// - checking if the device is MBR
// - checking whether the root mount is physically located on a single disk.
//   The check fails, for example, when the root mount is on an LVM
//   logical volume that spans multiple disks.
// - check for GRUB
// - warning for any mount points from partitions from other devices
type DisksCheck struct {
	getMBROverride func(devName string) ([]byte, error)
	inspector      mount.Inspector
}

// NewDisksCheck instantiates a DisksCheck instance.
func NewDisksCheck() Check {
	return &DisksCheck{inspector: mount.NewMountInspector()}
}

// GetName returns the name of the precheck step; this is shown to the user.
func (c *DisksCheck) GetName() string {
	return "Disks Check"
}

// Run executes the precheck step.
func (c *DisksCheck) Run() (r *Report, err error) {
	r = &Report{name: c.GetName()}

	mountInfo, err := c.inspector.Inspect("/")
	if err != nil {
		r.result = Unknown
		r.Warn("Failed to inspect the boot disk. Prior to importing, verify that the boot disk " +
			"contains the root filesystem, and that the root filesystem isn't virtualized over " +
			"multiple disks (using LVM, for example).")
		return r, nil
	}

	r.Info(fmt.Sprintf("root filesystem mounted on %s", mountInfo.BlockDevicePath))

	if len(mountInfo.UnderlyingBlockDevices) > 1 {
		format := "root filesystem spans multiple block devices (%s). Typically this occurs when an LVM logical " +
			"volume spans multiple block devices. Image import only supports single block device."
		r.Fatal(fmt.Sprintf(format, strings.Join(mountInfo.UnderlyingBlockDevices, ", ")))
		return r, nil
	}

	bootDisk := mountInfo.UnderlyingBlockDevices[0]
	r.Info(fmt.Sprintf("boot disk detected as %s", bootDisk))
	// MBR checking.
	var mbrData []byte
	if c.getMBROverride != nil {
		mbrData, err = c.getMBROverride(bootDisk)
	} else {
		mbrData, err = c.getMBR(bootDisk)
	}
	if err != nil {
		return nil, err
	}
	if mbrData[510] != 0x55 || mbrData[511] != 0xAA {
		r.Fatal("boot disk does not have an MBR partition table")
	} else {
		r.Info("boot disk has an MBR partition table")
	}
	if !bytes.Contains(mbrData, []byte("GRUB")) {
		r.Fatal("GRUB not detected in MBR")
	} else {
		r.Info("GRUB found in MBR")
	}

	return r, nil
}

func (c *DisksCheck) getMBR(devPath string) ([]byte, error) {
	f, err := os.Open(devPath)
	if err != nil {
		return nil, err
	}
	data := make([]byte, mbrSize)
	_, err = f.Read(data)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", devPath, err)
	}
	return data, nil
}

/*
Copyright 2017 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// disksCheck performs disk configuration checking:
// - finding the root filesystem partition
// - checking if the device is MBR
// - check for GRUB
// - warning for any mount points from partitions from other devices
type disksCheck struct {
	getMBROverride func(devName string) ([]byte, error)
	lsblkOverride  func() ([]byte, error)
}

func (c *disksCheck) getMBR(devName string) ([]byte, error) {
	devPath := filepath.Join("/dev", devName)
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

func (c *disksCheck) readMounts() (*mountPoints, error) {
	var lsblkOut []byte
	var err error
	if c.lsblkOverride != nil {
		lsblkOut, err = c.lsblkOverride()
	} else {
		cmd := exec.Command("lsblk", "--json", "--output", "name,mountpoint,type")
		lsblkOut, err = cmd.Output()
		if err != nil {
			exitErr := err.(*exec.ExitError)
			err = fmt.Errorf("lsblk: %v, stderr: %s", err, exitErr.Stderr)
		}
	}

	if err != nil {
		return nil, err
	}
	parsed := &LSBLKOutput{}
	err = json.Unmarshal(lsblkOut, parsed)
	if err != nil {
		return nil, err
	}
	mounts := &mountPoints{}
	mounts.addAll(parsed.Blockdevices, []string{})
	return mounts, nil
}

func (c *disksCheck) lsblk() (string, error) {
	cmd := exec.Command("lsblk", "-i")
	out, err := cmd.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		return "", fmt.Errorf("lsblk: %v, stderr: %s", err, exitErr.Stderr)
	}
	return string(out), nil
}

func (c *disksCheck) getName() string {
	return "Disks Check"
}

// mountPoints supports listing the mounts on a system, and determining which block device(s)
// the mount is physically contained. This is helpful since technologies such as LVM allow
// logical mounts to span physical devices.
type mountPoints struct {
	mounts []struct {
		dir       string
		hierarchy []string
	}
}

// listPhysicalDevicesForMount returns the the block device(s) that contain the mount.
// An empty return value means the mount was not found.
func (m *mountPoints) listPhysicalDevicesForMount(mount string) (devices []string) {
	set := map[string]struct{}{}
	for _, mp := range m.mounts {
		if mp.dir == mount {
			set[mp.hierarchy[0]] = struct{}{}
		}
	}
	for k, _ := range set {
		devices = append(devices, k)
	}
	// sorted for stability in testing.
	sort.Strings(devices)
	return devices
}

// listMountPoints returns all mounts.
func (m *mountPoints) listMountPoints() (mounts []string) {
	set := map[string]struct{}{}
	for _, mount := range m.mounts {
		set[mount.dir] = struct{}{}
	}
	for k, _ := range set {
		mounts = append(mounts, k)
	}
	// sorted for stability in testing.
	sort.Strings(mounts)
	return mounts
}

// addAll populates the mountPoints data structure with the response from lsblk.
func (m *mountPoints) addAll(elements []DiskElement, basePath []string) {
	for _, element := range elements {
		path := make([]string, len(basePath))
		copy(path, basePath)
		path = append(path, element.Name)
		if element.Mountpoint != "" {
			m.mounts = append(m.mounts, struct {
				dir       string
				hierarchy []string
			}{element.Mountpoint, path})
		}
		if len(element.Children) > 0 {
			m.addAll(element.Children, path)
		}
	}
}

func (c *disksCheck) run() (r *report, err error) {
	r = &report{name: c.getName()}

	allMounts, err := c.readMounts()
	if err != nil {
		r.Warn(fmt.Sprintf("Failed to execute lsblk: %s", err))
		return r, nil
	}

	rootDevices := allMounts.listPhysicalDevicesForMount("/")
	switch len(rootDevices) {
	case 0:
		r.Fatal("root filesystem partition not found on any block devices.")
		return r, nil
	case 1:
		r.Info(fmt.Sprintf("root filesystem found on device: %s", rootDevices[0]))
	default:
		format := "root filesystem spans multiple physical devices (%s). Translation only supports single block device."
		r.Fatal(fmt.Sprintf(format, rootDevices))
		return r, nil
	}

	rootDevice := rootDevices[0]
	for _, mount := range allMounts.listMountPoints() {
		if mount == "/" {
			continue
		}
		devices := allMounts.listPhysicalDevicesForMount(mount)
		switch len(devices) {
		case 0:
			// This implies a bug in mountPoints.addAll.
			panic(fmt.Sprintf("Invalid parse of mount %s", mount))
		case 1:
			if devices[0] != rootDevice {
				format := "mount %s is not on the root device %s and will be OMITTED from translation."
				r.Warn(fmt.Sprintf(format, mount, rootDevice))
			}
		default:
			format := "mount %s is on multiple physical devices (%s) and will be OMITTED from translation."
			r.Warn(fmt.Sprintf(format, mount, devices))
		}
	}

	// MBR checking.
	var mbrData []byte
	if c.getMBROverride != nil {
		mbrData, err = c.getMBROverride(rootDevice)
	} else {
		mbrData, err = c.getMBR(rootDevice)
	}
	if err != nil {
		return nil, err
	}
	if mbrData[510] != 0x55 || mbrData[511] != 0xAA {
		r.Fatal("root filesystem device is NOT MBR")
	} else {
		r.Info("root filesystem device is MBR.")
	}
	if !bytes.Contains(mbrData, []byte("GRUB")) {
		r.Fatal("GRUB not detected on MBR")
	} else {
		r.Info("GRUB found in root filesystem device MBR")
	}

	return r, nil
}

// LSBLKOutput is a struct representing the output from running
//  `lsblk --json`.
type LSBLKOutput struct {
	Blockdevices []DiskElement `json:"blockdevices"`
}

// DiskElement is a struct representing the nested fields within
// the output of `lsblk --json`. See the testdata directory for
// examples of nesting.
type DiskElement struct {
	Name       string        `json:"name"`
	Mountpoint string        `json:"mountpoint"`
	Children   []DiskElement `json:"children"`
}

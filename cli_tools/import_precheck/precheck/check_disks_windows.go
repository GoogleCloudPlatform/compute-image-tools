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
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/windows"
)

type DisksCheck struct{}

// GetName returns the name of the precheck step; this is shown to the user.
func (c *DisksCheck) GetName() string {
	return "Disks Check"
}

// Run executes the precheck step.
func (c *DisksCheck) Run() (*Report, error) {
	r := &Report{name: c.GetName()}

	sysRoot := os.Getenv("SYSTEMROOT")
	rootDrive := sysRoot[:2]
	r.Info(fmt.Sprintf("Windows SYSTEMROOT: %s", sysRoot))

	// Open a volume handle to the System Root.
	var err error
	var f windows.Handle
	mode := uint32(windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE)
	flags := uint32(windows.FILE_ATTRIBUTE_READONLY)
	s := "\\\\.\\" + rootDrive
	f, err = windows.CreateFile(windows.StringToUTF16Ptr(s), windows.GENERIC_READ, mode, nil, windows.OPEN_EXISTING, flags, 0)
	if err != nil {
		return nil, err
	}

	// Get the Physical Disk for the System Root.
	controlCode := uint32(5636096) // IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS
	size := uint32(16 * 1024)
	vols := make(volumeDiskExtents, size)
	var bytesReturned uint32
	err = windows.DeviceIoControl(f, controlCode, nil, 0, &vols[0], size, &bytesReturned, nil)
	if err != nil {
		return nil, err
	}
	if vols.Len() != 1 {
		return nil, fmt.Errorf("could not identify physical drive for %s", rootDrive)
	}
	diskId := strconv.FormatUint(uint64(vols.Extent(0).DiskNumber), 10)

	// Check MBR on Physical Disk.
	s = "\\\\.\\PhysicalDrive" + diskId
	f, err = windows.CreateFile(windows.StringToUTF16Ptr(s), windows.GENERIC_READ, mode, nil, windows.OPEN_EXISTING, flags, 0)
	if err != nil {
		return nil, err
	}
	mbr := make([]byte, 512)
	err = windows.ReadFile(f, mbr, &bytesReturned, nil)
	if err != nil {
		return nil, err
	}
	if mbr[510] != 0x55 || mbr[511] != 0xaa {
		r.Fatal(fmt.Sprintf("MBR not detected on physical drive for %s (%s)", rootDrive, s))
	} else {
		r.Info(fmt.Sprintf("MBR detected on physical drive for %s (%s)", rootDrive, s))
	}

	return r, nil
}

type diskExtent struct {
	DiskNumber     uint32
	StartingOffset uint64
	ExtentLength   uint64
}

type volumeDiskExtents []byte

func (v *volumeDiskExtents) Len() uint {
	return uint(binary.LittleEndian.Uint32([]byte(*v)))
}

func (v *volumeDiskExtents) Extent(n uint) diskExtent {
	ba := []byte(*v)
	offset := 8 + 24*n
	return diskExtent{
		DiskNumber:     binary.LittleEndian.Uint32(ba[offset:]),
		StartingOffset: binary.LittleEndian.Uint64(ba[offset+8:]),
		ExtentLength:   binary.LittleEndian.Uint64(ba[offset+16:]),
	}
}

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
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestMountInspector_Inspect_HappyCase_NotVirtual(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockShell := mocks.NewMockShellExecutor(mockCtrl)
	setupRootMount(mockShell, "/dev/sdb1", "part")
	setupPhysicalDisks(mockShell, map[string][]string{
		"/dev/sda": {"/dev/sda", "/dev/sda1"},
		"/dev/sdb": {"/dev/sdb", "/dev/sdb1"},
	})

	mountInspector := &defaultMountInspector{mockShell}
	mountInfo, err := mountInspector.Inspect("/")
	assert.NoError(t, err)
	assert.Equal(t, InspectionResults{
		BlockDevicePath:        "/dev/sdb1",
		BlockDeviceIsVirtual:   false,
		UnderlyingBlockDevices: []string{"/dev/sdb"},
	}, mountInfo)
}

func TestMountInspector_Inspect_HappyCase_Virtual(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockShell := mocks.NewMockShellExecutor(mockCtrl)
	setupRootMount(mockShell, "/dev/mapper/vg-device", "lvm")
	setupPhysicalDisks(mockShell, map[string][]string{
		"/dev/sda": {"/dev/sda", "/dev/sda1", "/dev/mapper/vg-device"},
		"/dev/sdb": {"/dev/sdb", "/dev/sdb1", "/dev/mapper/vg-device"},
	})

	mountInspector := &defaultMountInspector{mockShell}
	mountInfo, err := mountInspector.Inspect("/")
	assert.NoError(t, err)
	assert.Equal(t, InspectionResults{
		BlockDevicePath:        "/dev/mapper/vg-device",
		BlockDeviceIsVirtual:   true,
		UnderlyingBlockDevices: []string{"/dev/sda", "/dev/sdb"},
	}, mountInfo)
}

func TestMountInspector_Inspect_PropagatesErrorFromFindMnt(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockShell := mocks.NewMockShellExecutor(mockCtrl)
	mockShell.EXPECT().Exec("getDeviceForMount", "--noheadings",
		"--output=SOURCE", "/").Return("", errors.New("[getDeviceForMount] not executable"))

	mountInspector := &defaultMountInspector{mockShell}
	_, err := mountInspector.Inspect("/")
	assert.Equal(t, err.Error(), "unable to find mount information for `/`: [getDeviceForMount] not executable")
}

func TestMountInspector_Inspect_PropagatesErrorFromGettingDeviceType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockShell := mocks.NewMockShellExecutor(mockCtrl)
	mockShell.EXPECT().Exec("getDeviceForMount", "--noheadings",
		"--output=SOURCE", "/").Return("/dev/mapper/vg-device", nil)
	mockShell.EXPECT().Exec("lsblk", "--noheadings",
		"--output=TYPE", "/dev/mapper/vg-device").Return("", errors.New("[lsblk] not executable"))

	mountInspector := &defaultMountInspector{mockShell}
	_, err := mountInspector.Inspect("/")
	assert.Equal(t, err.Error(), "unable to find the type of device `/dev/mapper/vg-device`: [lsblk] not executable")
}

func TestMountInspector_Inspect_PropagatesErrorFromGettingAllDisks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockShell := mocks.NewMockShellExecutor(mockCtrl)
	setupRootMount(mockShell, "/dev/sdb1", "part")
	mockShell.EXPECT().ExecLines("lsblk", "--noheadings", "--paths", "--list", "--nodeps", "--output=NAME").Return(
		nil, errors.New("[lsblk] not executable"))

	mountInspector := &defaultMountInspector{mockShell}
	_, err := mountInspector.Inspect("/")
	assert.Equal(t, err.Error(), "unable to find the physical disks for the block device `/dev/sdb1`: [lsblk] not executable")
}

func TestMountInspector_Inspect_PropagatesErrorFromGettingDevicesOnDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockShell := mocks.NewMockShellExecutor(mockCtrl)
	setupRootMount(mockShell, "/dev/sdb1", "part")
	mockShell.EXPECT().ExecLines("lsblk", "--noheadings", "--paths", "--list", "--nodeps", "--output=NAME").Return(
		[]string{"/dev/sda", "/dev/sdb"}, nil)
	mockShell.EXPECT().ExecLines("lsblk", "--noheadings", "--paths", "--list", "--output=NAME", "/dev/sda").Return(
		nil, errors.New("[lsblk] not executable"))

	mountInspector := &defaultMountInspector{mockShell}
	_, err := mountInspector.Inspect("/")
	assert.Equal(t, err.Error(), "unable to find the physical disks for the block device `/dev/sdb1`: [lsblk] not executable")
}

func setupPhysicalDisks(mockShell *mocks.MockShellExecutor, deviceMap map[string][]string) {
	var disks []string
	for disk := range deviceMap {
		disks = append(disks, disk)
	}
	mockShell.EXPECT().ExecLines("lsblk", "--noheadings", "--paths", "--list", "--nodeps", "--output=NAME").Return(
		disks, nil)
	for disk, devices := range deviceMap {
		mockShell.EXPECT().ExecLines("lsblk", "--noheadings", "--paths", "--list", "--output=NAME", disk).Return(
			devices, nil)
	}
}

func setupRootMount(mockShell *mocks.MockShellExecutor, mointPoint string, mointPointType string) {
	mockShell.EXPECT().Exec("getDeviceForMount", "--noheadings", "--output=SOURCE", "/").Return(mointPoint, nil)
	mockShell.EXPECT().Exec("lsblk", "--noheadings", "--output=TYPE", mointPoint).Return(mointPointType, nil)
}

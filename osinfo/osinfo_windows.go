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

package osinfo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows"
)

var (
	version                     = windows.NewLazySystemDLL("version.dll")
	procGetFileVersionInfoSizeW = version.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = version.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = version.NewProc("VerQueryValueW")
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms647464(v=vs.85).aspx
func getTranslation(block []byte) (string, error) {
	var start uint
	var length uint
	blockStart := uintptr(unsafe.Pointer(&block[0]))
	if ret, _, _ := procVerQueryValueW.Call(
		blockStart,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(`\VarFileInfo\Translation`))),
		uintptr(unsafe.Pointer(&start)),
		uintptr(unsafe.Pointer(&length))); ret == 0 {
		return "", errors.New("zero return code from VerQueryValueW indicates failure")
	}

	begin := int(start) - int(blockStart)
	// For translation data length is bytes.
	trans := block[begin : begin+int(length)]

	// Each 'translation' is 4 bytes long (2 16-bit sections), we just want the
	// first one for simplicity.
	t := make([]byte, 4)
	// 16-bit language ID little endian
	// https://msdn.microsoft.com/en-us/library/windows/desktop/dd318693(v=vs.85).aspx
	t[0], t[1] = trans[1], trans[0]
	// 16-bit code page ID little endian
	// https://msdn.microsoft.com/en-us/library/windows/desktop/dd317756(v=vs.85).aspx
	t[2], t[3] = trans[3], trans[2]

	return fmt.Sprintf("%x", t), nil
}

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms647464(v=vs.85).aspx
func getFileVersion(block []byte, langCodePage string) (string, error) {
	var start uint
	var length uint
	blockStart := uintptr(unsafe.Pointer(&block[0]))
	if ret, _, _ := procVerQueryValueW.Call(
		blockStart,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(fmt.Sprintf(`\StringFileInfo\%s\FileVersion`, langCodePage)))),
		uintptr(unsafe.Pointer(&start)),
		uintptr(unsafe.Pointer(&length))); ret == 0 {
		return "", errors.New("zero return code from VerQueryValueW indicates failure")
	}
	begin := int(start) - int(blockStart)
	// For version information length is characters (UTF16).
	result := block[begin : begin+int(2*length)]

	// Result is UTF16LE.
	u16s := make([]uint16, length)
	for i := range u16s {
		u16s[i] = uint16(result[i*2+1])<<8 | uint16(result[i*2])
	}

	return syscall.UTF16ToString(u16s), nil
}

func getKernelVersion() (string, error) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	path := filepath.Join(root, "System32", "ntoskrnl.exe")
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	pPtr := unsafe.Pointer(syscall.StringToUTF16Ptr(path))

	size, _, _ := procGetFileVersionInfoSizeW.Call(
		uintptr(pPtr))
	if size <= 0 {
		return "", errors.New("GetFileVersionInfoSize call failed, data size can not be 0")
	}

	info := make([]byte, size)
	if ret, _, _ := procGetFileVersionInfoW.Call(
		uintptr(pPtr),
		0,
		uintptr(len(info)),
		uintptr(unsafe.Pointer(&info[0]))); ret == 0 {
		return "", errors.New("zero return code from GetFileVersionInfoW indicates failure")
	}

	// This should be something like 040904b0 for US English UTF16LE.
	langCodePage, err := getTranslation(info)
	if err != nil {
		return "", err
	}

	return getFileVersion(info, langCodePage)
}

// GetDistributionInfo reports DistributionInfo.
func GetDistributionInfo() (*DistributionInfo, error) {
	oi, err := osInfo()
	if err != nil {
		return nil, err
	}

	di := &DistributionInfo{ShortName: "windows", LongName: oi.Caption, Version: oi.Version, Kernel: oi.Version, Architecture: Architecture(runtime.GOARCH)}

	kVersion, err := getKernelVersion()
	if err != nil {
		return di, err
	}
	di.Kernel = kVersion
	return di, nil
}

type win32_OperatingSystem struct {
	Caption, Version string
}

func osInfo() (*win32_OperatingSystem, error) {
	var ops []win32_OperatingSystem
	if err := wmi.Query(wmi.CreateQuery(&ops, ""), &ops); err != nil {
		return nil, err
	}
	return &ops[0], nil
}

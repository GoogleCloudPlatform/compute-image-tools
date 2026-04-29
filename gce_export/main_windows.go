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

package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const IOCTL_DISK_GET_LENGTH_INFO = 0x0007405c

func diskLength(file *os.File) (int64, error) {
	dglibuf := make([]byte, 1024)
	var bytesReturned uint32
	if err := windows.DeviceIoControl(windows.Handle(file.Fd()), IOCTL_DISK_GET_LENGTH_INFO, nil, 0, &dglibuf[0], uint32(len(dglibuf)), &bytesReturned, nil); err != nil {
		return 0, err
	}

	// https://msdn.microsoft.com/en-us/library/aa365001
	type getLengthInfo struct {
		Length int64
	}

	gli := (*getLengthInfo)(unsafe.Pointer(&dglibuf[0]))
	return gli.Length, nil
}

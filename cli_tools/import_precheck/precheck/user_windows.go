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

	"golang.org/x/sys/windows"
)

// CheckRoot returns an error if the process's runtime user isn't root.
func CheckRoot() error {
	var adminGroup *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&adminGroup)
	if err != nil {
		return err
	}

	var t windows.Token // A nil Token will use the current thread's primary token.
	var b bool
	b, err = t.IsMember(adminGroup)
	if err != nil {
		return err
	} else if !b {
		return errors.New("must be run as Administrator")
	}
	return nil
}

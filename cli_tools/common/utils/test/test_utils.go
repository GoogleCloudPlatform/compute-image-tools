//  Copyright 2019 Google Inc. All Rights Reserved.
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

//+build !test

package testutils

import (
	"fmt"
	"os"
)

// BackupOsArgs backs up os.Args and returns a function that restores os.Args that can be used
// with defer.
func BackupOsArgs() func() {
	oldArgs := os.Args
	return func() { os.Args = oldArgs }
}

// BuildOsArgs builds os.Args from a map of key-value pairs containing arguments.
func BuildOsArgs(cliArgs map[string]interface{}) {
	os.Args = make([]string, len(cliArgs)+1)
	i := 0
	os.Args[i] = "cmd"
	i++
	for key, value := range cliArgs {
		if _, ok := value.(bool); ok || value != nil {
			os.Args[i] = formatCliArg(key, value)
			i++
		}
	}
}

func formatCliArg(argKey, argValue interface{}) string {
	if argValue == true {
		return fmt.Sprintf("-%v", argKey)
	}
	if argValue != false {
		return fmt.Sprintf("-%v=%v", argKey, argValue)
	}
	return ""
}

// SetStringP sets a value for a string pointer and returns a function that restores the pointer
// to initial state that can be used with defer
func SetStringP(p **string, value string) func() {
	oldValue := *p
	*p = &value
	return func() {
		*p = oldValue
	}
}

// SetIntP sets a value for an int pointer and returns a function that restores the pointer
// to initial state that can be used with defer
func SetIntP(p **int, value int) func() {
	oldValue := *p
	*p = &value
	return func() {
		*p = oldValue
	}
}

// SetBoolP sets a value for a boolean pointer and returns a function that restores the pointer
// to initial state that can be used with defer
func SetBoolP(p **bool, value bool) func() {
	oldValue := *p
	*p = &value
	return func() { *p = oldValue }
}

// ClearStringFlag clears a flag from a map with CLI args as well as from a pointer holding
// the corresponding value internally. Returns a function which restores the pointer to the previous
// value. This function can be used with defer.
func ClearStringFlag(cliArgs map[string]interface{}, flagKey string, flag **string) func() {
	delete(cliArgs, flagKey)
	return SetStringP(flag, "")
}

// ClearBoolFlag clears a flag from a map with CLI args as well as from a pointer holding
// the corresponding value internally. Returns a function which restores the pointer to the previous
// value. This function can be used with defer.
func ClearBoolFlag(cliArgs map[string]interface{}, flagKey string, flag **bool) func() {
	delete(cliArgs, flagKey)
	return SetBoolP(flag, false)
}

// ClearIntFlag clears a flag from a map with CLI args as well as from a pointer holding
// the corresponding value internally. Returns a function which restores the pointer to the previous
// value. This function can be used with defer.
func ClearIntFlag(cliArgs map[string]interface{}, flagKey string, flag **int) func() {
	delete(cliArgs, flagKey)
	return SetIntP(flag, 0)
}

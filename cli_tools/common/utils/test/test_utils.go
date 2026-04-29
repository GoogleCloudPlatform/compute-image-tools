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

//go:build !test
// +build !test

package test

import (
	"bytes"
	"compress/gzip"
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

// CreateCompressedFile creates a valid compressed file in memory
func CreateCompressedFile() string {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Name = "dummy.txt"
	zw.Write([]byte("some content"))
	zw.Close()
	return buf.String()
}

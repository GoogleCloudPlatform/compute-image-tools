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

// These test can only be run on windows, as the functions are highly dependent on windows OS.
package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	pathNotExistErr    = "The system cannot find the path specified."
	fileNotExistErr    = "The system cannot find the file specified."
	systemLogPath      = `C:\Windows\System32\winevt\Logs\System.evtx`
	kubeletLogFileName = "kubelet.log"
)

func pathNonExist(e error) bool {
	if strings.Contains(e.Error(), pathNotExistErr) {
		return true
	}
	return false
}

func fileNonExist(e error) bool {
	if strings.Contains(e.Error(), fileNotExistErr) {
		return true
	}
	return false
}

func TestCollectFilePaths(t *testing.T) {
	// Test setup: create dummy test folder and file for test, clean it up afterwards
	dir := os.TempDir()
	testRoot := filepath.Join(dir, "collectFilePathsTest")
	defer os.RemoveAll(testRoot)
	kubeletlogFilePath := filepath.Join(testRoot, kubeletLogFileName)
	os.Create(kubeletlogFilePath)
	nonExistFilePath := filepath.Join(dir, "xxx")

	type args struct {
		roots []string
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		errOK func(error) bool
	}{
		{"Nil roots", args{nil}, []string{}, nil},
		{"Empty roots", args{[]string{""}}, []string{}, pathNonExist},
		{"Existing roots", args{[]string{testRoot}}, []string{kubeletlogFilePath}, nil},
		{"Non-existing paths", args{[]string{nonExistFilePath}}, []string{}, fileNonExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFiles, gotErrs := collectFilePaths(tt.args.roots)
			if !reflect.DeepEqual(gotFiles, tt.want) {
				t.Errorf("unexpected filepaths, want %v, got %v", tt.want, gotFiles)
			}
			for _, err := range gotErrs {
				if tt.errOK == nil || !tt.errOK(err) {
					t.Errorf("collectFilePaths() got unexpected error = %v", gotErrs)
				}
			}
		})
	}
}

func stringArrayIncludesString(stringArray []string, target string) bool {
	for _, s := range stringArray {
		if s == target {
			return true
		}
	}
	return false
}

func TestGatherEventLogsGathersExpectedSystemLogFile(t *testing.T) {
	type args struct {
		logs chan logFolder
		errs chan error
	}
	test := struct {
		name string
		args args
	}{
		"GatherEventLogs",
		args{make(chan logFolder, 2), make(chan error)},
	}

	t.Run(test.name, func(t *testing.T) {
		go gatherEventLogs(test.args.logs, test.args.errs)
		select {
		case l := <-test.args.logs:
			if !stringArrayIncludesString(l.files, systemLogPath) {
				t.Errorf("Expect %s, but it's missing", systemLogPath)
			}
		case e := <-test.args.errs:
			t.Errorf(e.Error())
		}
	})
}

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
	"strings"
	"testing"
)

const (
	pathNotExistErr = "The system cannot find the path specified."
	fileNotExistErr = "The system cannot find the file specified."
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
	type args struct {
		roots []string
	}
	tests := []struct {
		name  string
		args  args
		errOK func(error) bool
	}{
		{"Nil roots", args{nil}, nil},
		{"Empty roots", args{[]string{""}}, pathNonExist},
		{"Existing roots", args{[]string{k8sLogsRoot, eventLogsRoot}}, nil},
		{"Non-existing paths", args{[]string{`C:\etc\kubernetes\logs\xxxx`}}, fileNonExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErrs := collectFilePaths(tt.args.roots)
			for _, err := range gotErrs {
				if tt.errOK == nil || !tt.errOK(err) {
					t.Errorf("collectFilePaths() got unexpected error = %v", gotErrs)
				}
			}
		})
	}
}

func TestGatherEventLogs(t *testing.T) {
	type args struct {
		logs chan logFolder
		errs chan error
	}
	tests := []struct {
		name string
		args args
	}{
		{"GatherEventLogs", args{make(chan logFolder, 2), make(chan error)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go gatherEventLogs(tt.args.logs, tt.args.errs)
			select {
			case <-tt.args.logs:
			case e := <-tt.args.errs:
				t.Errorf(e.Error())
			}
		})
	}
}

func TestGatherKubernetesLogs(t *testing.T) {
	type args struct {
		logs chan logFolder
		errs chan error
	}
	tests := []struct {
		name string
		args args
	}{
		{"GatherK8sLogs", args{make(chan logFolder, 2), make(chan error)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go gatherKubernetesLogs(tt.args.logs, tt.args.errs)
			select {
			case <-tt.args.logs:
			case e := <-tt.args.errs:
				if !strings.Contains(e.Error(), crashDump) {
					t.Errorf(e.Error())
				}
			}
		})
	}
}

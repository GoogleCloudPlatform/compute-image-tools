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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	pathNotExistErr            = "The system cannot find the path specified."
	fileNotExistErr            = "The system cannot find the file specified."
	systemLogPath              = `C:\Windows\System32\winevt\Logs\System.evtx`
	kubeletLogFileName         = "kubelet.log"
	applicationTextLogFileName = "Application.log"
)

func TestGetDockerImagesList(t *testing.T) {
	logFolderCh := make(chan logFolder, 2)
	errCh := make(chan error)
	// Test setup: create temp folder for test, clean it up afterwards
	var err error
	tmpFolder, err = os.MkdirTemp("", "getDockerImagesListTest")
	if err != nil {
		t.Errorf("Error creating a temporary test folder:\n%v", err.Error())
	}
	defer os.RemoveAll(tmpFolder)

	t.Run("Gathers docker images list", func(t *testing.T) {
		go getDockerImagesList(logFolderCh, errCh)
		select {
		case l := <-logFolderCh:
			if !stringArrayIncludesSubstring(l.files, dockerImageListFileName) {
				t.Errorf("Expect %s, but it's missing", dockerImageListFileName)
			}
		case e := <-errCh:
			if _, err := exec.LookPath("docker"); err == nil {
				t.Errorf(e.Error())
			}
		}
	})
}

func TestGatherRDPSettings(t *testing.T) {
	logFolderCh := make(chan logFolder, 2)
	errCh := make(chan error)

	// Test setup: use the temp test build package folder for test
	var err error
	tmpFolder, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Errorf("Error getting the temp test build folder:\n%v", err.Error())
	}

	// Copy the rdp_status.ps1 over to temp test build folder for execution
	rdpScriptFilePath := filepath.Join(tmpFolder, rdpScriptFileName)
	input, err := ioutil.ReadFile(rdpScriptFileName)
	if err != nil {
		t.Errorf("Error loading the rdp_status.ps1 file:\n%v", err.Error())
	}
	ioutil.WriteFile(rdpScriptFilePath, input, 0644)

	t.Run("Gathers Expected RDP Status File", func(t *testing.T) {
		go gatherRDPSettings(logFolderCh, errCh)
		select {
		case l := <-logFolderCh:
			if !stringArrayIncludesSubstring(l.files, rdpStatusFileName) {
				t.Errorf("Expect %s, but it's missing", rdpStatusFileName)
			}
		case e := <-errCh:
			t.Errorf(e.Error())
		}
	})
}

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

func TestGetPlainEventLogs(t *testing.T) {
	// Test setup: create temp test folder for test, clean it up afterwards
	var err error
	tmpFolder, err = os.MkdirTemp("", "getPlainEventLogsTest")
	if err != nil {
		t.Errorf("Error creating a temporary test folder:\n%v", err.Error())
	}
	defer os.RemoveAll(tmpFolder)

	tests := []struct {
		name         string
		args         []winEvt
		want         []string
		expectErrStr string
	}{
		{name: "Nil events",
			args:         nil,
			want:         []string{},
			expectErrStr: "",
		},
		{name: "Empty events",
			args:         []winEvt{},
			want:         []string{},
			expectErrStr: "",
		},
		{name: "Existing events logName",
			args:         []winEvt{{"Application", false}},
			want:         []string{filepath.Join(tmpFolder, applicationTextLogFileName)},
			expectErrStr: "",
		},
		{name: "Non-Existing events logName",
			args:         []winEvt{{"xxx", false}},
			want:         []string{},
			expectErrStr: "xxx",
		},
		{name: "Existing events providerName",
			args:         []winEvt{{"GCEWindowsAgent", true}},
			want:         []string{},
			expectErrStr: "",
		},
		{name: "Non-Existing events providerName",
			args:         []winEvt{{"System", true}},
			want:         []string{},
			expectErrStr: "System",
		},
	}
	errCh := make(chan error)
	gotFilesCh := make(chan []string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				gotFilesCh <- getPlainEventLogs(tt.args, errCh)
			}()
			select {
			case e := <-errCh:
				if tt.expectErrStr == "" || !strings.Contains(e.Error(), tt.expectErrStr) {
					t.Errorf("unexpected err, want %v, got %v", tt.expectErrStr, e.Error())
				}
			case gotFiles := <-gotFilesCh:
				if !reflect.DeepEqual(gotFiles, tt.want) {
					t.Errorf("unexpected filepaths, want %v, got %v", tt.want, gotFiles)
				}
			}
		})
	}
}

func TestCollectFilePaths(t *testing.T) {
	// Test setup: create dummy test folder and file for test, clean it up afterwards
	dir := os.TempDir()
	testRoot, err := os.MkdirTemp("", "collectFilePathsTest")
	if err != nil {
		t.Errorf("Error creating a temporary test folder:\n%v", err.Error())
	}
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

func stringArrayIncludesSubstring(stringArray []string, target string) bool {
	for _, s := range stringArray {
		if strings.Contains(s, target) {
			return true
		}
	}
	return false
}

func TestGatherEventLogs(t *testing.T) {
	logFolderCh := make(chan logFolder, 2)
	errCh := make(chan error)

	t.Run("Gathers Expected SystemLog File", func(t *testing.T) {
		go gatherEventLogs(logFolderCh, errCh)
		select {
		case l := <-logFolderCh:
			if !stringArrayIncludesSubstring(l.files, systemLogPath) {
				t.Errorf("Expect %s, but it's missing", systemLogPath)
			}
		case e := <-errCh:
			t.Errorf(e.Error())
		}
	})
}

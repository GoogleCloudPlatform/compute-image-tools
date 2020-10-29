//  Copyright 2020 Google Inc. All Rights Reserved.
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

// Package mocks is a generated GoMock package.
package mocks

import (
	domain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	gomock "github.com/golang/mock/gomock"
	v1 "google.golang.org/api/compute/v1"
	reflect "reflect"
)

// MockInstanceExportCleaner is a mock of InstanceExportCleaner interface
type MockInstanceExportCleaner struct {
	ctrl     *gomock.Controller
	recorder *MockInstanceExportCleanerMockRecorder
}

// MockInstanceExportCleanerMockRecorder is the mock recorder for MockInstanceExportCleaner
type MockInstanceExportCleanerMockRecorder struct {
	mock *MockInstanceExportCleaner
}

// NewMockInstanceExportCleaner creates a new mock instance
func NewMockInstanceExportCleaner(ctrl *gomock.Controller) *MockInstanceExportCleaner {
	mock := &MockInstanceExportCleaner{ctrl: ctrl}
	mock.recorder = &MockInstanceExportCleanerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockInstanceExportCleaner) EXPECT() *MockInstanceExportCleanerMockRecorder {
	return m.recorder
}

// Cancel mocks base method
func (m *MockInstanceExportCleaner) Cancel(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Cancel indicates an expected call of Cancel
func (mr *MockInstanceExportCleanerMockRecorder) Cancel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockInstanceExportCleaner)(nil).Cancel), arg0)
}

// Clean mocks base method
func (m *MockInstanceExportCleaner) Clean(arg0 *v1.Instance, arg1 *domain.OVFExportParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Clean", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Clean indicates an expected call of Clean
func (mr *MockInstanceExportCleanerMockRecorder) Clean(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Clean", reflect.TypeOf((*MockInstanceExportCleaner)(nil).Clean), arg0, arg1)
}

// TraceLogs mocks base method
func (m *MockInstanceExportCleaner) TraceLogs() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TraceLogs")
	ret0, _ := ret[0].([]string)
	return ret0
}

// TraceLogs indicates an expected call of TraceLogs
func (mr *MockInstanceExportCleanerMockRecorder) TraceLogs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TraceLogs", reflect.TypeOf((*MockInstanceExportCleaner)(nil).TraceLogs))
}

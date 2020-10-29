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
	disk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockInspector is a mock of Inspector interface
type MockInspector struct {
	ctrl     *gomock.Controller
	recorder *MockInspectorMockRecorder
}

// MockInspectorMockRecorder is the mock recorder for MockInspector
type MockInspectorMockRecorder struct {
	mock *MockInspector
}

// NewMockInspector creates a new mock instance
func NewMockInspector(ctrl *gomock.Controller) *MockInspector {
	mock := &MockInspector{ctrl: ctrl}
	mock.recorder = &MockInspectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockInspector) EXPECT() *MockInspectorMockRecorder {
	return m.recorder
}

// Cancel mocks base method
func (m *MockInspector) Cancel(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Cancel indicates an expected call of Cancel
func (mr *MockInspectorMockRecorder) Cancel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockInspector)(nil).Cancel), arg0)
}

// Inspect mocks base method
func (m *MockInspector) Inspect(arg0 string, arg1 bool) (disk.InspectionResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Inspect", arg0, arg1)
	ret0, _ := ret[0].(disk.InspectionResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Inspect indicates an expected call of Inspect
func (mr *MockInspectorMockRecorder) Inspect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Inspect", reflect.TypeOf((*MockInspector)(nil).Inspect), arg0, arg1)
}

// TraceLogs mocks base method
func (m *MockInspector) TraceLogs() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TraceLogs")
	ret0, _ := ret[0].([]string)
	return ret0
}

// TraceLogs indicates an expected call of TraceLogs
func (mr *MockInspectorMockRecorder) TraceLogs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TraceLogs", reflect.TypeOf((*MockInspector)(nil).TraceLogs))
}

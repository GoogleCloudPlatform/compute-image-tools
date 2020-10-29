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
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockOvfManifestGenerator is a mock of OvfManifestGenerator interface
type MockOvfManifestGenerator struct {
	ctrl     *gomock.Controller
	recorder *MockOvfManifestGeneratorMockRecorder
}

// MockOvfManifestGeneratorMockRecorder is the mock recorder for MockOvfManifestGenerator
type MockOvfManifestGeneratorMockRecorder struct {
	mock *MockOvfManifestGenerator
}

// NewMockOvfManifestGenerator creates a new mock instance
func NewMockOvfManifestGenerator(ctrl *gomock.Controller) *MockOvfManifestGenerator {
	mock := &MockOvfManifestGenerator{ctrl: ctrl}
	mock.recorder = &MockOvfManifestGeneratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOvfManifestGenerator) EXPECT() *MockOvfManifestGeneratorMockRecorder {
	return m.recorder
}

// Cancel mocks base method
func (m *MockOvfManifestGenerator) Cancel(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Cancel indicates an expected call of Cancel
func (mr *MockOvfManifestGeneratorMockRecorder) Cancel(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockOvfManifestGenerator)(nil).Cancel), arg0)
}

// GenerateAndWriteToGCS mocks base method
func (m *MockOvfManifestGenerator) GenerateAndWriteToGCS(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateAndWriteToGCS", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GenerateAndWriteToGCS indicates an expected call of GenerateAndWriteToGCS
func (mr *MockOvfManifestGeneratorMockRecorder) GenerateAndWriteToGCS(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateAndWriteToGCS", reflect.TypeOf((*MockOvfManifestGenerator)(nil).GenerateAndWriteToGCS), arg0, arg1)
}

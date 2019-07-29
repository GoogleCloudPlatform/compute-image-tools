// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute (interfaces: Client)

// Package mocks is a generated GoMock package.
package mocks

import (
	compute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	gomock "github.com/golang/mock/gomock"
	v0_beta "google.golang.org/api/compute/v0.beta"
	v1 "google.golang.org/api/compute/v1"
	googleapi "google.golang.org/api/googleapi"
	reflect "reflect"
)

// MockClient is a mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// AttachDisk mocks base method
func (m *MockClient) AttachDisk(arg0, arg1, arg2 string, arg3 *v1.AttachedDisk) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AttachDisk", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// AttachDisk indicates an expected call of AttachDisk
func (mr *MockClientMockRecorder) AttachDisk(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AttachDisk", reflect.TypeOf((*MockClient)(nil).AttachDisk), arg0, arg1, arg2, arg3)
}

// BasePath mocks base method
func (m *MockClient) BasePath() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BasePath")
	ret0, _ := ret[0].(string)
	return ret0
}

// BasePath indicates an expected call of BasePath
func (mr *MockClientMockRecorder) BasePath() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BasePath", reflect.TypeOf((*MockClient)(nil).BasePath))
}

// CreateDisk mocks base method
func (m *MockClient) CreateDisk(arg0, arg1 string, arg2 *v1.Disk) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDisk", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateDisk indicates an expected call of CreateDisk
func (mr *MockClientMockRecorder) CreateDisk(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDisk", reflect.TypeOf((*MockClient)(nil).CreateDisk), arg0, arg1, arg2)
}

// CreateFirewallRule mocks base method
func (m *MockClient) CreateFirewallRule(arg0 string, arg1 *v1.Firewall) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFirewallRule", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateFirewallRule indicates an expected call of CreateFirewallRule
func (mr *MockClientMockRecorder) CreateFirewallRule(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFirewallRule", reflect.TypeOf((*MockClient)(nil).CreateFirewallRule), arg0, arg1)
}

// CreateForwardingRule mocks base method
func (m *MockClient) CreateForwardingRule(arg0, arg1 string, arg2 *v1.ForwardingRule) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateForwardingRule", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateForwardingRule indicates an expected call of CreateForwardingRule
func (mr *MockClientMockRecorder) CreateForwardingRule(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateForwardingRule", reflect.TypeOf((*MockClient)(nil).CreateForwardingRule), arg0, arg1, arg2)
}

// CreateImage mocks base method
func (m *MockClient) CreateImage(arg0 string, arg1 *v1.Image) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateImage", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateImage indicates an expected call of CreateImage
func (mr *MockClientMockRecorder) CreateImage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateImage", reflect.TypeOf((*MockClient)(nil).CreateImage), arg0, arg1)
}

// CreateImageBeta mocks base method
func (m *MockClient) CreateImageBeta(arg0 string, arg1 *v0_beta.Image) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateImageBeta", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateImageBeta indicates an expected call of CreateImageBeta
func (mr *MockClientMockRecorder) CreateImageBeta(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateImageBeta", reflect.TypeOf((*MockClient)(nil).CreateImageBeta), arg0, arg1)
}

// CreateInstance mocks base method
func (m *MockClient) CreateInstance(arg0, arg1 string, arg2 *v1.Instance) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateInstance indicates an expected call of CreateInstance
func (mr *MockClientMockRecorder) CreateInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateInstance", reflect.TypeOf((*MockClient)(nil).CreateInstance), arg0, arg1, arg2)
}

// CreateNetwork mocks base method
func (m *MockClient) CreateNetwork(arg0 string, arg1 *v1.Network) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNetwork", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateNetwork indicates an expected call of CreateNetwork
func (mr *MockClientMockRecorder) CreateNetwork(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNetwork", reflect.TypeOf((*MockClient)(nil).CreateNetwork), arg0, arg1)
}

// CreateSubnetwork mocks base method
func (m *MockClient) CreateSubnetwork(arg0, arg1 string, arg2 *v1.Subnetwork) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSubnetwork", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateSubnetwork indicates an expected call of CreateSubnetwork
func (mr *MockClientMockRecorder) CreateSubnetwork(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSubnetwork", reflect.TypeOf((*MockClient)(nil).CreateSubnetwork), arg0, arg1, arg2)
}

// CreateTargetInstance mocks base method
func (m *MockClient) CreateTargetInstance(arg0, arg1 string, arg2 *v1.TargetInstance) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTargetInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTargetInstance indicates an expected call of CreateTargetInstance
func (mr *MockClientMockRecorder) CreateTargetInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTargetInstance", reflect.TypeOf((*MockClient)(nil).CreateTargetInstance), arg0, arg1, arg2)
}

// DeleteDisk mocks base method
func (m *MockClient) DeleteDisk(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDisk", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDisk indicates an expected call of DeleteDisk
func (mr *MockClientMockRecorder) DeleteDisk(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDisk", reflect.TypeOf((*MockClient)(nil).DeleteDisk), arg0, arg1, arg2)
}

// DeleteFirewallRule mocks base method
func (m *MockClient) DeleteFirewallRule(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFirewallRule", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFirewallRule indicates an expected call of DeleteFirewallRule
func (mr *MockClientMockRecorder) DeleteFirewallRule(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFirewallRule", reflect.TypeOf((*MockClient)(nil).DeleteFirewallRule), arg0, arg1)
}

// DeleteForwardingRule mocks base method
func (m *MockClient) DeleteForwardingRule(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteForwardingRule", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteForwardingRule indicates an expected call of DeleteForwardingRule
func (mr *MockClientMockRecorder) DeleteForwardingRule(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteForwardingRule", reflect.TypeOf((*MockClient)(nil).DeleteForwardingRule), arg0, arg1, arg2)
}

// DeleteImage mocks base method
func (m *MockClient) DeleteImage(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteImage", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteImage indicates an expected call of DeleteImage
func (mr *MockClientMockRecorder) DeleteImage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteImage", reflect.TypeOf((*MockClient)(nil).DeleteImage), arg0, arg1)
}

// DeleteInstance mocks base method
func (m *MockClient) DeleteInstance(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteInstance indicates an expected call of DeleteInstance
func (mr *MockClientMockRecorder) DeleteInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteInstance", reflect.TypeOf((*MockClient)(nil).DeleteInstance), arg0, arg1, arg2)
}

// DeleteNetwork mocks base method
func (m *MockClient) DeleteNetwork(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNetwork", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNetwork indicates an expected call of DeleteNetwork
func (mr *MockClientMockRecorder) DeleteNetwork(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNetwork", reflect.TypeOf((*MockClient)(nil).DeleteNetwork), arg0, arg1)
}

// DeleteSubnetwork mocks base method
func (m *MockClient) DeleteSubnetwork(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSubnetwork", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSubnetwork indicates an expected call of DeleteSubnetwork
func (mr *MockClientMockRecorder) DeleteSubnetwork(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSubnetwork", reflect.TypeOf((*MockClient)(nil).DeleteSubnetwork), arg0, arg1, arg2)
}

// DeleteTargetInstance mocks base method
func (m *MockClient) DeleteTargetInstance(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTargetInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTargetInstance indicates an expected call of DeleteTargetInstance
func (mr *MockClientMockRecorder) DeleteTargetInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTargetInstance", reflect.TypeOf((*MockClient)(nil).DeleteTargetInstance), arg0, arg1, arg2)
}

// DeprecateImage mocks base method
func (m *MockClient) DeprecateImage(arg0, arg1 string, arg2 *v1.DeprecationStatus) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeprecateImage", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeprecateImage indicates an expected call of DeprecateImage
func (mr *MockClientMockRecorder) DeprecateImage(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeprecateImage", reflect.TypeOf((*MockClient)(nil).DeprecateImage), arg0, arg1, arg2)
}

// DetachDisk mocks base method
func (m *MockClient) DetachDisk(arg0, arg1, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DetachDisk", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// DetachDisk indicates an expected call of DetachDisk
func (mr *MockClientMockRecorder) DetachDisk(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DetachDisk", reflect.TypeOf((*MockClient)(nil).DetachDisk), arg0, arg1, arg2, arg3)
}

// GetDisk mocks base method
func (m *MockClient) GetDisk(arg0, arg1, arg2 string) (*v1.Disk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDisk", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1.Disk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDisk indicates an expected call of GetDisk
func (mr *MockClientMockRecorder) GetDisk(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDisk", reflect.TypeOf((*MockClient)(nil).GetDisk), arg0, arg1, arg2)
}

// GetFirewallRule mocks base method
func (m *MockClient) GetFirewallRule(arg0, arg1 string) (*v1.Firewall, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFirewallRule", arg0, arg1)
	ret0, _ := ret[0].(*v1.Firewall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFirewallRule indicates an expected call of GetFirewallRule
func (mr *MockClientMockRecorder) GetFirewallRule(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFirewallRule", reflect.TypeOf((*MockClient)(nil).GetFirewallRule), arg0, arg1)
}

// GetForwardingRule mocks base method
func (m *MockClient) GetForwardingRule(arg0, arg1, arg2 string) (*v1.ForwardingRule, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetForwardingRule", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1.ForwardingRule)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetForwardingRule indicates an expected call of GetForwardingRule
func (mr *MockClientMockRecorder) GetForwardingRule(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetForwardingRule", reflect.TypeOf((*MockClient)(nil).GetForwardingRule), arg0, arg1, arg2)
}

// GetGuestAttributes mocks base method
func (m *MockClient) GetGuestAttributes(arg0, arg1, arg2, arg3, arg4 string) (*v0_beta.GuestAttributes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGuestAttributes", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(*v0_beta.GuestAttributes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGuestAttributes indicates an expected call of GetGuestAttributes
func (mr *MockClientMockRecorder) GetGuestAttributes(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGuestAttributes", reflect.TypeOf((*MockClient)(nil).GetGuestAttributes), arg0, arg1, arg2, arg3, arg4)
}

// GetImage mocks base method
func (m *MockClient) GetImage(arg0, arg1 string) (*v1.Image, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImage", arg0, arg1)
	ret0, _ := ret[0].(*v1.Image)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImage indicates an expected call of GetImage
func (mr *MockClientMockRecorder) GetImage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImage", reflect.TypeOf((*MockClient)(nil).GetImage), arg0, arg1)
}

// GetImageBeta mocks base method
func (m *MockClient) GetImageBeta(arg0, arg1 string) (*v0_beta.Image, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImageBeta", arg0, arg1)
	ret0, _ := ret[0].(*v0_beta.Image)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImageBeta indicates an expected call of GetImageBeta
func (mr *MockClientMockRecorder) GetImageBeta(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImageBeta", reflect.TypeOf((*MockClient)(nil).GetImageBeta), arg0, arg1)
}

// GetImageFromFamily mocks base method
func (m *MockClient) GetImageFromFamily(arg0, arg1 string) (*v1.Image, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImageFromFamily", arg0, arg1)
	ret0, _ := ret[0].(*v1.Image)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImageFromFamily indicates an expected call of GetImageFromFamily
func (mr *MockClientMockRecorder) GetImageFromFamily(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImageFromFamily", reflect.TypeOf((*MockClient)(nil).GetImageFromFamily), arg0, arg1)
}

// GetInstance mocks base method
func (m *MockClient) GetInstance(arg0, arg1, arg2 string) (*v1.Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstance indicates an expected call of GetInstance
func (mr *MockClientMockRecorder) GetInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstance", reflect.TypeOf((*MockClient)(nil).GetInstance), arg0, arg1, arg2)
}

// GetLicense mocks base method
func (m *MockClient) GetLicense(arg0, arg1 string) (*v1.License, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLicense", arg0, arg1)
	ret0, _ := ret[0].(*v1.License)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLicense indicates an expected call of GetLicense
func (mr *MockClientMockRecorder) GetLicense(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLicense", reflect.TypeOf((*MockClient)(nil).GetLicense), arg0, arg1)
}

// GetMachineType mocks base method
func (m *MockClient) GetMachineType(arg0, arg1, arg2 string) (*v1.MachineType, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMachineType", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1.MachineType)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMachineType indicates an expected call of GetMachineType
func (mr *MockClientMockRecorder) GetMachineType(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMachineType", reflect.TypeOf((*MockClient)(nil).GetMachineType), arg0, arg1, arg2)
}

// GetNetwork mocks base method
func (m *MockClient) GetNetwork(arg0, arg1 string) (*v1.Network, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetwork", arg0, arg1)
	ret0, _ := ret[0].(*v1.Network)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNetwork indicates an expected call of GetNetwork
func (mr *MockClientMockRecorder) GetNetwork(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetwork", reflect.TypeOf((*MockClient)(nil).GetNetwork), arg0, arg1)
}

// GetProject mocks base method
func (m *MockClient) GetProject(arg0 string) (*v1.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProject", arg0)
	ret0, _ := ret[0].(*v1.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProject indicates an expected call of GetProject
func (mr *MockClientMockRecorder) GetProject(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProject", reflect.TypeOf((*MockClient)(nil).GetProject), arg0)
}

// GetSerialPortOutput mocks base method
func (m *MockClient) GetSerialPortOutput(arg0, arg1, arg2 string, arg3, arg4 int64) (*v1.SerialPortOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSerialPortOutput", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(*v1.SerialPortOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSerialPortOutput indicates an expected call of GetSerialPortOutput
func (mr *MockClientMockRecorder) GetSerialPortOutput(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSerialPortOutput", reflect.TypeOf((*MockClient)(nil).GetSerialPortOutput), arg0, arg1, arg2, arg3, arg4)
}

// GetSubnetwork mocks base method
func (m *MockClient) GetSubnetwork(arg0, arg1, arg2 string) (*v1.Subnetwork, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSubnetwork", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1.Subnetwork)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSubnetwork indicates an expected call of GetSubnetwork
func (mr *MockClientMockRecorder) GetSubnetwork(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSubnetwork", reflect.TypeOf((*MockClient)(nil).GetSubnetwork), arg0, arg1, arg2)
}

// GetTargetInstance mocks base method
func (m *MockClient) GetTargetInstance(arg0, arg1, arg2 string) (*v1.TargetInstance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTargetInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1.TargetInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTargetInstance indicates an expected call of GetTargetInstance
func (mr *MockClientMockRecorder) GetTargetInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTargetInstance", reflect.TypeOf((*MockClient)(nil).GetTargetInstance), arg0, arg1, arg2)
}

// GetZone mocks base method
func (m *MockClient) GetZone(arg0, arg1 string) (*v1.Zone, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetZone", arg0, arg1)
	ret0, _ := ret[0].(*v1.Zone)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetZone indicates an expected call of GetZone
func (mr *MockClientMockRecorder) GetZone(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetZone", reflect.TypeOf((*MockClient)(nil).GetZone), arg0, arg1)
}

// InstanceStatus mocks base method
func (m *MockClient) InstanceStatus(arg0, arg1, arg2 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstanceStatus", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InstanceStatus indicates an expected call of InstanceStatus
func (mr *MockClientMockRecorder) InstanceStatus(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstanceStatus", reflect.TypeOf((*MockClient)(nil).InstanceStatus), arg0, arg1, arg2)
}

// InstanceStopped mocks base method
func (m *MockClient) InstanceStopped(arg0, arg1, arg2 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstanceStopped", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InstanceStopped indicates an expected call of InstanceStopped
func (mr *MockClientMockRecorder) InstanceStopped(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstanceStopped", reflect.TypeOf((*MockClient)(nil).InstanceStopped), arg0, arg1, arg2)
}

// ListDisks mocks base method
func (m *MockClient) ListDisks(arg0, arg1 string, arg2 ...compute.ListCallOption) ([]*v1.Disk, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListDisks", varargs...)
	ret0, _ := ret[0].([]*v1.Disk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDisks indicates an expected call of ListDisks
func (mr *MockClientMockRecorder) ListDisks(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDisks", reflect.TypeOf((*MockClient)(nil).ListDisks), varargs...)
}

// ListFirewallRules mocks base method
func (m *MockClient) ListFirewallRules(arg0 string, arg1 ...compute.ListCallOption) ([]*v1.Firewall, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListFirewallRules", varargs...)
	ret0, _ := ret[0].([]*v1.Firewall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListFirewallRules indicates an expected call of ListFirewallRules
func (mr *MockClientMockRecorder) ListFirewallRules(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFirewallRules", reflect.TypeOf((*MockClient)(nil).ListFirewallRules), varargs...)
}

// ListForwardingRules mocks base method
func (m *MockClient) ListForwardingRules(arg0, arg1 string, arg2 ...compute.ListCallOption) ([]*v1.ForwardingRule, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListForwardingRules", varargs...)
	ret0, _ := ret[0].([]*v1.ForwardingRule)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListForwardingRules indicates an expected call of ListForwardingRules
func (mr *MockClientMockRecorder) ListForwardingRules(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListForwardingRules", reflect.TypeOf((*MockClient)(nil).ListForwardingRules), varargs...)
}

// ListImages mocks base method
func (m *MockClient) ListImages(arg0 string, arg1 ...compute.ListCallOption) ([]*v1.Image, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListImages", varargs...)
	ret0, _ := ret[0].([]*v1.Image)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListImages indicates an expected call of ListImages
func (mr *MockClientMockRecorder) ListImages(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListImages", reflect.TypeOf((*MockClient)(nil).ListImages), varargs...)
}

// ListInstances mocks base method
func (m *MockClient) ListInstances(arg0, arg1 string, arg2 ...compute.ListCallOption) ([]*v1.Instance, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListInstances", varargs...)
	ret0, _ := ret[0].([]*v1.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListInstances indicates an expected call of ListInstances
func (mr *MockClientMockRecorder) ListInstances(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListInstances", reflect.TypeOf((*MockClient)(nil).ListInstances), varargs...)
}

// ListMachineTypes mocks base method
func (m *MockClient) ListMachineTypes(arg0, arg1 string, arg2 ...compute.ListCallOption) ([]*v1.MachineType, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListMachineTypes", varargs...)
	ret0, _ := ret[0].([]*v1.MachineType)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListMachineTypes indicates an expected call of ListMachineTypes
func (mr *MockClientMockRecorder) ListMachineTypes(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListMachineTypes", reflect.TypeOf((*MockClient)(nil).ListMachineTypes), varargs...)
}

// ListNetworks mocks base method
func (m *MockClient) ListNetworks(arg0 string, arg1 ...compute.ListCallOption) ([]*v1.Network, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListNetworks", varargs...)
	ret0, _ := ret[0].([]*v1.Network)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNetworks indicates an expected call of ListNetworks
func (mr *MockClientMockRecorder) ListNetworks(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNetworks", reflect.TypeOf((*MockClient)(nil).ListNetworks), varargs...)
}

// ListRegions mocks base method
func (m *MockClient) ListRegions(arg0 string, arg1 ...compute.ListCallOption) ([]*v1.Region, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListRegions", varargs...)
	ret0, _ := ret[0].([]*v1.Region)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRegions indicates an expected call of ListRegions
func (mr *MockClientMockRecorder) ListRegions(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRegions", reflect.TypeOf((*MockClient)(nil).ListRegions), varargs...)
}

// ListSubnetworks mocks base method
func (m *MockClient) ListSubnetworks(arg0, arg1 string, arg2 ...compute.ListCallOption) ([]*v1.Subnetwork, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListSubnetworks", varargs...)
	ret0, _ := ret[0].([]*v1.Subnetwork)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSubnetworks indicates an expected call of ListSubnetworks
func (mr *MockClientMockRecorder) ListSubnetworks(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSubnetworks", reflect.TypeOf((*MockClient)(nil).ListSubnetworks), varargs...)
}

// ListTargetInstances mocks base method
func (m *MockClient) ListTargetInstances(arg0, arg1 string, arg2 ...compute.ListCallOption) ([]*v1.TargetInstance, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListTargetInstances", varargs...)
	ret0, _ := ret[0].([]*v1.TargetInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTargetInstances indicates an expected call of ListTargetInstances
func (mr *MockClientMockRecorder) ListTargetInstances(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTargetInstances", reflect.TypeOf((*MockClient)(nil).ListTargetInstances), varargs...)
}

// ListZones mocks base method
func (m *MockClient) ListZones(arg0 string, arg1 ...compute.ListCallOption) ([]*v1.Zone, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListZones", varargs...)
	ret0, _ := ret[0].([]*v1.Zone)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListZones indicates an expected call of ListZones
func (mr *MockClientMockRecorder) ListZones(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListZones", reflect.TypeOf((*MockClient)(nil).ListZones), varargs...)
}

// ResizeDisk mocks base method
func (m *MockClient) ResizeDisk(arg0, arg1, arg2 string, arg3 *v1.DisksResizeRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResizeDisk", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// ResizeDisk indicates an expected call of ResizeDisk
func (mr *MockClientMockRecorder) ResizeDisk(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResizeDisk", reflect.TypeOf((*MockClient)(nil).ResizeDisk), arg0, arg1, arg2, arg3)
}

// Retry mocks base method
func (m *MockClient) Retry(arg0 func(...googleapi.CallOption) (*v1.Operation, error), arg1 ...googleapi.CallOption) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Retry", varargs...)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Retry indicates an expected call of Retry
func (mr *MockClientMockRecorder) Retry(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Retry", reflect.TypeOf((*MockClient)(nil).Retry), varargs...)
}

// RetryBeta mocks base method
func (m *MockClient) RetryBeta(arg0 func(...googleapi.CallOption) (*v0_beta.Operation, error), arg1 ...googleapi.CallOption) (*v0_beta.Operation, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RetryBeta", varargs...)
	ret0, _ := ret[0].(*v0_beta.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RetryBeta indicates an expected call of RetryBeta
func (mr *MockClientMockRecorder) RetryBeta(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetryBeta", reflect.TypeOf((*MockClient)(nil).RetryBeta), varargs...)
}

// SetCommonInstanceMetadata mocks base method
func (m *MockClient) SetCommonInstanceMetadata(arg0 string, arg1 *v1.Metadata) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetCommonInstanceMetadata", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetCommonInstanceMetadata indicates an expected call of SetCommonInstanceMetadata
func (mr *MockClientMockRecorder) SetCommonInstanceMetadata(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetCommonInstanceMetadata", reflect.TypeOf((*MockClient)(nil).SetCommonInstanceMetadata), arg0, arg1)
}

// SetInstanceMetadata mocks base method
func (m *MockClient) SetInstanceMetadata(arg0, arg1, arg2 string, arg3 *v1.Metadata) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetInstanceMetadata", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetInstanceMetadata indicates an expected call of SetInstanceMetadata
func (mr *MockClientMockRecorder) SetInstanceMetadata(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetInstanceMetadata", reflect.TypeOf((*MockClient)(nil).SetInstanceMetadata), arg0, arg1, arg2, arg3)
}

// StartInstance mocks base method
func (m *MockClient) StartInstance(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// StartInstance indicates an expected call of StartInstance
func (mr *MockClientMockRecorder) StartInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartInstance", reflect.TypeOf((*MockClient)(nil).StartInstance), arg0, arg1, arg2)
}

// StopInstance mocks base method
func (m *MockClient) StopInstance(arg0, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StopInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// StopInstance indicates an expected call of StopInstance
func (mr *MockClientMockRecorder) StopInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopInstance", reflect.TypeOf((*MockClient)(nil).StopInstance), arg0, arg1, arg2)
}

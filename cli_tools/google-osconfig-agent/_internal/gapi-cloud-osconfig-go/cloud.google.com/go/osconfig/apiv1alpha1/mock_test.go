// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// AUTO-GENERATED CODE. DO NOT EDIT.

package osconfig

import (
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	field_maskpb "google.golang.org/genproto/protobuf/field_mask"
)

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	gstatus "google.golang.org/grpc/status"
)

var _ = io.EOF
var _ = ptypes.MarshalAny
var _ status.Status

type mockOsConfigServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	osconfigpb.OsConfigServiceServer

	reqs []proto.Message

	// If set, all calls return this error.
	err error

	// responses to return if err == nil
	resps []proto.Message
}

func (s *mockOsConfigServer) CreateOsConfig(ctx context.Context, req *osconfigpb.CreateOsConfigRequest) (*osconfigpb.OsConfig, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.OsConfig), nil
}

func (s *mockOsConfigServer) GetOsConfig(ctx context.Context, req *osconfigpb.GetOsConfigRequest) (*osconfigpb.OsConfig, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.OsConfig), nil
}

func (s *mockOsConfigServer) ListOsConfigs(ctx context.Context, req *osconfigpb.ListOsConfigsRequest) (*osconfigpb.ListOsConfigsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.ListOsConfigsResponse), nil
}

func (s *mockOsConfigServer) UpdateOsConfig(ctx context.Context, req *osconfigpb.UpdateOsConfigRequest) (*osconfigpb.OsConfig, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.OsConfig), nil
}

func (s *mockOsConfigServer) DeleteOsConfig(ctx context.Context, req *osconfigpb.DeleteOsConfigRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*emptypb.Empty), nil
}

func (s *mockOsConfigServer) CreatePatchPolicy(ctx context.Context, req *osconfigpb.CreatePatchPolicyRequest) (*osconfigpb.PatchPolicy, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.PatchPolicy), nil
}

func (s *mockOsConfigServer) GetPatchPolicy(ctx context.Context, req *osconfigpb.GetPatchPolicyRequest) (*osconfigpb.PatchPolicy, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.PatchPolicy), nil
}

func (s *mockOsConfigServer) ListPatchPolicies(ctx context.Context, req *osconfigpb.ListPatchPoliciesRequest) (*osconfigpb.ListPatchPoliciesResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.ListPatchPoliciesResponse), nil
}

func (s *mockOsConfigServer) UpdatePatchPolicy(ctx context.Context, req *osconfigpb.UpdatePatchPolicyRequest) (*osconfigpb.PatchPolicy, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.PatchPolicy), nil
}

func (s *mockOsConfigServer) DeletePatchPolicy(ctx context.Context, req *osconfigpb.DeletePatchPolicyRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*emptypb.Empty), nil
}

func (s *mockOsConfigServer) CreateAssignment(ctx context.Context, req *osconfigpb.CreateAssignmentRequest) (*osconfigpb.Assignment, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.Assignment), nil
}

func (s *mockOsConfigServer) GetAssignment(ctx context.Context, req *osconfigpb.GetAssignmentRequest) (*osconfigpb.Assignment, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.Assignment), nil
}

func (s *mockOsConfigServer) ListAssignments(ctx context.Context, req *osconfigpb.ListAssignmentsRequest) (*osconfigpb.ListAssignmentsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.ListAssignmentsResponse), nil
}

func (s *mockOsConfigServer) UpdateAssignment(ctx context.Context, req *osconfigpb.UpdateAssignmentRequest) (*osconfigpb.Assignment, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.Assignment), nil
}

func (s *mockOsConfigServer) DeleteAssignment(ctx context.Context, req *osconfigpb.DeleteAssignmentRequest) (*emptypb.Empty, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*emptypb.Empty), nil
}

func (s *mockOsConfigServer) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*iampb.Policy), nil
}

func (s *mockOsConfigServer) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*iampb.Policy), nil
}

func (s *mockOsConfigServer) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*iampb.TestIamPermissionsResponse), nil
}

func (s *mockOsConfigServer) LookupConfigs(ctx context.Context, req *osconfigpb.LookupConfigsRequest) (*osconfigpb.LookupConfigsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*osconfigpb.LookupConfigsResponse), nil
}

// clientOpt is the option tests should use to connect to the test server.
// It is initialized by TestMain.
var clientOpt option.ClientOption

var (
	mockOsConfig mockOsConfigServer
)

func TestMain(m *testing.M) {
	flag.Parse()

	serv := grpc.NewServer()
	osconfigpb.RegisterOsConfigServiceServer(serv, &mockOsConfig)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	clientOpt = option.WithGRPCConn(conn)

	os.Exit(m.Run())
}

func TestOsConfigServiceCreateOsConfig(t *testing.T) {
	var name string = "name3373707"
	var description string = "description-1724546052"
	var expectedResponse = &osconfigpb.OsConfig{
		Name:        name,
		Description: description,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var osConfig *osconfigpb.OsConfig = &osconfigpb.OsConfig{}
	var request = &osconfigpb.CreateOsConfigRequest{
		Parent:   formattedParent,
		OsConfig: osConfig,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreateOsConfig(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceCreateOsConfigError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var osConfig *osconfigpb.OsConfig = &osconfigpb.OsConfig{}
	var request = &osconfigpb.CreateOsConfigRequest{
		Parent:   formattedParent,
		OsConfig: osConfig,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreateOsConfig(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceGetOsConfig(t *testing.T) {
	var name2 string = "name2-1052831874"
	var description string = "description-1724546052"
	var expectedResponse = &osconfigpb.OsConfig{
		Name:        name2,
		Description: description,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("organizations/%s/osConfigs/%s", "[ORGANIZATION]", "[OS_CONFIG]")
	var request = &osconfigpb.GetOsConfigRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetOsConfig(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceGetOsConfigError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("organizations/%s/osConfigs/%s", "[ORGANIZATION]", "[OS_CONFIG]")
	var request = &osconfigpb.GetOsConfigRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetOsConfig(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceListOsConfigs(t *testing.T) {
	var nextPageToken string = ""
	var osConfigsElement *osconfigpb.OsConfig = &osconfigpb.OsConfig{}
	var osConfigs = []*osconfigpb.OsConfig{osConfigsElement}
	var expectedResponse = &osconfigpb.ListOsConfigsResponse{
		NextPageToken: nextPageToken,
		OsConfigs:     osConfigs,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var request = &osconfigpb.ListOsConfigsRequest{
		Parent: formattedParent,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListOsConfigs(context.Background(), request).Next()

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	want := (interface{})(expectedResponse.OsConfigs[0])
	got := (interface{})(resp)
	var ok bool

	switch want := (want).(type) {
	case proto.Message:
		ok = proto.Equal(want, got.(proto.Message))
	default:
		ok = want == got
	}
	if !ok {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceListOsConfigsError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var request = &osconfigpb.ListOsConfigsRequest{
		Parent: formattedParent,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListOsConfigs(context.Background(), request).Next()

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceUpdateOsConfig(t *testing.T) {
	var name2 string = "name2-1052831874"
	var description string = "description-1724546052"
	var expectedResponse = &osconfigpb.OsConfig{
		Name:        name2,
		Description: description,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("organizations/%s/osConfigs/%s", "[ORGANIZATION]", "[OS_CONFIG]")
	var osConfig *osconfigpb.OsConfig = &osconfigpb.OsConfig{}
	var updateMask *field_maskpb.FieldMask = &field_maskpb.FieldMask{}
	var request = &osconfigpb.UpdateOsConfigRequest{
		Name:       formattedName,
		OsConfig:   osConfig,
		UpdateMask: updateMask,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.UpdateOsConfig(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceUpdateOsConfigError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("organizations/%s/osConfigs/%s", "[ORGANIZATION]", "[OS_CONFIG]")
	var osConfig *osconfigpb.OsConfig = &osconfigpb.OsConfig{}
	var updateMask *field_maskpb.FieldMask = &field_maskpb.FieldMask{}
	var request = &osconfigpb.UpdateOsConfigRequest{
		Name:       formattedName,
		OsConfig:   osConfig,
		UpdateMask: updateMask,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.UpdateOsConfig(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceDeleteOsConfig(t *testing.T) {
	var expectedResponse *emptypb.Empty = &emptypb.Empty{}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("organizations/%s/osConfigs/%s", "[ORGANIZATION]", "[OS_CONFIG]")
	var request = &osconfigpb.DeleteOsConfigRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DeleteOsConfig(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

}

func TestOsConfigServiceDeleteOsConfigError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("organizations/%s/osConfigs/%s", "[ORGANIZATION]", "[OS_CONFIG]")
	var request = &osconfigpb.DeleteOsConfigRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DeleteOsConfig(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
}
func TestOsConfigServiceCreatePatchPolicy(t *testing.T) {
	var name string = "name3373707"
	var description string = "description-1724546052"
	var expectedResponse = &osconfigpb.PatchPolicy{
		Name:        name,
		Description: description,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("projects/%s", "[PROJECT]")
	var patchPolicy *osconfigpb.PatchPolicy = &osconfigpb.PatchPolicy{}
	var request = &osconfigpb.CreatePatchPolicyRequest{
		Parent:      formattedParent,
		PatchPolicy: patchPolicy,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreatePatchPolicy(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceCreatePatchPolicyError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedParent string = fmt.Sprintf("projects/%s", "[PROJECT]")
	var patchPolicy *osconfigpb.PatchPolicy = &osconfigpb.PatchPolicy{}
	var request = &osconfigpb.CreatePatchPolicyRequest{
		Parent:      formattedParent,
		PatchPolicy: patchPolicy,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreatePatchPolicy(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceGetPatchPolicy(t *testing.T) {
	var name2 string = "name2-1052831874"
	var description string = "description-1724546052"
	var expectedResponse = &osconfigpb.PatchPolicy{
		Name:        name2,
		Description: description,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("projects/%s/patchPolicies/%s", "[PROJECT]", "[PATCH_POLICY]")
	var request = &osconfigpb.GetPatchPolicyRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetPatchPolicy(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceGetPatchPolicyError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("projects/%s/patchPolicies/%s", "[PROJECT]", "[PATCH_POLICY]")
	var request = &osconfigpb.GetPatchPolicyRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetPatchPolicy(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceListPatchPolicies(t *testing.T) {
	var nextPageToken string = ""
	var patchPoliciesElement *osconfigpb.PatchPolicy = &osconfigpb.PatchPolicy{}
	var patchPolicies = []*osconfigpb.PatchPolicy{patchPoliciesElement}
	var expectedResponse = &osconfigpb.ListPatchPoliciesResponse{
		NextPageToken: nextPageToken,
		PatchPolicies: patchPolicies,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("projects/%s", "[PROJECT]")
	var request = &osconfigpb.ListPatchPoliciesRequest{
		Parent: formattedParent,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListPatchPolicies(context.Background(), request).Next()

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	want := (interface{})(expectedResponse.PatchPolicies[0])
	got := (interface{})(resp)
	var ok bool

	switch want := (want).(type) {
	case proto.Message:
		ok = proto.Equal(want, got.(proto.Message))
	default:
		ok = want == got
	}
	if !ok {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceListPatchPoliciesError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedParent string = fmt.Sprintf("projects/%s", "[PROJECT]")
	var request = &osconfigpb.ListPatchPoliciesRequest{
		Parent: formattedParent,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListPatchPolicies(context.Background(), request).Next()

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceUpdatePatchPolicy(t *testing.T) {
	var name2 string = "name2-1052831874"
	var description string = "description-1724546052"
	var expectedResponse = &osconfigpb.PatchPolicy{
		Name:        name2,
		Description: description,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("projects/%s/patchPolicies/%s", "[PROJECT]", "[PATCH_POLICY]")
	var patchPolicy *osconfigpb.PatchPolicy = &osconfigpb.PatchPolicy{}
	var updateMask *field_maskpb.FieldMask = &field_maskpb.FieldMask{}
	var request = &osconfigpb.UpdatePatchPolicyRequest{
		Name:        formattedName,
		PatchPolicy: patchPolicy,
		UpdateMask:  updateMask,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.UpdatePatchPolicy(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceUpdatePatchPolicyError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("projects/%s/patchPolicies/%s", "[PROJECT]", "[PATCH_POLICY]")
	var patchPolicy *osconfigpb.PatchPolicy = &osconfigpb.PatchPolicy{}
	var updateMask *field_maskpb.FieldMask = &field_maskpb.FieldMask{}
	var request = &osconfigpb.UpdatePatchPolicyRequest{
		Name:        formattedName,
		PatchPolicy: patchPolicy,
		UpdateMask:  updateMask,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.UpdatePatchPolicy(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceDeletePatchPolicy(t *testing.T) {
	var expectedResponse *emptypb.Empty = &emptypb.Empty{}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("projects/%s/patchPolicies/%s", "[PROJECT]", "[PATCH_POLICY]")
	var request = &osconfigpb.DeletePatchPolicyRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DeletePatchPolicy(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

}

func TestOsConfigServiceDeletePatchPolicyError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("projects/%s/patchPolicies/%s", "[PROJECT]", "[PATCH_POLICY]")
	var request = &osconfigpb.DeletePatchPolicyRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DeletePatchPolicy(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
}
func TestOsConfigServiceCreateAssignment(t *testing.T) {
	var name string = "name3373707"
	var description string = "description-1724546052"
	var expression string = "expression-1795452264"
	var expectedResponse = &osconfigpb.Assignment{
		Name:        name,
		Description: description,
		Expression:  expression,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var assignment *osconfigpb.Assignment = &osconfigpb.Assignment{}
	var request = &osconfigpb.CreateAssignmentRequest{
		Parent:     formattedParent,
		Assignment: assignment,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreateAssignment(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceCreateAssignmentError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var assignment *osconfigpb.Assignment = &osconfigpb.Assignment{}
	var request = &osconfigpb.CreateAssignmentRequest{
		Parent:     formattedParent,
		Assignment: assignment,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.CreateAssignment(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceGetAssignment(t *testing.T) {
	var name2 string = "name2-1052831874"
	var description string = "description-1724546052"
	var expression string = "expression-1795452264"
	var expectedResponse = &osconfigpb.Assignment{
		Name:        name2,
		Description: description,
		Expression:  expression,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("organizations/%s/assignments/%s", "[ORGANIZATION]", "[ASSIGNMENT]")
	var request = &osconfigpb.GetAssignmentRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetAssignment(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceGetAssignmentError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("organizations/%s/assignments/%s", "[ORGANIZATION]", "[ASSIGNMENT]")
	var request = &osconfigpb.GetAssignmentRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetAssignment(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceListAssignments(t *testing.T) {
	var nextPageToken string = ""
	var assignmentsElement *osconfigpb.Assignment = &osconfigpb.Assignment{}
	var assignments = []*osconfigpb.Assignment{assignmentsElement}
	var expectedResponse = &osconfigpb.ListAssignmentsResponse{
		NextPageToken: nextPageToken,
		Assignments:   assignments,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var request = &osconfigpb.ListAssignmentsRequest{
		Parent: formattedParent,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListAssignments(context.Background(), request).Next()

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	want := (interface{})(expectedResponse.Assignments[0])
	got := (interface{})(resp)
	var ok bool

	switch want := (want).(type) {
	case proto.Message:
		ok = proto.Equal(want, got.(proto.Message))
	default:
		ok = want == got
	}
	if !ok {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceListAssignmentsError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedParent string = fmt.Sprintf("organizations/%s", "[ORGANIZATION]")
	var request = &osconfigpb.ListAssignmentsRequest{
		Parent: formattedParent,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ListAssignments(context.Background(), request).Next()

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceUpdateAssignment(t *testing.T) {
	var name2 string = "name2-1052831874"
	var description string = "description-1724546052"
	var expression string = "expression-1795452264"
	var expectedResponse = &osconfigpb.Assignment{
		Name:        name2,
		Description: description,
		Expression:  expression,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("organizations/%s/assignments/%s", "[ORGANIZATION]", "[ASSIGNMENT]")
	var assignment *osconfigpb.Assignment = &osconfigpb.Assignment{}
	var updateMask *field_maskpb.FieldMask = &field_maskpb.FieldMask{}
	var request = &osconfigpb.UpdateAssignmentRequest{
		Name:       formattedName,
		Assignment: assignment,
		UpdateMask: updateMask,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.UpdateAssignment(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceUpdateAssignmentError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("organizations/%s/assignments/%s", "[ORGANIZATION]", "[ASSIGNMENT]")
	var assignment *osconfigpb.Assignment = &osconfigpb.Assignment{}
	var updateMask *field_maskpb.FieldMask = &field_maskpb.FieldMask{}
	var request = &osconfigpb.UpdateAssignmentRequest{
		Name:       formattedName,
		Assignment: assignment,
		UpdateMask: updateMask,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.UpdateAssignment(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceDeleteAssignment(t *testing.T) {
	var expectedResponse *emptypb.Empty = &emptypb.Empty{}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("organizations/%s/assignments/%s", "[ORGANIZATION]", "[ASSIGNMENT]")
	var request = &osconfigpb.DeleteAssignmentRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DeleteAssignment(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

}

func TestOsConfigServiceDeleteAssignmentError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedName string = fmt.Sprintf("organizations/%s/assignments/%s", "[ORGANIZATION]", "[ASSIGNMENT]")
	var request = &osconfigpb.DeleteAssignmentRequest{
		Name: formattedName,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	err = c.DeleteAssignment(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
}
func TestOsConfigServiceGetIamPolicy(t *testing.T) {
	var version int32 = 351608024
	var etag []byte = []byte("21")
	var expectedResponse = &iampb.Policy{
		Version: version,
		Etag:    etag,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedResource string = fmt.Sprintf("organizations/%s", "[ORGANIZATION_PATH]")
	var request = &iampb.GetIamPolicyRequest{
		Resource: formattedResource,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetIamPolicy(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceGetIamPolicyError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedResource string = fmt.Sprintf("organizations/%s", "[ORGANIZATION_PATH]")
	var request = &iampb.GetIamPolicyRequest{
		Resource: formattedResource,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetIamPolicy(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceSetIamPolicy(t *testing.T) {
	var version int32 = 351608024
	var etag []byte = []byte("21")
	var expectedResponse = &iampb.Policy{
		Version: version,
		Etag:    etag,
	}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedResource string = fmt.Sprintf("organizations/%s", "[ORGANIZATION_PATH]")
	var policy *iampb.Policy = &iampb.Policy{}
	var request = &iampb.SetIamPolicyRequest{
		Resource: formattedResource,
		Policy:   policy,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.SetIamPolicy(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceSetIamPolicyError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedResource string = fmt.Sprintf("organizations/%s", "[ORGANIZATION_PATH]")
	var policy *iampb.Policy = &iampb.Policy{}
	var request = &iampb.SetIamPolicyRequest{
		Resource: formattedResource,
		Policy:   policy,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.SetIamPolicy(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceTestIamPermissions(t *testing.T) {
	var expectedResponse *iampb.TestIamPermissionsResponse = &iampb.TestIamPermissionsResponse{}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedResource string = fmt.Sprintf("organizations/%s", "[ORGANIZATION_PATH]")
	var permissions []string = nil
	var request = &iampb.TestIamPermissionsRequest{
		Resource:    formattedResource,
		Permissions: permissions,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.TestIamPermissions(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceTestIamPermissionsError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedResource string = fmt.Sprintf("organizations/%s", "[ORGANIZATION_PATH]")
	var permissions []string = nil
	var request = &iampb.TestIamPermissionsRequest{
		Resource:    formattedResource,
		Permissions: permissions,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.TestIamPermissions(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}
func TestOsConfigServiceLookupConfigs(t *testing.T) {
	var expectedResponse *osconfigpb.LookupConfigsResponse = &osconfigpb.LookupConfigsResponse{}

	mockOsConfig.err = nil
	mockOsConfig.reqs = nil

	mockOsConfig.resps = append(mockOsConfig.resps[:0], expectedResponse)

	var formattedResource string = fmt.Sprintf("projects/%s/zones/%s/instances/%s", "[PROJECT]", "[ZONE]", "[INSTANCE]")
	var osInfo *osconfigpb.LookupConfigsRequest_OsInfo = &osconfigpb.LookupConfigsRequest_OsInfo{}
	var configTypes []osconfigpb.LookupConfigsRequest_ConfigType = nil
	var request = &osconfigpb.LookupConfigsRequest{
		Resource:    formattedResource,
		OsInfo:      osInfo,
		ConfigTypes: configTypes,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.LookupConfigs(context.Background(), request)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockOsConfig.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}

	if want, got := expectedResponse, resp; !proto.Equal(want, got) {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestOsConfigServiceLookupConfigsError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockOsConfig.err = gstatus.Error(errCode, "test error")

	var formattedResource string = fmt.Sprintf("projects/%s/zones/%s/instances/%s", "[PROJECT]", "[ZONE]", "[INSTANCE]")
	var osInfo *osconfigpb.LookupConfigsRequest_OsInfo = &osconfigpb.LookupConfigsRequest_OsInfo{}
	var configTypes []osconfigpb.LookupConfigsRequest_ConfigType = nil
	var request = &osconfigpb.LookupConfigsRequest{
		Resource:    formattedResource,
		OsInfo:      osInfo,
		ConfigTypes: configTypes,
	}

	c, err := NewClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.LookupConfigs(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}

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
	"math"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/proto"
	gax "github.com/googleapis/gax-go"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// CallOptions contains the retry settings for each method of Client.
type CallOptions struct {
	CreateOsConfig     []gax.CallOption
	GetOsConfig        []gax.CallOption
	ListOsConfigs      []gax.CallOption
	UpdateOsConfig     []gax.CallOption
	DeleteOsConfig     []gax.CallOption
	CreatePatchPolicy  []gax.CallOption
	GetPatchPolicy     []gax.CallOption
	ListPatchPolicies  []gax.CallOption
	UpdatePatchPolicy  []gax.CallOption
	DeletePatchPolicy  []gax.CallOption
	CreateAssignment   []gax.CallOption
	GetAssignment      []gax.CallOption
	ListAssignments    []gax.CallOption
	UpdateAssignment   []gax.CallOption
	DeleteAssignment   []gax.CallOption
	GetIamPolicy       []gax.CallOption
	SetIamPolicy       []gax.CallOption
	TestIamPermissions []gax.CallOption
	LookupConfigs      []gax.CallOption
}

func defaultClientOptions() []option.ClientOption {
	return []option.ClientOption{
		option.WithEndpoint("osconfig.googleapis.com:443"),
		option.WithScopes(DefaultAuthScopes()...),
	}
}

func defaultCallOptions() *CallOptions {
	retry := map[[2]string][]gax.CallOption{
		{"default", "idempotent"}: {
			gax.WithRetry(func() gax.Retryer {
				return gax.OnCodes([]codes.Code{
					codes.DeadlineExceeded,
					codes.Unavailable,
				}, gax.Backoff{
					Initial:    100 * time.Millisecond,
					Max:        60000 * time.Millisecond,
					Multiplier: 1.3,
				})
			}),
		},
	}
	return &CallOptions{
		CreateOsConfig:     retry[[2]string{"default", "non_idempotent"}],
		GetOsConfig:        retry[[2]string{"default", "idempotent"}],
		ListOsConfigs:      retry[[2]string{"default", "idempotent"}],
		UpdateOsConfig:     retry[[2]string{"default", "non_idempotent"}],
		DeleteOsConfig:     retry[[2]string{"default", "idempotent"}],
		CreatePatchPolicy:  retry[[2]string{"default", "non_idempotent"}],
		GetPatchPolicy:     retry[[2]string{"default", "idempotent"}],
		ListPatchPolicies:  retry[[2]string{"default", "idempotent"}],
		UpdatePatchPolicy:  retry[[2]string{"default", "non_idempotent"}],
		DeletePatchPolicy:  retry[[2]string{"default", "idempotent"}],
		CreateAssignment:   retry[[2]string{"default", "non_idempotent"}],
		GetAssignment:      retry[[2]string{"default", "idempotent"}],
		ListAssignments:    retry[[2]string{"default", "idempotent"}],
		UpdateAssignment:   retry[[2]string{"default", "non_idempotent"}],
		DeleteAssignment:   retry[[2]string{"default", "idempotent"}],
		GetIamPolicy:       retry[[2]string{"default", "non_idempotent"}],
		SetIamPolicy:       retry[[2]string{"default", "non_idempotent"}],
		TestIamPermissions: retry[[2]string{"default", "non_idempotent"}],
		LookupConfigs:      retry[[2]string{"default", "non_idempotent"}],
	}
}

// Client is a client for interacting with Cloud OS Config API.
//
// Methods, except Close, may be called concurrently. However, fields must not be modified concurrently with method calls.
type Client struct {
	// The connection to the service.
	conn *grpc.ClientConn

	// The gRPC API client.
	client osconfigpb.OsConfigServiceClient

	// The call options for this service.
	CallOptions *CallOptions

	// The x-goog-* metadata to be sent with each request.
	xGoogMetadata metadata.MD
}

// NewClient creates a new os config service client.
//
// OS Config API
//
// The OS Config service is the server-side component that allows users to
// manage package installations and patch policies for virtual machines.
func NewClient(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	conn, err := transport.DialGRPC(ctx, append(defaultClientOptions(), opts...)...)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:        conn,
		CallOptions: defaultCallOptions(),

		client: osconfigpb.NewOsConfigServiceClient(conn),
	}
	c.setGoogleClientInfo()
	return c, nil
}

// Connection returns the client's connection to the API service.
func (c *Client) Connection() *grpc.ClientConn {
	return c.conn
}

// Close closes the connection to the API service. The user should invoke this when
// the client is no longer required.
func (c *Client) Close() error {
	return c.conn.Close()
}

// setGoogleClientInfo sets the name and version of the application in
// the `x-goog-api-client` header passed on each request. Intended for
// use by Google-written clients.
func (c *Client) setGoogleClientInfo(keyval ...string) {
	kv := append([]string{"gax", gax.Version, "grpc", grpc.Version}, keyval...)
	c.xGoogMetadata = metadata.Pairs("x-goog-api-client", gax.XGoogHeader(kv...))
}

// CreateOsConfig create an OsConfig.
func (c *Client) CreateOsConfig(ctx context.Context, req *osconfigpb.CreateOsConfigRequest, opts ...gax.CallOption) (*osconfigpb.OsConfig, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.CreateOsConfig[0:len(c.CallOptions.CreateOsConfig):len(c.CallOptions.CreateOsConfig)], opts...)
	var resp *osconfigpb.OsConfig
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.CreateOsConfig(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOsConfig get an OsConfig.
func (c *Client) GetOsConfig(ctx context.Context, req *osconfigpb.GetOsConfigRequest, opts ...gax.CallOption) (*osconfigpb.OsConfig, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.GetOsConfig[0:len(c.CallOptions.GetOsConfig):len(c.CallOptions.GetOsConfig)], opts...)
	var resp *osconfigpb.OsConfig
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.GetOsConfig(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListOsConfigs get a page of OsConfigs.
func (c *Client) ListOsConfigs(ctx context.Context, req *osconfigpb.ListOsConfigsRequest, opts ...gax.CallOption) *OsConfigIterator {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.ListOsConfigs[0:len(c.CallOptions.ListOsConfigs):len(c.CallOptions.ListOsConfigs)], opts...)
	it := &OsConfigIterator{}
	req = proto.Clone(req).(*osconfigpb.ListOsConfigsRequest)
	it.InternalFetch = func(pageSize int, pageToken string) ([]*osconfigpb.OsConfig, string, error) {
		var resp *osconfigpb.ListOsConfigsResponse
		req.PageToken = pageToken
		if pageSize > math.MaxInt32 {
			req.PageSize = math.MaxInt32
		} else {
			req.PageSize = int32(pageSize)
		}
		err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
			var err error
			resp, err = c.client.ListOsConfigs(ctx, req, settings.GRPC...)
			return err
		}, opts...)
		if err != nil {
			return nil, "", err
		}
		return resp.OsConfigs, resp.NextPageToken, nil
	}
	fetch := func(pageSize int, pageToken string) (string, error) {
		items, nextPageToken, err := it.InternalFetch(pageSize, pageToken)
		if err != nil {
			return "", err
		}
		it.items = append(it.items, items...)
		return nextPageToken, nil
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(fetch, it.bufLen, it.takeBuf)
	it.pageInfo.MaxSize = int(req.PageSize)
	return it
}

// UpdateOsConfig update an OsConfig.
func (c *Client) UpdateOsConfig(ctx context.Context, req *osconfigpb.UpdateOsConfigRequest, opts ...gax.CallOption) (*osconfigpb.OsConfig, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.UpdateOsConfig[0:len(c.CallOptions.UpdateOsConfig):len(c.CallOptions.UpdateOsConfig)], opts...)
	var resp *osconfigpb.OsConfig
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.UpdateOsConfig(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteOsConfig delete an OsConfig.
func (c *Client) DeleteOsConfig(ctx context.Context, req *osconfigpb.DeleteOsConfigRequest, opts ...gax.CallOption) error {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.DeleteOsConfig[0:len(c.CallOptions.DeleteOsConfig):len(c.CallOptions.DeleteOsConfig)], opts...)
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		_, err = c.client.DeleteOsConfig(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	return err
}

// CreatePatchPolicy create an OS Config PatchPolicy.
func (c *Client) CreatePatchPolicy(ctx context.Context, req *osconfigpb.CreatePatchPolicyRequest, opts ...gax.CallOption) (*osconfigpb.PatchPolicy, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.CreatePatchPolicy[0:len(c.CallOptions.CreatePatchPolicy):len(c.CallOptions.CreatePatchPolicy)], opts...)
	var resp *osconfigpb.PatchPolicy
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.CreatePatchPolicy(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPatchPolicy get a PatchPolicy.
func (c *Client) GetPatchPolicy(ctx context.Context, req *osconfigpb.GetPatchPolicyRequest, opts ...gax.CallOption) (*osconfigpb.PatchPolicy, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.GetPatchPolicy[0:len(c.CallOptions.GetPatchPolicy):len(c.CallOptions.GetPatchPolicy)], opts...)
	var resp *osconfigpb.PatchPolicy
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.GetPatchPolicy(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListPatchPolicies get a page of PatchPolicies.
func (c *Client) ListPatchPolicies(ctx context.Context, req *osconfigpb.ListPatchPoliciesRequest, opts ...gax.CallOption) *PatchPolicyIterator {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.ListPatchPolicies[0:len(c.CallOptions.ListPatchPolicies):len(c.CallOptions.ListPatchPolicies)], opts...)
	it := &PatchPolicyIterator{}
	req = proto.Clone(req).(*osconfigpb.ListPatchPoliciesRequest)
	it.InternalFetch = func(pageSize int, pageToken string) ([]*osconfigpb.PatchPolicy, string, error) {
		var resp *osconfigpb.ListPatchPoliciesResponse
		req.PageToken = pageToken
		if pageSize > math.MaxInt32 {
			req.PageSize = math.MaxInt32
		} else {
			req.PageSize = int32(pageSize)
		}
		err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
			var err error
			resp, err = c.client.ListPatchPolicies(ctx, req, settings.GRPC...)
			return err
		}, opts...)
		if err != nil {
			return nil, "", err
		}
		return resp.PatchPolicies, resp.NextPageToken, nil
	}
	fetch := func(pageSize int, pageToken string) (string, error) {
		items, nextPageToken, err := it.InternalFetch(pageSize, pageToken)
		if err != nil {
			return "", err
		}
		it.items = append(it.items, items...)
		return nextPageToken, nil
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(fetch, it.bufLen, it.takeBuf)
	it.pageInfo.MaxSize = int(req.PageSize)
	return it
}

// UpdatePatchPolicy update a PatchPolicy.
func (c *Client) UpdatePatchPolicy(ctx context.Context, req *osconfigpb.UpdatePatchPolicyRequest, opts ...gax.CallOption) (*osconfigpb.PatchPolicy, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.UpdatePatchPolicy[0:len(c.CallOptions.UpdatePatchPolicy):len(c.CallOptions.UpdatePatchPolicy)], opts...)
	var resp *osconfigpb.PatchPolicy
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.UpdatePatchPolicy(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DeletePatchPolicy delete a PatchPolicy.
func (c *Client) DeletePatchPolicy(ctx context.Context, req *osconfigpb.DeletePatchPolicyRequest, opts ...gax.CallOption) error {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.DeletePatchPolicy[0:len(c.CallOptions.DeletePatchPolicy):len(c.CallOptions.DeletePatchPolicy)], opts...)
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		_, err = c.client.DeletePatchPolicy(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	return err
}

// CreateAssignment create an OS Config Assignment.
func (c *Client) CreateAssignment(ctx context.Context, req *osconfigpb.CreateAssignmentRequest, opts ...gax.CallOption) (*osconfigpb.Assignment, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.CreateAssignment[0:len(c.CallOptions.CreateAssignment):len(c.CallOptions.CreateAssignment)], opts...)
	var resp *osconfigpb.Assignment
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.CreateAssignment(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetAssignment get an OS Config Assignment.
func (c *Client) GetAssignment(ctx context.Context, req *osconfigpb.GetAssignmentRequest, opts ...gax.CallOption) (*osconfigpb.Assignment, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.GetAssignment[0:len(c.CallOptions.GetAssignment):len(c.CallOptions.GetAssignment)], opts...)
	var resp *osconfigpb.Assignment
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.GetAssignment(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ListAssignments get a page of OS Config Assignments.
func (c *Client) ListAssignments(ctx context.Context, req *osconfigpb.ListAssignmentsRequest, opts ...gax.CallOption) *AssignmentIterator {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.ListAssignments[0:len(c.CallOptions.ListAssignments):len(c.CallOptions.ListAssignments)], opts...)
	it := &AssignmentIterator{}
	req = proto.Clone(req).(*osconfigpb.ListAssignmentsRequest)
	it.InternalFetch = func(pageSize int, pageToken string) ([]*osconfigpb.Assignment, string, error) {
		var resp *osconfigpb.ListAssignmentsResponse
		req.PageToken = pageToken
		if pageSize > math.MaxInt32 {
			req.PageSize = math.MaxInt32
		} else {
			req.PageSize = int32(pageSize)
		}
		err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
			var err error
			resp, err = c.client.ListAssignments(ctx, req, settings.GRPC...)
			return err
		}, opts...)
		if err != nil {
			return nil, "", err
		}
		return resp.Assignments, resp.NextPageToken, nil
	}
	fetch := func(pageSize int, pageToken string) (string, error) {
		items, nextPageToken, err := it.InternalFetch(pageSize, pageToken)
		if err != nil {
			return "", err
		}
		it.items = append(it.items, items...)
		return nextPageToken, nil
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(fetch, it.bufLen, it.takeBuf)
	it.pageInfo.MaxSize = int(req.PageSize)
	return it
}

// UpdateAssignment update an OS Config Assignment.
func (c *Client) UpdateAssignment(ctx context.Context, req *osconfigpb.UpdateAssignmentRequest, opts ...gax.CallOption) (*osconfigpb.Assignment, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.UpdateAssignment[0:len(c.CallOptions.UpdateAssignment):len(c.CallOptions.UpdateAssignment)], opts...)
	var resp *osconfigpb.Assignment
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.UpdateAssignment(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteAssignment delete an OS Config Assignment.
func (c *Client) DeleteAssignment(ctx context.Context, req *osconfigpb.DeleteAssignmentRequest, opts ...gax.CallOption) error {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.DeleteAssignment[0:len(c.CallOptions.DeleteAssignment):len(c.CallOptions.DeleteAssignment)], opts...)
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		_, err = c.client.DeleteAssignment(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	return err
}

// GetIamPolicy gets the access control policy for an OsConfig or an OS Config Assignment.
// Returns NOT_FOUND error if the OsConfig does not exist. Returns an empty
// policy if the resource exists but does not have a policy set.
func (c *Client) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.GetIamPolicy[0:len(c.CallOptions.GetIamPolicy):len(c.CallOptions.GetIamPolicy)], opts...)
	var resp *iampb.Policy
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.GetIamPolicy(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SetIamPolicy sets the access control policy for an OsConfig or an OS Config Assignment.
// Replaces any existing policy.
func (c *Client) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.SetIamPolicy[0:len(c.CallOptions.SetIamPolicy):len(c.CallOptions.SetIamPolicy)], opts...)
	var resp *iampb.Policy
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.SetIamPolicy(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// TestIamPermissions test the access control policy for an OsConfig or an OS Config Assignment.
func (c *Client) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest, opts ...gax.CallOption) (*iampb.TestIamPermissionsResponse, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.TestIamPermissions[0:len(c.CallOptions.TestIamPermissions):len(c.CallOptions.TestIamPermissions)], opts...)
	var resp *iampb.TestIamPermissionsResponse
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.TestIamPermissions(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// LookupConfigs lookup the configs that are assigned to an instance. This lookup
// will merge all configs that are assigned to the instance resolving
// conflicts as necessary.
// This is usually called by the agent running on the instance but can be
// called directly to see what configs
// This
func (c *Client) LookupConfigs(ctx context.Context, req *osconfigpb.LookupConfigsRequest, opts ...gax.CallOption) (*osconfigpb.LookupConfigsResponse, error) {
	ctx = insertMetadata(ctx, c.xGoogMetadata)
	opts = append(c.CallOptions.LookupConfigs[0:len(c.CallOptions.LookupConfigs):len(c.CallOptions.LookupConfigs)], opts...)
	var resp *osconfigpb.LookupConfigsResponse
	err := gax.Invoke(ctx, func(ctx context.Context, settings gax.CallSettings) error {
		var err error
		resp, err = c.client.LookupConfigs(ctx, req, settings.GRPC...)
		return err
	}, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// AssignmentIterator manages a stream of *osconfigpb.Assignment.
type AssignmentIterator struct {
	items    []*osconfigpb.Assignment
	pageInfo *iterator.PageInfo
	nextFunc func() error

	// InternalFetch is for use by the Google Cloud Libraries only.
	// It is not part of the stable interface of this package.
	//
	// InternalFetch returns results from a single call to the underlying RPC.
	// The number of results is no greater than pageSize.
	// If there are no more results, nextPageToken is empty and err is nil.
	InternalFetch func(pageSize int, pageToken string) (results []*osconfigpb.Assignment, nextPageToken string, err error)
}

// PageInfo supports pagination. See the google.golang.org/api/iterator package for details.
func (it *AssignmentIterator) PageInfo() *iterator.PageInfo {
	return it.pageInfo
}

// Next returns the next result. Its second return value is iterator.Done if there are no more
// results. Once Next returns Done, all subsequent calls will return Done.
func (it *AssignmentIterator) Next() (*osconfigpb.Assignment, error) {
	var item *osconfigpb.Assignment
	if err := it.nextFunc(); err != nil {
		return item, err
	}
	item = it.items[0]
	it.items = it.items[1:]
	return item, nil
}

func (it *AssignmentIterator) bufLen() int {
	return len(it.items)
}

func (it *AssignmentIterator) takeBuf() interface{} {
	b := it.items
	it.items = nil
	return b
}

// OsConfigIterator manages a stream of *osconfigpb.OsConfig.
type OsConfigIterator struct {
	items    []*osconfigpb.OsConfig
	pageInfo *iterator.PageInfo
	nextFunc func() error

	// InternalFetch is for use by the Google Cloud Libraries only.
	// It is not part of the stable interface of this package.
	//
	// InternalFetch returns results from a single call to the underlying RPC.
	// The number of results is no greater than pageSize.
	// If there are no more results, nextPageToken is empty and err is nil.
	InternalFetch func(pageSize int, pageToken string) (results []*osconfigpb.OsConfig, nextPageToken string, err error)
}

// PageInfo supports pagination. See the google.golang.org/api/iterator package for details.
func (it *OsConfigIterator) PageInfo() *iterator.PageInfo {
	return it.pageInfo
}

// Next returns the next result. Its second return value is iterator.Done if there are no more
// results. Once Next returns Done, all subsequent calls will return Done.
func (it *OsConfigIterator) Next() (*osconfigpb.OsConfig, error) {
	var item *osconfigpb.OsConfig
	if err := it.nextFunc(); err != nil {
		return item, err
	}
	item = it.items[0]
	it.items = it.items[1:]
	return item, nil
}

func (it *OsConfigIterator) bufLen() int {
	return len(it.items)
}

func (it *OsConfigIterator) takeBuf() interface{} {
	b := it.items
	it.items = nil
	return b
}

// PatchPolicyIterator manages a stream of *osconfigpb.PatchPolicy.
type PatchPolicyIterator struct {
	items    []*osconfigpb.PatchPolicy
	pageInfo *iterator.PageInfo
	nextFunc func() error

	// InternalFetch is for use by the Google Cloud Libraries only.
	// It is not part of the stable interface of this package.
	//
	// InternalFetch returns results from a single call to the underlying RPC.
	// The number of results is no greater than pageSize.
	// If there are no more results, nextPageToken is empty and err is nil.
	InternalFetch func(pageSize int, pageToken string) (results []*osconfigpb.PatchPolicy, nextPageToken string, err error)
}

// PageInfo supports pagination. See the google.golang.org/api/iterator package for details.
func (it *PatchPolicyIterator) PageInfo() *iterator.PageInfo {
	return it.pageInfo
}

// Next returns the next result. Its second return value is iterator.Done if there are no more
// results. Once Next returns Done, all subsequent calls will return Done.
func (it *PatchPolicyIterator) Next() (*osconfigpb.PatchPolicy, error) {
	var item *osconfigpb.PatchPolicy
	if err := it.nextFunc(); err != nil {
		return item, err
	}
	item = it.items[0]
	it.items = it.items[1:]
	return item, nil
}

func (it *PatchPolicyIterator) bufLen() int {
	return len(it.items)
}

func (it *PatchPolicyIterator) takeBuf() interface{} {
	b := it.items
	it.items = nil
	return b
}

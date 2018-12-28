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

package osconfig_test

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func ExampleNewClient() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use client.
	_ = c
}

func ExampleClient_CreateOsConfig() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.CreateOsConfigRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.CreateOsConfig(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_GetOsConfig() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.GetOsConfigRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.GetOsConfig(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_ListOsConfigs() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.ListOsConfigsRequest{
		// TODO: Fill request struct fields.
	}
	it := c.ListOsConfigs(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// TODO: Handle error.
		}
		// TODO: Use resp.
		_ = resp
	}
}

func ExampleClient_UpdateOsConfig() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.UpdateOsConfigRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.UpdateOsConfig(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_DeleteOsConfig() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.DeleteOsConfigRequest{
		// TODO: Fill request struct fields.
	}
	err = c.DeleteOsConfig(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
}

func ExampleClient_CreatePatchPolicy() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.CreatePatchPolicyRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.CreatePatchPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_GetPatchPolicy() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.GetPatchPolicyRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.GetPatchPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_ListPatchPolicies() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.ListPatchPoliciesRequest{
		// TODO: Fill request struct fields.
	}
	it := c.ListPatchPolicies(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// TODO: Handle error.
		}
		// TODO: Use resp.
		_ = resp
	}
}

func ExampleClient_UpdatePatchPolicy() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.UpdatePatchPolicyRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.UpdatePatchPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_DeletePatchPolicy() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.DeletePatchPolicyRequest{
		// TODO: Fill request struct fields.
	}
	err = c.DeletePatchPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
}

func ExampleClient_CreateAssignment() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.CreateAssignmentRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.CreateAssignment(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_GetAssignment() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.GetAssignmentRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.GetAssignment(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_ListAssignments() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.ListAssignmentsRequest{
		// TODO: Fill request struct fields.
	}
	it := c.ListAssignments(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// TODO: Handle error.
		}
		// TODO: Use resp.
		_ = resp
	}
}

func ExampleClient_UpdateAssignment() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.UpdateAssignmentRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.UpdateAssignment(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_DeleteAssignment() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.DeleteAssignmentRequest{
		// TODO: Fill request struct fields.
	}
	err = c.DeleteAssignment(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
}

func ExampleClient_GetIamPolicy() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &iampb.GetIamPolicyRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.GetIamPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_SetIamPolicy() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &iampb.SetIamPolicyRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.SetIamPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_TestIamPermissions() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &iampb.TestIamPermissionsRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.TestIamPermissions(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

func ExampleClient_LookupConfigs() {
	ctx := context.Background()
	c, err := osconfig.NewClient(ctx)
	if err != nil {
		// TODO: Handle error.
	}

	req := &osconfigpb.LookupConfigsRequest{
		// TODO: Fill request struct fields.
	}
	resp, err := c.LookupConfigs(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	_ = resp
}

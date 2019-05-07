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

// Package osconfigserver contains wrapper around osconfig service APIs and helper methods
package osconfigserver

import (
	"context"
	"fmt"

	osconfigpb "github.com/GoogleCloudPlatform/osconfig/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/kylelemons/godebug/pretty"
)

var dump = &pretty.Config{IncludeUnexported: true}

// Assignment is a wrapper around osconfig Assignment object
type Assignment struct {
	*osconfigpb.Assignment
}

// CreateAssignment is a wrapper around createAssignment API
func CreateAssignment(ctx context.Context, assignment *osconfigpb.Assignment, parent string) (*Assignment, error) {
	client, err := GetOsConfigClient(ctx)
	if err != nil {
		return nil, err
	}

	req := &osconfigpb.CreateAssignmentRequest{
		Parent:     parent,
		Assignment: assignment,
	}

	res, err := client.CreateAssignment(ctx, req)
	if err != nil {
		return nil, err
	}
	return &Assignment{Assignment: res}, nil
}

// Cleanup function will cleanup the assignment created under project
func (a *Assignment) Cleanup(ctx context.Context, projectID string) error {
	client, err := GetOsConfigClient(ctx)
	if err != nil {
		return err
	}

	deleteReq := &osconfigpb.DeleteAssignmentRequest{
		Name: fmt.Sprintf("projects/%s/assignments/%s", projectID, a.Name),
	}
	return client.DeleteAssignment(ctx, deleteReq)
}

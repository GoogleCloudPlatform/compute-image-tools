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
	"errors"
	"fmt"
	"log"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

type Assignment struct {
	*osconfigpb.Assignment
}

// CreateAssignment is a wrapper around createAssignment API
func CreateAssignment(ctx context.Context, logger *log.Logger, assignment *Assignment, parent string) (*Assignment, error) {
	client, err := GetOsConfigClient(ctx, logger)

	if err != nil {
		return nil, err
	}

	req := &osconfigpb.CreateAssignmentRequest{
		Parent:     parent,
		Assignment: assignment.Assignment,
	}

	logger.Printf("create assignment request:\n%s\n\n", dump.Sprint(req))

	res, err := client.CreateAssignment(ctx, req)
	if err != nil {
		logger.Printf("error while creating assignment:\n%s\n\n", err)
		return nil, err
	}
	logger.Printf("create assignment response:\n%s\n", dump.Sprint(res))

	return &Assignment{Assignment: res}, nil
}

// Cleanup function will cleanup the assignment created under project
func (o *Assignment) Cleanup(ctx context.Context, logger *log.Logger) error {
	client, err := GetOsConfigClient(ctx, logger)

	if err != nil {
		return err
	}

	logger.Printf("Deleting assignment...")

	deleteReq := &osconfigpb.DeleteAssignmentRequest{
		Name: fmt.Sprintf("projects/compute-image-test-pool-001/assignments/%s", o.Name),
	}
	ok := client.DeleteAssignment(ctx, deleteReq)
	if ok != nil {
		logger.Printf("error while cleaning up")
		return errors.New(fmt.Sprintf("error while cleaning up the assignment: %s\n", ok))
	}

	logger.Printf("Assignment cleanup done.")
	return nil
}

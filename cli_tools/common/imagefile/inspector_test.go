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

package imagefile

import (
	"context"
	"errors"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	mountError      = "mount failure"
	inspectionError = "inspection failure"
)

func TestGCSInspector_RoundBytesUp(t *testing.T) {
	cases := []struct {
		name                                string
		actualSizeBytes, virtualSizeBytes   int64
		expectedActualGB, expectedVirtualGB int64
	}{
		{
			name: "zero doesn't round up",
		},

		{
			name:              "whole doesn't round up",
			actualSizeBytes:   bytesPerGB * 10,
			expectedActualGB:  10,
			virtualSizeBytes:  bytesPerGB * 8,
			expectedVirtualGB: 8,
		},

		{
			name:              "plus one rounds up",
			actualSizeBytes:   bytesPerGB + 1,
			expectedActualGB:  2,
			virtualSizeBytes:  (bytesPerGB * 2) + 1,
			expectedVirtualGB: 3,
		},

		{
			name:              "minus one rounds up",
			actualSizeBytes:   bytesPerGB - 1,
			expectedActualGB:  1,
			virtualSizeBytes:  (bytesPerGB * 2) - 1,
			expectedVirtualGB: 2,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gcsURI, client := setupClient(t, 0, 0, ImageInfo{
				Format:           "vmdk",
				ActualSizeBytes:  tt.actualSizeBytes,
				VirtualSizeBytes: tt.virtualSizeBytes,
			})
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(120*time.Millisecond))
			defer cancel()
			result, err := client.Inspect(ctx, gcsURI)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedActualGB, result.PhysicalSizeGB)
			assert.Equal(t, tt.expectedVirtualGB, result.VirtualSizeGB)
		})
	}
}

func TestGCSInspector_StopMounting_IfContextCancelled(t *testing.T) {
	gcsURI, client := setupClient(t, 1000, 0, ImageInfo{
		Format:           "vmdk",
		ActualSizeBytes:  1024,
		VirtualSizeBytes: 1024,
	})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(120*time.Millisecond))
	defer cancel()
	_, err := client.Inspect(ctx, gcsURI)
	assert.EqualError(t, err, mountError)
}

func TestGCSInspector_StopInspecting_IfContextCancelled(t *testing.T) {
	// We setup the mock client to fail 1000 times, but provide a short deadline of 120 milliseconds.
	// The implementation uses 50ms delay between failures, so only a couple of failures
	// will occur prior to the error being propagated.
	inspectFailures := 1000
	gcsURI, client := setupClient(t, 0, inspectFailures, ImageInfo{
		Format:           "vmdk",
		ActualSizeBytes:  1024,
		VirtualSizeBytes: 1024,
	})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(120*time.Millisecond))
	defer cancel()
	_, err := client.Inspect(ctx, gcsURI)
	assert.EqualError(t, err, inspectionError)
}

func TestGCSInspector_PerformRetry_WhenMountingFails(t *testing.T) {
	// Fail mounting three times, and then successfully mount.
	mountFailures := 3
	gcsURI, client := setupClient(t, mountFailures, 0, ImageInfo{
		Format:           "vmdk",
		ActualSizeBytes:  1024,
		VirtualSizeBytes: 1024,
	})
	_, err := client.Inspect(context.Background(), gcsURI)
	assert.NoError(t, err)
}

func TestGCSInspector_PerformRetry_WhenInspectionFails(t *testing.T) {
	// Fail inspection three times, and then successfully inspect.
	inspectFailures := 3
	gcsURI, client := setupClient(t, 0, inspectFailures, ImageInfo{
		Format:           "vmdk",
		ActualSizeBytes:  1024,
		VirtualSizeBytes: 1024,
	})
	_, err := client.Inspect(context.Background(), gcsURI)
	assert.NoError(t, err)
}

func setupClient(t *testing.T, mountFailures, inspectFailures int, qemuResult ImageInfo) (
	string, Inspector) {
	pathToFakeMount, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer pathToFakeMount.Close()
	assert.NoError(t, err)
	objectName := path.Base(pathToFakeMount.Name())
	fakeMountDir := path.Dir(pathToFakeMount.Name())
	gcsURI := "gs://bucket/" + objectName

	inspector := NewGCSInspector().(gcsInspector)
	inspector.fuseClient = &mockGCSFuse{
		failuresRemaining: mountFailures,
		expectedBucket:    "bucket",
		t:                 t,
		returnValue:       fakeMountDir,
	}
	inspector.qemuClient = &mockQemuClient{
		failuresRemaining: inspectFailures,
		expectedFilename:  pathToFakeMount.Name(),
		t:                 t,
		returnValue:       qemuResult,
	}
	return gcsURI, inspector
}

type mockGCSFuse struct {
	failuresRemaining int
	expectedBucket    string
	t                 *testing.T
	returnValue       string
}

func (m *mockGCSFuse) MountToTemp(ctx context.Context, bucket string) (string, error) {
	assert.Equal(m.t, m.expectedBucket, bucket)
	if m.failuresRemaining > 0 {
		m.failuresRemaining--
		err := errors.New(mountError)
		m.t.Logf("gcsfuse returning %v", err)
		return "", err
	}
	return m.returnValue, nil
}

func (m *mockGCSFuse) Unmount(directory string) error {
	return nil
}

type mockQemuClient struct {
	failuresRemaining int
	expectedFilename  string
	t                 *testing.T
	returnValue       ImageInfo
}

func (m *mockQemuClient) GetInfo(ctx context.Context, filename string) (ImageInfo, error) {
	assert.Equal(m.t, m.expectedFilename, filename)
	if m.failuresRemaining > 0 {
		m.failuresRemaining--
		err := errors.New(inspectionError)
		m.t.Logf("qemu-img returning %v", err)
		return ImageInfo{}, err
	}
	return m.returnValue, nil
}

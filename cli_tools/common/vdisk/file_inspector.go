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

package vdisk

import (
	"context"
	"fmt"
	"math"
	"path"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/gcsfuse"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/files"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"

	"github.com/cenkalti/backoff/v4"
)

const bytesPerGB = int64(1024 * 1024 * 1024)

// VirtualDiskFileMetadata contains metadata about a virtual disk file.
type VirtualDiskFileMetadata struct {
	// Gigabytes used by the file itself, rounded up to the nearest GB.
	PhysicalSizeGB int64

	// Gigabytes used by the disk, after inflation. Rounded up to the nearest GB.
	VirtualSizeGB int64

	// The file format used for encoding the VM disk.
	FileFormat Format
}

// VirtualDiskFileInspector returns metadata about virtual disk files.
type VirtualDiskFileInspector interface {
	// Inspect returns VirtualDiskFileMetadata for the virtual disk file associated
	// with a reference. IO operations will be retried until the context is cancelled.
	Inspect(ctx context.Context, reference string) (VirtualDiskFileMetadata, error)
}

// NewGCSInspector returns a virtualDiskFileInspector that inspects virtual
// disk files that are stored in the GCS bucket. The Inspect method expects
// a GCS URI to the file to be inspected.
func NewGCSInspector() VirtualDiskFileInspector {
	return gcsInspector{
		qemuClient: NewInfoClient(),
		fuseClient: gcsfuse.NewClient()}
}

// gcsInspector implements virtualDiskFileInspector using qemu-img gcsfuse.
type gcsInspector struct {
	qemuClient InfoClient
	fuseClient gcsfuse.Client
}

func (inspector gcsInspector) Inspect(ctx context.Context, gcsURI string) (metadata VirtualDiskFileMetadata, err error) {
	operation := func() error {
		metadata, err = inspector.inspectOnce(ctx, gcsURI)
		return err
	}
	return metadata, backoff.Retry(operation,
		backoff.WithContext(backoff.NewConstantBackOff(50*time.Millisecond), ctx))
}

func (inspector gcsInspector) inspectOnce(ctx context.Context, gcsURI string) (metadata VirtualDiskFileMetadata, err error) {
	bucket, object, err := storage.GetGCSObjectPathElements(gcsURI)
	if err != nil {
		return metadata, err
	}
	mountedDirectory, err := inspector.fuseClient.MountToTemp(ctx, bucket)
	defer inspector.fuseClient.Unmount(mountedDirectory)
	if err != nil {
		return metadata, err
	}
	absPath := path.Join(mountedDirectory, object)
	if !files.Exists(absPath) {
		return metadata, fmt.Errorf("the file %q was not found", gcsURI)
	}
	imageInfo, err := inspector.qemuClient.GetInfo(ctx, absPath)
	if err != nil {
		return metadata, err
	}
	return VirtualDiskFileMetadata{
		PhysicalSizeGB: bytesToGB(imageInfo.ActualSizeBytes),
		VirtualSizeGB:  bytesToGB(imageInfo.VirtualSizeBytes),
		FileFormat:     imageInfo.Format,
	}, nil
}

// bytesToGB rounds up to the nearest GB.
func bytesToGB(bytes int64) int64 {
	return int64(math.Ceil(float64(bytes) / float64(bytesPerGB)))
}

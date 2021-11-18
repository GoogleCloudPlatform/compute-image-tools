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

package gcsfuse

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
)

// Client provides methods for mounting and unmounting FUSE filesystems
// that are backed by GCS.
type Client interface {
	// MountToTemp mounts a bucket within within the operating system's temporary directory,
	// and returns an absolute path to the newly-created directory. IO operations will retry
	// until the context is cancelled.
	MountToTemp(ctx context.Context, bucket string) (string, error)
	Unmount(directory string) error
}

// NewClient creates a new gcsfuse.Client.
func NewClient() Client {
	return defaultClient{}
}

type defaultClient struct{}

func (client defaultClient) MountToTemp(ctx context.Context, bucket string) (string, error) {
	dir, err := ioutil.TempDir("", bucket)
	if err != nil {
		return "", fmt.Errorf("failed to create a destination directory: %w", err)
	}
	cmd := exec.CommandContext(ctx, "gcsfuse", "--implicit-dirs", bucket, dir)
	_, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (client defaultClient) Unmount(directory string) error {
	cmd := exec.Command("umount", directory)
	_, err := cmd.Output()
	return err
}

//  Copyright 2021 Google Inc. All Rights Reserved.
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

package fileinspect

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/stretchr/testify/assert"
)

func TestInspectDisk(t *testing.T) {
	deadline, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))
	defer cancelFunc()

	inspector := imagefile.NewGCSInspector()
	metaData, err := inspector.Inspect(deadline, "gs://compute-image-tools-test-resources/file-inflation-test/virt-8G.vmdk")
	assert.NoError(t, err)
	assert.Equal(t, metaData.Checksum, "dffcdd4e62005b1d5558d4c11fb85073-d41d8cd98f00b204e9800998ecf8427e-d41d8cd98f00b204e9800998ecf8427e-dffcdd4e62005b1d5558d4c11fb85073")
}

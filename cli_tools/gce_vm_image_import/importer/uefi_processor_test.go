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

package importer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUefiProcessor(t *testing.T) {
	for i, tt := range uefiTests {
		name := fmt.Sprintf("%v. inspect disk: disk is UEFI: %v, input arg UEFI compatible: %v", i+1, tt.isUEFIDisk, tt.isInputArgUEFICompatible)
		t.Run(name, func(t *testing.T) {
			args := ImportArguments{
				UefiCompatible: tt.isInputArgUEFICompatible,
			}
			p := newDiskMutationProcessor(mockComputeDiskClient{}, args)
			pd, err := p.process(persistentDisk{uri: "old-uri", isUEFIDetected: tt.isUEFIDisk || tt.isInputArgUEFICompatible})
			assert.NoError(t, err)
			if tt.isUEFIDisk && !tt.isInputArgUEFICompatible {
				assert.Truef(t, strings.HasSuffix(pd.uri, "uefi"), "UEFI Disk URI should have suffix 'uefi', actual: %v", pd.uri)
			} else {
				assert.Falsef(t, strings.HasSuffix(pd.uri, "uefi"), "Disk URI shouldn't have suffix 'uefi', actual: %v", pd.uri)
			}
		})
	}
}

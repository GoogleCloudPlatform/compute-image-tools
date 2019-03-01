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

package ospackage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

func runZypperRepositories(repos []*osconfigpb.ZypperRepository) (string, error) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	testRepo := filepath.Join(td, "testRepo")

	if err := zypperRepositories(repos, testRepo); err != nil {
		return "", fmt.Errorf("error running zypperRepositories: %v", err)
	}

	data, err := ioutil.ReadFile(testRepo)
	if err != nil {
		return "", fmt.Errorf("error reading testRepo: %v", err)
	}

	return string(data), nil
}

func TestZypperRepositories(t *testing.T) {
	tests := []struct {
		desc  string
		repos []*osconfigpb.ZypperRepository
		want  string
	}{
		{"no repos", []*osconfigpb.ZypperRepository{}, "# Repo file managed by Google OSConfig agent\n"},
		{
			"1 repo",
			[]*osconfigpb.ZypperRepository{
				{BaseUrl: "http://repo1-url/", Id: "id"},
			},
			"# Repo file managed by Google OSConfig agent\n\n[id]\nname: id\nbaseurl: http://repo1-url/\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\n",
		},
		{
			"2 repos",
			[]*osconfigpb.ZypperRepository{
				{BaseUrl: "http://repo1-url/", Id: "id1", DisplayName: "displayName1", GpgKeys: []string{"https://url/key"}},
				{BaseUrl: "http://repo1-url/", Id: "id2", DisplayName: "displayName2", GpgKeys: []string{"https://url/key1", "https://url/key2"}},
			},
			"# Repo file managed by Google OSConfig agent\n\n[id1]\nname: displayName1\nbaseurl: http://repo1-url/\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\ngpgkey=https://url/key\n\n[id2]\nname: displayName2\nbaseurl: http://repo1-url/\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\ngpgkey=https://url/key1\n       https://url/key2\n",
		},
	}

	for _, tt := range tests {
		got, err := runZypperRepositories(tt.repos)
		if err != nil {
			t.Fatal(err)
		}

		if got != tt.want {
			t.Errorf("%s: got:\n%q\nwant:\n%q", tt.desc, got, tt.want)
		}
	}
}

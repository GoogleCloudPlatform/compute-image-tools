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

// +build !public

package ospackage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

func runGooGetRepositories(repos []*osconfigpb.GooRepository) (string, error) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	testRepo := filepath.Join(td, "testRepo")

	if err := googetRepositories(repos, testRepo); err != nil {
		return "", fmt.Errorf("error running googetRepositories: %v", err)
	}

	data, err := ioutil.ReadFile(testRepo)
	if err != nil {
		return "", fmt.Errorf("error reading testRepo: %v", err)
	}

	return string(data), nil
}

func TestGooGetRepositories(t *testing.T) {
	tests := []struct {
		desc  string
		repos []*osconfigpb.GooRepository
		want  string
	}{
		{"no repos", []*osconfigpb.GooRepository{}, "# Repo file managed by Google OSConfig agent\n"},
		{
			"1 repo",
			[]*osconfigpb.GooRepository{
				{Url: "http://repo1-url/", Name: "name"},
			},
			"# Repo file managed by Google OSConfig agent\n\n- name: name\n  url: http://repo1-url/\n",
		},
		{
			"2 repos",
			[]*osconfigpb.GooRepository{
				{Url: "http://repo1-url/", Name: "name1"},
				{Url: "http://repo2-url/", Name: "name2"},
			},
			"# Repo file managed by Google OSConfig agent\n\n- name: name1\n  url: http://repo1-url/\n\n- name: name2\n  url: http://repo2-url/\n",
		},
	}

	for _, tt := range tests {
		got, err := runGooGetRepositories(tt.repos)
		if err != nil {
			t.Fatal(err)
		}

		if got != tt.want {
			t.Errorf("%s: got:\n%q\nwant:\n%q", tt.desc, got, tt.want)
		}
	}
}

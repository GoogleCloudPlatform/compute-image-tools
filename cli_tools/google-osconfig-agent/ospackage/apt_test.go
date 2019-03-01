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

func runAptRepositories(repos []*osconfigpb.AptRepository) (string, error) {
	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	testRepo := filepath.Join(td, "testRepo")

	if err := aptRepositories(repos, testRepo); err != nil {
		return "", fmt.Errorf("error running aptRepositories: %v", err)
	}

	data, err := ioutil.ReadFile(testRepo)
	if err != nil {
		return "", fmt.Errorf("error reading testRepo: %v", err)
	}

	return string(data), nil
}

func TestAptRepositories(t *testing.T) {
	tests := []struct {
		desc  string
		repos []*osconfigpb.AptRepository
		want  string
	}{
		{"no repos", []*osconfigpb.AptRepository{}, "# Repo file managed by Google OSConfig agent\n"},
		{
			"1 repo",
			[]*osconfigpb.AptRepository{
				{Uri: "http://repo1-url/", Distribution: "distribution", Components: []string{"component1"}},
			},
			"# Repo file managed by Google OSConfig agent\n\ndeb http://repo1-url/ distribution component1\n",
		},
		{
			"2 repos",
			[]*osconfigpb.AptRepository{
				{Uri: "http://repo1-url/", Distribution: "distribution", Components: []string{"component1"}, ArchiveType: osconfigpb.AptRepository_DEB_SRC},
				{Uri: "http://repo2-url/", Distribution: "distribution", Components: []string{"component1", "component2"}, ArchiveType: osconfigpb.AptRepository_DEB},
			},
			"# Repo file managed by Google OSConfig agent\n\ndeb-src http://repo1-url/ distribution component1\n\ndeb http://repo2-url/ distribution component1 component2\n",
		},
	}

	for _, tt := range tests {
		got, err := runAptRepositories(tt.repos)
		if err != nil {
			t.Fatal(err)
		}

		if got != tt.want {
			t.Errorf("%s: got:\n%q\nwant:\n%q", tt.desc, got, tt.want)
		}
	}
}

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
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
)

func TestGetNecessaryChanges(t *testing.T) {
	tests := [...]struct {
		name            string
		installedPkgs   []packages.PkgInfo
		upgradablePkgs  []packages.PkgInfo
		packageInstalls []*osconfigpb.Package
		packageRemovals []*osconfigpb.Package
		want            changes
	}{
		{
			name:            "install from empty",
			installedPkgs:   createPkgInfos(),
			upgradablePkgs:  createPkgInfos(),
			packageInstalls: createPackages("foo"),
			packageRemovals: createPackages(),
			want: changes{
				packagesToInstall: []string{"foo"},
				packagesToUpgrade: []string{},
				packagesToRemove:  []string{},
			},
		}, {
			name:            "single upgrade",
			installedPkgs:   createPkgInfos("foo"),
			upgradablePkgs:  createPkgInfos("foo"),
			packageInstalls: createPackages("foo"),
			packageRemovals: createPackages(),
			want: changes{
				packagesToInstall: []string{},
				packagesToUpgrade: []string{"foo"},
				packagesToRemove:  []string{},
			},
		}, {
			name:            "remove",
			installedPkgs:   createPkgInfos("foo"),
			upgradablePkgs:  createPkgInfos("foo"),
			packageInstalls: createPackages(),
			packageRemovals: createPackages("foo"),
			want: changes{
				packagesToInstall: []string{},
				packagesToUpgrade: []string{},
				packagesToRemove:  []string{"foo"},
			},
		}, {
			name:            "mixed",
			installedPkgs:   createPkgInfos("foo", "bar", "buz"),
			upgradablePkgs:  createPkgInfos("bar"),
			packageInstalls: createPackages("foo", "baz"),
			packageRemovals: createPackages("buz"),
			want: changes{
				packagesToInstall: []string{"baz"},
				packagesToUpgrade: []string{},
				packagesToRemove:  []string{"buz"},
			},
		},
	}

	for _, tt := range tests {
		got := getNecessaryChanges(tt.installedPkgs, tt.upgradablePkgs, tt.packageInstalls, tt.packageRemovals)

		if !equalChanges(&got, &tt.want) {
			t.Errorf("Did not get expected changes for '%s', got: %v, want: %v", tt.name, got, tt.want)
		}
	}
}

func equalChanges(got *changes, want *changes) bool {
	return equalSlices(got.packagesToInstall, want.packagesToInstall) &&
		equalSlices(got.packagesToRemove, want.packagesToRemove) &&
		equalSlices(got.packagesToUpgrade, want.packagesToUpgrade)
}

func equalSlices(got []string, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}

func createPkgInfos(names ...string) []packages.PkgInfo {
	var res []packages.PkgInfo
	for _, n := range names {
		res = append(res, packages.PkgInfo{Name: n})
	}
	return res
}

func createPackages(names ...string) []*osconfigpb.Package {
	var res []*osconfigpb.Package
	for _, n := range names {
		res = append(res, &osconfigpb.Package{Name: n})
	}
	return res
}

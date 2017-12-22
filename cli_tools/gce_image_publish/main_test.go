//  Copyright 2017 Google Inc. All Rights Reserved.
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

package main

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	compute "google.golang.org/api/compute/v1"
)

func TestCreatePrintOut(t *testing.T) {
	tests := []struct {
		name string
		args *daisy.CreateImages
		want []string
	}{
		{"empty", nil, nil},
		{"one image", &daisy.CreateImages{&daisy.CreateImage{Image: compute.Image{Name: "foo", Description: "bar"}}}, []string{"foo: (bar)"}},
		{"two images", &daisy.CreateImages{
			&daisy.CreateImage{Image: compute.Image{Name: "foo1", Description: "bar1"}},
			&daisy.CreateImage{Image: compute.Image{Name: "foo2", Description: "bar2"}}},
			[]string{"foo1: (bar1)", "foo2: (bar2)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToCreate := createPrintOut(tt.args)
			if !reflect.DeepEqual(gotToCreate, tt.want) {
				t.Errorf("createPrintOut() got = %v, want %v", gotToCreate, tt.want)
			}
		})
	}
}

func TestDeletePrintOut(t *testing.T) {
	tests := []struct {
		name string
		args *daisy.DeleteResources
		want []string
	}{
		{"empty", nil, nil},
		{"not an image", &daisy.DeleteResources{Disks: []string{"foo"}}, nil},
		{"one image", &daisy.DeleteResources{Images: []string{"foo"}}, []string{"foo"}},
		{"two images", &daisy.DeleteResources{Images: []string{"foo", "bar"}}, []string{"foo", "bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToDelete := deletePrintOut(tt.args)
			if !reflect.DeepEqual(gotToDelete, tt.want) {
				t.Errorf("deletePrintOut() got = %v, want %v", gotToDelete, tt.want)
			}
		})
	}
}

func TestDeprecatePrintOut(t *testing.T) {
	tests := []struct {
		name          string
		args          *daisy.DeprecateImages
		toDeprecate   []string
		toObsolete    []string
		toUndeprecate []string
	}{
		{"empty", nil, nil, nil, nil},
		{"unknown state", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "foo"}}}, nil, nil, nil},
		{"only DEPRECATED", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}}}, []string{"foo"}, nil, nil},
		{"only OBSOLETE", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "OBSOLETE"}}}, nil, []string{"foo"}, nil},
		{"only un-deprecated", &daisy.DeprecateImages{&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: ""}}}, nil, nil, []string{"foo"}},
		{"all three", &daisy.DeprecateImages{
			&daisy.DeprecateImage{Image: "foo", DeprecationStatus: compute.DeprecationStatus{State: "DEPRECATED"}},
			&daisy.DeprecateImage{Image: "bar", DeprecationStatus: compute.DeprecationStatus{State: "OBSOLETE"}},
			&daisy.DeprecateImage{Image: "baz", DeprecationStatus: compute.DeprecationStatus{State: ""}}},
			[]string{"foo"}, []string{"bar"}, []string{"baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToDeprecate, gotToObsolete, gotToUndeprecate := deprecatePrintOut(tt.args)
			if !reflect.DeepEqual(gotToDeprecate, tt.toDeprecate) {
				t.Errorf("deprecatePrintOut() got = %v, want %v", gotToDeprecate, tt.toDeprecate)
			}
			if !reflect.DeepEqual(gotToObsolete, tt.toObsolete) {
				t.Errorf("deprecatePrintOut() got1 = %v, want %v", gotToObsolete, tt.toObsolete)
			}
			if !reflect.DeepEqual(gotToUndeprecate, tt.toUndeprecate) {
				t.Errorf("deprecatePrintOut() got2 = %v, want %v", gotToUndeprecate, tt.toUndeprecate)
			}
		})
	}
}

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

package testconfig

import (
	"reflect"
	"testing"
)

func TestGetZone(t *testing.T) {
	in := map[string]int{"zone1": 10, "zone2": 20, "zone3": 30}
	p := GetProject("projectID", in)

	got := map[string]int{"zone1": 0, "zone2": 0, "zone3": 0}
	for i := 0; i < 60; i++ {
		zone := p.GetZone()
		for z := range got {
			if z == zone {
				got[z]++
			}
		}
	}

	want := map[string]int{"zone1": 10, "zone2": 20, "zone3": 30}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("GetZone output:\nwant: %+v\ngot: %+v", want, got)
	}

	want = map[string]int{"zone1": 0, "zone2": 0, "zone3": 0}
	if !reflect.DeepEqual(want, in) {
		t.Errorf("testZones end state:\nwant: %+v\ngot: %+v", want, in)
	}
	if len(p.zoneIndices) != 0 {
		t.Errorf("len(p.zoneIndices) != 0: %d", len(p.zoneIndices))
	}

	// Return all the zones.
	for z, n := range got {
		for i := 0; i < n; i++ {
			p.ReturnZone(z)
		}
	}
	want = map[string]int{"zone1": 10, "zone2": 20, "zone3": 30}
	if !reflect.DeepEqual(want, in) {
		t.Errorf("GetZone output:\nwant: %+v\ngot: %+v", want, in)
	}
	if len(p.zoneIndices) != 3 {
		t.Errorf("len(p.zoneIndices) != 3: %d", len(p.zoneIndices))
	}
}

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

package inventory

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

func decodePackages(str string) packages.Packages {
	decoded, _ := base64.StdEncoding.DecodeString(str)
	zr, _ := gzip.NewReader(bytes.NewReader(decoded))
	var buf bytes.Buffer
	io.Copy(&buf, zr)
	zr.Close()

	var pkgs packages.Packages
	json.Unmarshal(buf.Bytes(), &pkgs)
	return pkgs
}

func TestWriteInventory(t *testing.T) {
	inv := &instanceInventory{
		Hostname:      "Hostname",
		LongName:      "LongName",
		ShortName:     "ShortName",
		Architecture:  "Architecture",
		KernelVersion: "KernelVersion",
		Version:       "Version",
		InstalledPackages: packages.Packages{
			Yum: []packages.PkgInfo{{Name: "Name", Arch: "Arch", Version: "Version"}},
			WUA: []packages.WUAPackage{{Title: "Title"}},
			QFE: []packages.QFEPackage{{HotFixID: "HotFixID"}},
		},
		PackageUpdates: packages.Packages{
			Apt: []packages.PkgInfo{{Name: "Name", Arch: "Arch", Version: "Version"}},
		},
	}

	want := map[string]bool{
		"Hostname":          false,
		"LongName":          false,
		"ShortName":         false,
		"Architecture":      false,
		"KernelVersion":     false,
		"Version":           false,
		"InstalledPackages": false,
		"PackageUpdates":    false,
	}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r.Body); err != nil {
			t.Fatal(err)
		}

		switch url {
		case "/Hostname":
			if buf.String() != inv.Hostname {
				t.Errorf("did not get expected Hostname, got: %q, want: %q", buf.String(), inv.Hostname)
			}
			want["Hostname"] = true
		case "/LongName":
			if buf.String() != inv.LongName {
				t.Errorf("did not get expected LongName, got: %q, want: %q", buf.String(), inv.LongName)
			}
			want["LongName"] = true
		case "/ShortName":
			if buf.String() != inv.ShortName {
				t.Errorf("did not get expected ShortName, got: %q, want: %q", buf.String(), inv.ShortName)
			}
			want["ShortName"] = true
		case "/Architecture":
			if buf.String() != inv.Architecture {
				t.Errorf("did not get expected Architecture, got: %q, want: %q", buf.String(), inv.Architecture)
			}
			want["Architecture"] = true
		case "/KernelVersion":
			if buf.String() != inv.KernelVersion {
				t.Errorf("did not get expected KernelVersion, got: %q, want: %q", buf.String(), inv.KernelVersion)
			}
			want["KernelVersion"] = true
		case "/Version":
			if buf.String() != inv.Version {
				t.Errorf("did not get expected Version, got: %q, want: %q", buf.String(), inv.Version)
			}
			want["Version"] = true
		case "/InstalledPackages":
			got := decodePackages(buf.String())
			if !reflect.DeepEqual(got, inv.InstalledPackages) {
				t.Errorf("did not get expected InstalledPackages, got: %q, want: %q", got, inv.InstalledPackages)
			}
			want["InstalledPackages"] = true
		case "/PackageUpdates":
			got := decodePackages(buf.String())
			if !reflect.DeepEqual(got, inv.PackageUpdates) {
				t.Errorf("did not get expected PackageUpdates, got: %q, want: %q", got, inv.PackageUpdates)
			}
			want["PackageUpdates"] = true
		default:
			w.WriteHeader(500)
			fmt.Fprintln(w, "URL and Method not recognized:", r.Method, url)
		}
	}))
	defer svr.Close()

	writeInventory(inv, svr.URL)

	for k, v := range want {
		if v {
			continue
		}
		t.Errorf("writeInventory call did not write %q", k)
	}
}

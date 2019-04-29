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

package daisy

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/davecgh/go-spew/spew"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	godebugDiff "github.com/kylelemons/godebug/diff"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/grpc"
)

const DNE = "DNE!"

type mockStep struct {
	populateImpl func(context.Context, *Step) dErr
	runImpl      func(context.Context, *Step) dErr
	validateImpl func(context.Context, *Step) dErr
}

func (m *mockStep) populate(ctx context.Context, s *Step) dErr {
	if m.populateImpl != nil {
		return m.populateImpl(ctx, s)
	}
	return nil
}

func (m *mockStep) run(ctx context.Context, s *Step) dErr {
	if m.runImpl != nil {
		return m.runImpl(ctx, s)
	}
	return nil
}

func (m *mockStep) validate(ctx context.Context, s *Step) dErr {
	if m.validateImpl != nil {
		return m.validateImpl(ctx, s)
	}
	return nil
}

var (
	testWf             = "test-wf"
	testProject        = "test-project"
	testZone           = "test-zone"
	testRegion         = "test-zo"
	testDisk           = "test-disk"
	testForwardingRule = "test-forwarding-rule"
	testFirewallRule   = "test-firewall-rule"
	testImage          = "test-image"
	testInstance       = "test-instance"
	testMachineType    = "test-machine-type"
	testLicense        = "test-license"
	testNetwork        = "test-network"
	testSubnetwork     = "test-subnetwork"
	testTargetInstance = "test-target-instance"
	testFamily         = "test-family"
	testGCSPath        = "gs://test-bucket"
	testGCSObjs        []string
	testGCSObjsMx      = sync.Mutex{}

	spewCfg = spew.ConfigState{
		Indent:                  "\t",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
		SortKeys:                true,
		SpewKeys:                true,
	}
)

func testWorkflow() *Workflow {
	w := New()
	w.id = "abcdef"
	w.Name = testWf
	w.GCSPath = testGCSPath
	w.Project = testProject
	w.Zone = testZone
	w.ComputeClient, _ = newTestGCEClient()
	w.StorageClient, _ = newTestGCSClient()
	w.cloudLoggingClient, _ = newTestLoggingClient()
	w.Cancel = make(chan struct{})
	w.Logger = &MockLogger{}
	return w
}

func addGCSObj(o string) {
	testGCSObjsMx.Lock()
	defer testGCSObjsMx.Unlock()
	testGCSObjs = append(testGCSObjs, o)
}

func diff(x, y interface{}, depth int) string {
	cfg := spewCfg
	cfg.MaxDepth = depth
	return godebugDiff.Diff(cfg.Sdump(x), cfg.Sdump(y))
}

func newTestGCEClient() (*daisyCompute.TestClient, error) {
	_, c, err := daisyCompute.NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=1") {
			fmt.Fprintln(w, `{"Contents":"failsuccess","Start":"0"}`)
		} else if r.Method == "GET" && strings.Contains(r.URL.String(), "serialPort?alt=json&port=2") {
			fmt.Fprintln(w, `{"Contents":"successfail","Start":"0"}`)
		} else {
			fmt.Fprintln(w, `{"Status":"DONE","SelfLink":"link"}`)
		}
	}))

	c.GetProjectFn = func(project string) (*compute.Project, error) {
		if project == DNE {
			return nil, &googleapi.Error{Code: http.StatusNotFound}
		}
		if project == testProject {
			return nil, nil
		}
		return nil, errors.New("bad project")
	}
	c.GetZoneFn = func(_, zone string) (*compute.Zone, error) {
		if zone == DNE {
			return nil, &googleapi.Error{Code: http.StatusNotFound}
		}
		if zone == testZone {
			return nil, nil
		}
		return nil, errors.New("bad zone")
	}
	c.GetMachineTypeFn = func(_, _, mt string) (*compute.MachineType, error) {
		if mt == testMachineType {
			return nil, nil
		}
		return nil, errors.New("bad machinetype")
	}
	c.ListMachineTypesFn = func(p, z string, _ ...daisyCompute.ListCallOption) ([]*compute.MachineType, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		if z != testZone {
			return nil, errors.New("bad zone: " + z)
		}
		return []*compute.MachineType{{Name: testMachineType}}, nil
	}
	c.ListZonesFn = func(_ string, _ ...daisyCompute.ListCallOption) ([]*compute.Zone, error) {
		return []*compute.Zone{{Name: testZone}}, nil
	}
	c.ListFirewallRulesFn = func(p string, _ ...daisyCompute.ListCallOption) ([]*compute.Firewall, error) {
		if p == testProject {
			return []*compute.Firewall{{Name: testFirewallRule}}, nil
		}
		return []*compute.Firewall{{Name: testFirewallRule}}, nil
	}
	c.ListImagesFn = func(p string, _ ...daisyCompute.ListCallOption) ([]*computeAlpha.Image, error) {
		if p == testProject {
			return []*computeAlpha.Image{{Name: testImage}}, nil
		}
		return []*computeAlpha.Image{{Name: testImage, Deprecated: &computeAlpha.DeprecationStatus{State: "OBSOLETE"}}}, nil
	}
	c.ListDisksFn = func(p, z string, _ ...daisyCompute.ListCallOption) ([]*compute.Disk, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		if z != testZone {
			return nil, errors.New("bad zone: " + z)
		}
		return []*compute.Disk{{Name: testDisk}}, nil
	}
	c.ListForwardingRulesFn = func(p, r string, _ ...daisyCompute.ListCallOption) ([]*compute.ForwardingRule, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		if r != testRegion {
			return nil, errors.New("bad region: " + r)
		}
		return []*compute.ForwardingRule{{Name: testForwardingRule}}, nil
	}
	c.GetLicenseFn = func(p, l string) (*compute.License, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		if l != testLicense {
			return nil, errors.New("bad license: " + l)
		}
		return nil, nil
	}
	c.ListNetworksFn = func(p string, _ ...daisyCompute.ListCallOption) ([]*compute.Network, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		return []*compute.Network{{Name: testNetwork}}, nil
	}
	c.ListSubnetworksFn = func(p, r string, _ ...daisyCompute.ListCallOption) ([]*compute.Subnetwork, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		if r != testRegion {
			return nil, errors.New("bad region: " + r)
		}
		return []*compute.Subnetwork{{Name: testSubnetwork}}, nil
	}
	c.ListTargetInstancesFn = func(p, z string, _ ...daisyCompute.ListCallOption) ([]*compute.TargetInstance, error) {
		if p != testProject {
			return nil, errors.New("bad project: " + p)
		}
		if z != testZone {
			return nil, errors.New("bad zone: " + z)
		}
		return []*compute.TargetInstance{{Name: testTargetInstance}}, nil
	}
	c.CreateFirewallRuleFn = func(p string, i *compute.Firewall) error {
		if p != testProject {
			return errors.New("bad project: " + p)
		}
		if i.Name != testFirewallRule {
			return errors.New("bad FirewallRule name: " + i.Name)
		}
		return nil
	}
	c.CreateImageFn = func(p string, i *computeAlpha.Image) error {
		if p != testProject {
			return errors.New("bad project: " + p)
		}
		if i.Name != testImage {
			return errors.New("bad image name: " + i.Name)
		}
		if i.SourceDisk != "" && i.SourceDisk != testDisk {
			return errors.New("bad source disk: " + i.SourceDisk)
		}
		return nil
	}
	c.GetImageFromFamilyFn = func(_, f string) (*computeAlpha.Image, error) {
		if f == testFamily {
			return &computeAlpha.Image{Name: testImage}, nil
		}
		return &computeAlpha.Image{Name: testImage, Deprecated: &computeAlpha.DeprecationStatus{State: "OBSOLETE"}}, nil
	}
	c.AttachDiskFn = func(p, z, i string, _ *compute.AttachedDisk) error {
		if i == "bad" {
			return errors.New("bad instance")
		}
		return nil
	}
	c.DetachDiskFn = func(p, z, i, _ string) error {
		if i == "bad" {
			return errors.New("bad instance")
		}
		return nil
	}

	return c, err
}

func newTestGCSClient() (*storage.Client, error) {
	nameRgx := regexp.MustCompile(`"name":"([^"].*)"`)
	rewriteRgx := regexp.MustCompile(`/b/([^/]+)/o/([^/]+)/rewriteTo/b/([^/]+)/o/([^?]+)`)
	uploadRgx := regexp.MustCompile(`/b/([^/]+)/o?.*uploadType=multipart.*`)
	getObjRgx := regexp.MustCompile(`/b/.+/o/.+alt=json&projection=full`)
	getBktRgx := regexp.MustCompile(`/b/.+alt=json&prettyPrint=false&projection=full`)
	deleteObjRgx := regexp.MustCompile(`/b/.+/o/.+alt=json`)
	listObjsRgx := regexp.MustCompile(`/b/.+/o\?alt=json&delimiter=&pageToken=&prefix=.+&projection=full&versions=false`)
	listObjsNoPrefixRgx := regexp.MustCompile(`/b/.+/o\?alt=json&delimiter=&pageToken=&prefix=&projection=full&versions=false`)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		u := r.URL.String()
		m := r.Method

		if match := uploadRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			body, _ := ioutil.ReadAll(r.Body)
			n := nameRgx.FindStringSubmatch(string(body))[1]
			addGCSObj(n)
			fmt.Fprintf(w, `{"kind":"storage#object","bucket":"%s","name":"%s"}`, match[1], n)
		} else if match := rewriteRgx.FindStringSubmatch(u); m == "POST" && match != nil {
			if strings.Contains(match[1], "dne") || strings.Contains(match[2], "dne") {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, storage.ErrObjectNotExist)
				return
			}
			path, err := url.PathUnescape(match[4])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, err)
				return
			}
			addGCSObj(path)
			o := fmt.Sprintf(`{"bucket":"%s","name":"%s"}`, match[3], match[4])
			fmt.Fprintf(w, `{"kind": "storage#rewriteResponse", "done": true, "objectSize": "1", "totalBytesRewritten": "1", "resource": %s}`, o)
		} else if match := getObjRgx.FindStringSubmatch(u); m == "GET" && match != nil {
			// Return StatusNotFound for objects that do not exist.
			if strings.Contains(match[0], "dne") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Yes this object exists, we don't need to fill out the values, just return something.
			fmt.Fprint(w, "{}")
		} else if match := getBktRgx.FindStringSubmatch(u); m == "GET" && match != nil {
			// Return StatusNotFound for objects that do not exist.
			if strings.Contains(match[0], "dne") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Yes this object exists, we don't need to fill out the values, just return something.
			fmt.Fprint(w, "{}")
		} else if match := deleteObjRgx.FindStringSubmatch(u); m == "DELETE" && match != nil {
			// Return StatusNotFound for objects that do not exist.
			if strings.Contains(match[0], "dne") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Yes this object exists, we don't need to fill out the values, just return something.
			fmt.Fprint(w, "{}")
		} else if match := listObjsRgx.FindStringSubmatch(u); m == "GET" && match != nil {
			// Return StatusNotFound for objects that do not exist.
			if strings.Contains(match[0], "dne") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Return 2 objects for testing recursiveGCS.
			fmt.Fprint(w, `{"kind": "storage#objects", "items": [{"kind": "storage#object", "name": "folder/object", "size": "1"},{"kind": "storage#object", "name": "folder/folder/object", "size": "1"}]}`)
		} else if match := listObjsNoPrefixRgx.FindStringSubmatch(u); m == "GET" && match != nil {
			// Return 2 objects for testing recursiveGCS.
			fmt.Fprint(w, `{"kind": "storage#objects", "items": [{"kind": "storage#object", "name": "object", "size": "1"},{"kind": "storage#object", "name": "folder/object", "size": "1"}]}`)
		} else if m == "PUT" && u == "/b/bucket/o/object/acl/allUsers?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "PUT" && u == "/b/bucket/o/object%2Ffolder%2Ffolder%2Fobject/acl/allUsers?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "PUT" && u == "/b/bucket/o/object%2Ffolder%2Fobject/acl/allUsers?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "GET" && u == "/b?alt=json&pageToken=&prefix=&prettyPrint=false&project=foo-project&projection=full" {
			fmt.Fprint(w, `{}`)
		} else if m == "GET" && u == "/b?alt=json&pageToken=&prefix=&prettyPrint=false&project=bar-project&projection=full" {
			fmt.Fprint(w, `{"items": [{"name": "bar-project-daisy-bkt"}]}`)
		} else if m == "POST" && u == "/b?alt=json&prettyPrint=false&project=foo-project" {
			fmt.Fprint(w, `{}`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "testGCSClient unknown request: %+v\n", r)
		}
	}))

	return storage.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
}

func newTestLoggingClient() (*logging.Client, error) {
	addr, err := newFakeLoggingServer()
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c, err := logging.NewClient(context.Background(), "test-project", option.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func newFakeLoggingServer() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	addr := l.Addr().String()
	gsrv := grpc.NewServer()

	logpb.RegisterLoggingServiceV2Server(gsrv, &loggingHandler{})
	go gsrv.Serve(l)
	return addr, nil
}

type loggingHandler struct {
	logpb.LoggingServiceV2Server
}

func (h *loggingHandler) WriteLogEntries(_ context.Context, _ *logpb.WriteLogEntriesRequest) (*logpb.WriteLogEntriesResponse, error) {
	return &logpb.WriteLogEntriesResponse{}, nil
}

func (h *loggingHandler) DeleteLog(_ context.Context, _ *logpb.DeleteLogRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (h *loggingHandler) ListLogEntries(_ context.Context, _ *logpb.ListLogEntriesRequest) (*logpb.ListLogEntriesResponse, error) {
	return &logpb.ListLogEntriesResponse{
		Entries:       nil,
		NextPageToken: "",
	}, nil
}

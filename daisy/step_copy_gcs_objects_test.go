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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func TestCopyGCSObjectsPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	ws := &CopyGCSObjects{
		{Source: "gs://bucket/object", Destination: "gs://bucket/object", ACLRules: []*storage.ACLRule{{Entity: "allUsers", Role: "OWNER"}}},
		{Source: "gs://bucket/object", Destination: "gs://bucket/object", ACLRules: []*storage.ACLRule{{Entity: "allAuthenticatedUsers", Role: "writer"}}},
	}
	if err := ws.populate(ctx, s); err != nil {
		t.Errorf("error running CopyGCSObjects.populate(): %v", err)
	}
	want := &CopyGCSObjects{
		{Source: "gs://bucket/object", Destination: "gs://bucket/object", ACLRules: []*storage.ACLRule{{Entity: "allUsers", Role: "OWNER"}}},
		{Source: "gs://bucket/object", Destination: "gs://bucket/object", ACLRules: []*storage.ACLRule{{Entity: "allAuthenticatedUsers", Role: "WRITER"}}},
	}
	if diffRes := diff(ws, want, 0); diffRes != "" {
		t.Errorf("populated CopyGCSObjects does not match expectation: (-got +want)\n%s", diffRes)
	}
}

func TestCopyGCSObjectsValidate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.String()
		m := r.Method

		if m == "GET" && u == "/b/bucket1?alt=json&prettyPrint=false&projection=full" {
			fmt.Fprint(w, `{}`)
		} else if m == "GET" && u == "/b/bucket2?alt=json&prettyPrint=false&projection=full" {
			fmt.Fprint(w, `{}`)
		} else if m == "GET" && u == "/b/bucket/o?alt=json&delimiter=&endOffset=&pageToken=&prefix=&prettyPrint=false&projection=full&startOffset=&versions=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "POST" && (u == "/b/bucket1/o?alt=json&prettyPrint=false&projection=full&uploadType=multipart" ||
			u == "/upload/storage/v1/b/bucket1/o?alt=json&name=daisy-validate--abcdef&prettyPrint=false&projection=full&uploadType=multipart") {
			fmt.Fprint(w, `{}`)
		} else if m == "POST" && (u == "/b/bucket2/o?alt=json&prettyPrint=false&projection=full&uploadType=multipart" ||
			u == "/upload/storage/v1/b/bucket2/o?alt=json&name=daisy-validate--abcdef&prettyPrint=false&projection=full&uploadType=multipart") {
			fmt.Fprint(w, `{}`)
		} else if m == "DELETE" && u == "/b/bucket1/o/abcdef?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "DELETE" && u == "/b/bucket1/o/daisy-validate--abcdef?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "DELETE" && u == "/b/bucket2/o/daisy-validate--abcdef?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "PUT" && u == "/b/bucket1/o/daisy-validate--abcdef/acl/allUsers?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else if m == "PUT" && u == "/b/bucket2/o/daisy-validate--abcdef/acl/allUsers?alt=json&prettyPrint=false" {
			fmt.Fprint(w, `{}`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "testGCSClient copy unknown request: %+v\n", r)
		}
	}))
	sc, err := storage.NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	w := testWorkflow()
	w.StorageClient = sc
	s := &Step{w: w}

	ws := &CopyGCSObjects{
		{Source: "gs://bucket1", Destination: "gs://bucket1"},
		{Source: "gs://bucket1", Destination: "gs://bucket2", ACLRules: []*storage.ACLRule{{Entity: "allUsers", Role: "OWNER"}}},
	}
	if err := ws.validate(ctx, s); err != nil {
		t.Errorf("error running CopyGCSObjects.validate(): %v", err)
	}

	for _, ws := range []*CopyGCSObjects{
		{{Source: "gs://bucket1", Destination: ""}},
		{{Source: "", Destination: "gs://bucket1"}},
		{{Source: "gs://bucket2", Destination: "gs://bucket1"}},
		{{Source: "gs://bucket1", Destination: "gs://bucket2"}},
		{{Source: "gs://bucket1", Destination: "gs://bucket3"}},
		{{Source: "gs://bucket1", Destination: "gs://bucket1", ACLRules: []*storage.ACLRule{{Role: "owner"}}}},
		{{Source: "gs://bucket1", Destination: "gs://bucket1", ACLRules: []*storage.ACLRule{{Entity: "allUsers", Role: "owner"}}}},
		{{Source: "gs://bucket1", Destination: "gs://bucket1", ACLRules: []*storage.ACLRule{{Entity: "someUser", Role: "OWNER"}}}},
	} {
		if err := ws.validate(ctx, s); err == nil {
			t.Error("expected error")
		}
		// Reset.
		readableBkts = validatedBkts{}
		writableBkts = validatedBkts{}
	}
}

func TestCopyGCSObjectsRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	ws := &CopyGCSObjects{
		{Source: "gs://bucket", Destination: "gs://bucket"},
		{Source: "gs://bucket/object", Destination: "gs://bucket/object"},
		{Source: "gs://bucket/object", Destination: "gs://bucket/object", ACLRules: []*storage.ACLRule{{Entity: "allUsers", Role: "OWNER"}}},
		{Source: "gs://bucket/object/", Destination: "gs://bucket/object/", ACLRules: []*storage.ACLRule{{Entity: "allUsers", Role: "OWNER"}}},
	}
	if err := ws.run(ctx, s); err != nil {
		t.Errorf("error running CopyGCSObjects.run(): %v", err)
	}

	for _, ws := range []*CopyGCSObjects{
		{{Source: "gs://bucketerror", Destination: ""}},
		{{Source: "", Destination: "gs://bucketerror"}},
		{{Source: "gs://bucketerror/object/", Destination: "gs://bucketerror/object/", ACLRules: []*storage.ACLRule{{Entity: "someUser", Role: "OWNER"}}}},
	} {
		if err := ws.run(ctx, s); err == nil {
			t.Error("expected error")
		}
	}
}

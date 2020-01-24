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
	"errors"
	"strings"
	"testing"
)

func TestAddErrs(t *testing.T) {
	tests := []struct {
		desc string
		base DError
		errs []error
		want DError
	}{
		{"add nil to nil case", nil, nil, nil},
		{"add error to nil case", nil, []error{errors.New("foo")}, &dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{""}}},
		{"add nil to DError case", &dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{""}}, nil, &dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{""}}},
		{"add errors to DError case", &dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{""}}, []error{errors.New("bar"), errors.New("baz")}, &dErrImpl{errs: []error{errors.New("foo"), errors.New("bar"), errors.New("baz")}, errsType: []string{"", "", ""}}},
	}

	for _, tt := range tests {
		got := addErrs(tt.base, tt.errs...)
		if diffRes := diff(got, tt.want, 0); diffRes != "" {
			t.Errorf("%s: (-got,+want)\n%s", tt.desc, diffRes)
		}
		if diffRes := diff(tt.base, tt.want, 0); tt.base != nil && diffRes != "" {
			t.Errorf("%s: base DError not modified as expected: (-got,+want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestDErrError(t *testing.T) {
	tests := []struct {
		desc string
		err  DError
		want string
	}{
		{"no error type case", &dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{""}}, "foo"},
		{"error type case", &dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{"FOO"}}, "FOO: foo"},
		{
			"multierror case",
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errsType: []string{"", ""}},
			"Multiple errors:\n* foo\n* bar",
		},
	}

	for _, tt := range tests {
		got := tt.err.Error()
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.desc, got, tt.want)
		}
	}
}

func TestErrf(t *testing.T) {
	got := Errf("%s %s", "hello", "world")

	want := &dErrImpl{errs: []error{errors.New("hello world")}, errsType: []string{""}}
	if diffRes := diff(got, want, 0); diffRes != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestTypedErrf(t *testing.T) {
	got := typedErrf("FOO", "%s %s", "hello", "world")
	want := &dErrImpl{errs: []error{errors.New("hello world")}, errsType: []string{"FOO"}}
	if diffRes := diff(got, want, 0); diffRes != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDErrImplAdd(t *testing.T) {
	tests := []struct {
		desc        string
		base        *dErrImpl
		add         error
		want        *dErrImpl
		wantErrType string
	}{
		{
			"add error case",
			&dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{"FOO"}},
			errors.New("bar"),
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errsType: []string{"FOO", ""}},
			multiError,
		},
		{
			"add dErrImpl case",
			&dErrImpl{errs: []error{errors.New("foo")}, errsType: []string{"FOO"}},
			&dErrImpl{errs: []error{errors.New("bar")}, errsType: []string{"FOO"}},
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errsType: []string{"FOO", "BAR"}},
			multiError,
		},
		{
			"add " + multiError + " case",
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errsType: []string{"FOO", "BAR"}},
			&dErrImpl{errs: []error{errors.New("baz"), errors.New("gaz")}, errsType: []string{"FOO", "BAR"}},
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar"), errors.New("baz"), errors.New("gaz")}, errsType: []string{"FOO", "BAR", "FOO", "BAR"}},
			multiError,
		},
	}

	for _, tt := range tests {
		tt.base.add(tt.add)
		if diffRes := diff(tt.base, tt.want, 0); diffRes != "" {
			t.Errorf("%s: base dErrImpl not modified as expected: (-got,+want)\n%s", tt.desc, diffRes)
		}
		if diffRes := diff(tt.base.etype(), tt.wantErrType, 0); diffRes != "" {
			t.Errorf("%s: base dErrImpl not modified as expected: (-got,+want)\n%s", tt.desc, diffRes)
		}
	}
}

func TestNestedAnonymizedDErrorMessage(t *testing.T) {
	innerDErr1 := Errf("inner error 1: %v %v", "root cause 1", "root cause 2")
	innerDErr2 := Errf("inner error 2: %v %v", "root cause 3", "root cause 4")
	innerDErr1.add(innerDErr2)
	outerDErr := wrapErrf(innerDErr1, "outer error: %v", "bad news")

	gotAnonymizedMsg := strings.Join(outerDErr.AnonymizedErrs(), ",")
	wantAnonymizedMsg := "outer error: %v: inner error 1: %v %v; inner error 2: %v %v"
	if diffRes := diff(wantAnonymizedMsg, gotAnonymizedMsg, 0); diffRes != "" {
		t.Errorf("nested DError doesn't have correct anonymized error message: (-got,+want)\n%s", diffRes)
	}

	gotMsg := outerDErr.Error()
	wantMsg := "outer error: bad news: Multiple errors:\n" +
		"* inner error 1: root cause 1 root cause 2\n" +
		"* inner error 2: root cause 3 root cause 4"
	if diffRes := diff(wantMsg, gotMsg, 0); diffRes != "" {
		t.Errorf("nested DError doesn't have correct error message: (-got,+want)\n%s", diffRes)
	}

}

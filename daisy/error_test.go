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
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestAddErrs(t *testing.T) {
	tests := []struct {
		desc string
		base dErr
		errs []error
		want dErr
	}{
		{"add nil to nil case", nil, nil, nil},
		{"add error to nil case", nil, []error{errors.New("foo")}, &dErrImpl{errs: []error{errors.New("foo")}}},
		{"add nil to dErr case", &dErrImpl{errs: []error{errors.New("foo")}}, nil, &dErrImpl{errs: []error{errors.New("foo")}}},
		{"add errors to dErr case", &dErrImpl{errs: []error{errors.New("foo")}}, []error{errors.New("bar"), errors.New("baz")}, &dErrImpl{errs: []error{errors.New("foo"), errors.New("bar"), errors.New("baz")}, errType: multiError}},
	}

	for _, tt := range tests {
		got := addErrs(tt.base, tt.errs...)
		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: (-got,+want)\n%s", tt.desc, diff)
		}
		if diff := pretty.Compare(tt.base, tt.want); tt.base != nil && diff != "" {
			t.Errorf("%s: base dErr not modified as expected: (-got,+want)\n%s", tt.desc, diff)
		}
	}
}

func TestDErrError(t *testing.T) {
	tests := []struct {
		desc string
		err  dErr
		want string
	}{
		{"no error type case", &dErrImpl{errs: []error{errors.New("foo")}}, "foo"},
		{"error type case", &dErrImpl{errs: []error{errors.New("foo")}, errType: "FOO"}, "FOO: foo"},
		{
			"multierror case",
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errType: multiError},
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
	got := errf("%s %s", "hello", "world")

	want := &dErrImpl{errs: []error{errors.New("hello world")}}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diff)
	}
}

func TestTypedErrf(t *testing.T) {
	got := typedErrf("FOO", "%s %s", "hello", "world")
	want := &dErrImpl{errs: []error{errors.New("hello world")}, errType: "FOO"}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diff)
	}
}

func TestDErrImplAdd(t *testing.T) {
	tests := []struct {
		desc string
		base *dErrImpl
		add  error
		want *dErrImpl
	}{
		{
			"add error case",
			&dErrImpl{errs: []error{errors.New("foo")}, errType: "FOO"},
			errors.New("bar"),
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errType: multiError}},
		{
			"add dErrImpl case",
			&dErrImpl{errs: []error{errors.New("foo")}, errType: "FOO"},
			&dErrImpl{errs: []error{errors.New("bar")}, errType: "BAR"},
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errType: multiError},
		},
		{
			"add " + multiError + " case",
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar")}, errType: multiError},
			&dErrImpl{errs: []error{errors.New("baz"), errors.New("gaz")}, errType: multiError},
			&dErrImpl{errs: []error{errors.New("foo"), errors.New("bar"), errors.New("baz"), errors.New("gaz")}, errType: multiError},
		},
	}

	for _, tt := range tests {
		tt.base.add(tt.add)
		if diff := pretty.Compare(tt.base, tt.want); diff != "" {
			t.Errorf("%s: base dErrImpl not modified as expected: (-got,+want)\n%s", tt.desc, diff)
		}
	}
}

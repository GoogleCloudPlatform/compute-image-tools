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
	"fmt"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestErrorCast(t *testing.T) {
	var err *dError

	if err.cast() != nil {
		t.Error("should have returned a nil error")
	}
	err = errorf("hey!")
	if err.cast() == nil {
		t.Error("should have returned an error")
	}
}

func TestErrorError(t *testing.T) {
	msg := "foo"
	tests := []struct {
		desc, errType string
		want          string
	}{
		{"no error type case", untypedError, msg},
		{"error type case", "MYERROR", fmt.Sprintf("MYERROR: %s", msg)},
	}

	for _, tt := range tests {
		e := dError{Msg: msg, ErrType: tt.errType}
		got := e.Error()
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.desc, got, tt.want)
		}
	}
}

func TestErrorErrorf(t *testing.T) {
	got := errorf("%s %s", "hello", "world")

	want := dError{Msg: "hello world"}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diff)
	}
}

func TestErrorTypedErrorf(t *testing.T) {
	got := typedErrorf("MYERROR", "%s %s", "hello", "world")
	want := dError{Msg: "hello world", ErrType: "MYERROR"}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diff)
	}
}

func TestErrorsAdd(t *testing.T) {
	var errs dErrors

	errs.add(nil)
	if errs != nil {
		t.Errorf("errs should be nil")
	}

	err := &dError{Msg: "error"}
	errs.add(err)
	if diff := pretty.Compare(errs, dErrors{err}); diff != "" {
		t.Errorf("errs not modified as expected: (-got,+want)\n%s", diff)
	}
}

func TestErrorsCast(t *testing.T) {
	var errs dErrors

	if errs.cast() != nil {
		t.Error("should have returned a nil error")
	}
	errs.add(errorf("hey!"))
	if errs.cast() == nil {
		t.Error("should have returned an error")
	}
}

func TestErrorsError(t *testing.T) {
	var errs dErrors

	errs.add(errorf("hey!"))
	want := "Errors:\n  * hey!"
	got := errs.Error()
	if got != want {
		t.Errorf("Error did not print as expected: got %q, want %q", got, want)
	}
}

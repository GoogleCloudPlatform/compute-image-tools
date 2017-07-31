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
	"github.com/kylelemons/godebug/pretty"
	"testing"
)

func TestErrorCast(t *testing.T) {
	var err *Error

	if err.cast() != nil {
		t.Error("should have returned a nil error")
	}
	err = Errorf("hey!")
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
		{"no error type case", ERRNOTYPE, msg},
		{"error type case", "MYERROR", fmt.Sprintf("MYERROR: %s", msg)},
	}

	for _, tt := range tests {
		e := Error{Msg: msg, ErrType: tt.errType}
		got := e.Error()
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.desc, got, tt.want)
		}
	}
}

func TestErrorErrorf(t *testing.T) {
	got := Errorf("%s %s", "hello", "world")

	want := Error{Msg: "hello world"}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diff)
	}
}

func TestErrorTypedErrorf(t *testing.T) {
	got := TypedErrorf("MYERROR", "%s %s", "hello", "world")
	want := Error{Msg: "hello world", ErrType: "MYERROR"}
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("Error not created as expected: (-got,+want)\n%s", diff)
	}
}

func TestErrorsAdd(t *testing.T) {
	var errs Errors

	errs.add(nil)
	if errs != nil {
		t.Errorf("errs should be nil")
	}

	err := &Error{Msg: "error"}
	errs.add(err)
	if diff := pretty.Compare(errs, Errors{err}); diff != "" {
		t.Errorf("errs not modified as expected: (-got,+want)\n%s", diff)
	}
}

func TestErrorsCast(t *testing.T) {
	var errs Errors

	if errs.cast() != nil {
		t.Error("should have returned a nil error")
	}
	errs.add(Errorf("hey!"))
	if errs.cast() == nil {
		t.Error("should have returned an error")
	}
}

func TestErrorsError(t *testing.T) {
	var errs Errors

	errs.add(Errorf("hey!"))
	want := "Errors:\n  * hey!"
	got := errs.Error()
	if got != want {
		t.Errorf("Error did not print as expected: got %q, want %q", got, want)
	}
}

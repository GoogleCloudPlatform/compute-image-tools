package workflow

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

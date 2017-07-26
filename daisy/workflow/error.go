package workflow

import (
	"fmt"
	"strings"
)

const (
	ERRNOTYPE = ""
)

type Error struct {
	Msg, ErrType string
}

func (e *Error) cast() error {
	if e == nil {
		return nil
	}
	return e
}

func (e *Error) Error() string {
	if e.ErrType != ERRNOTYPE {
		return fmt.Sprintf("%s: %s", e.ErrType, e.Msg)
	}
	return e.Msg
}

func Errorf(format string, a ...interface{}) *Error {
	return &Error{Msg: fmt.Sprintf(format, a...)}
}

func TypedErrorf(errType, format string, a ...interface{}) *Error {
	err := Errorf(format, a...)
	err.ErrType = errType
	return err
}

type Errors []*Error

func (e *Errors) cast() error {
	if e == nil || *e == nil {
		return nil
	}
	return e
}

func (e *Errors) add(errs ...*Error) {
	for _, err := range errs {
		if err != nil {
			*e = append(*e, err)
		}
	}
}

func (e *Errors) Error() string {
	var errStrs []string
	for _, err := range *e {
		errStrs = append(errStrs, fmt.Sprintf("  * %s", err.Error()))
	}
	return fmt.Sprintf("Errors:\n%s", strings.Join(errStrs, "\n"))
}

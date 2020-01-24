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
	"strings"
)

const (
	untypedError              = ""
	multiError                = "MultiError"
	fileIOError               = "FileIOError"
	resourceDNEError          = "ResourceDoesNotExist"
	imageObsoleteDeletedError = "ImageObsoleteOrDeleted"

	apiError    = "APIError"
	apiError404 = "APIError404"
)

// DError is a Daisy external error type.
// It has:
// - optional error typing
// - multiple error aggregation
// - safe error messages in which privacy information is removed
//
// Default implementation:
// The default DError implementation is flat, DError.add(anotherDErr) will merge the two dErrs
// into a single, flat DError instead of making anotherDErr a child to DError.
type DError interface {
	error

	// add shouldn't be called directly, instead call addErrs(DError, error).
	// This assists with nil dErrs. addErrs(nil, e) will return a new DError.
	add(error)

	etype() string
	errors() []error
	errorsType() []string
	AnonymizedErrs() []string
	CausedByErrType(t string) bool
}

// addErrs adds an error to a DError.
// The DError can be nil. If both the DError and errors are nil, a nil DError is returned.
// If DError is nil, but errors are not nil, a new DError is instantiated, the errors are added,
// and the new DError is returned.
// Any nil error in errs is disregarded. Therefore, `var e DError; e = addErrs(e, nil)`
// preserves e's nil-ness.
func addErrs(e DError, errs ...error) DError {
	for _, err := range errs {
		if err != nil {
			if e == nil {
				e = &dErrImpl{}
			}
			e.add(err)
		}
	}
	return e
}

// Errf returns a DError by constructing error message with given format.
func Errf(format string, a ...interface{}) DError {
	return newErr(format, fmt.Errorf(format, a...))
}

// wrapErrf returns a DError by keeping errors type and replacing original error message.
func wrapErrf(e DError, formatPrefix string, a ...interface{}) DError {
	f := fmt.Sprintf("%v: %v", formatPrefix, strings.Join(e.AnonymizedErrs(), "; "))
	return &dErrImpl{
		errs:           []error{fmt.Errorf("%v: %v", fmt.Sprintf(formatPrefix, a...), e.Error())},
		errsType:       e.errorsType(),
		anonymizedErrs: []string{f},
	}
}

// newErr returns a DError. newErr is used to wrap another error as a DError.
// If e is already a DError, e is copied and returned.
// If e is a normal error, anonymizedErrMsg is used to hide privacy info.
// If e is nil, nil is returned.
func newErr(anonymizedErrMsg string, e error) DError {
	if e == nil {
		return nil
	}
	if dE, ok := e.(*dErrImpl); ok {
		return dE
	}
	return &dErrImpl{errs: []error{e}, errsType: []string{""}, anonymizedErrs: []string{anonymizedErrMsg}}
}

// ToDError returns a DError. ToDError is used to wrap another error as a DError.
// If e is already a DError, e is copied and returned.
// If e is a normal error, error message is reused as format.
// If e is nil, nil is returned.
func ToDError(e error) DError {
	if e == nil {
		return nil
	}
	if dE, ok := e.(*dErrImpl); ok {
		return dE
	}
	return &dErrImpl{errs: []error{e}, errsType: []string{""}, anonymizedErrs: []string{e.Error()}}
}

func typedErr(errType string, safeErrMsg string, e error) DError {
	if e == nil {
		return nil
	}
	safeErrMsg = fmt.Sprintf("%v: %v", errType, safeErrMsg)
	dE := newErr(safeErrMsg, e)
	dE.(*dErrImpl).errsType = []string{errType}
	return dE
}

func typedErrf(errType, format string, a ...interface{}) DError {
	return typedErr(errType, format, fmt.Errorf(format, a...))
}

type dErrImpl struct {
	errs           []error
	errsType       []string
	anonymizedErrs []string
}

func (e *dErrImpl) add(err error) {
	if e2, ok := err.(*dErrImpl); ok {
		e.merge(e2)
	} else if !ok {
		// This is some other error type. Add it.
		e.errs = append(e.errs, err)
		e.errsType = append(e.errsType, "")
	}
}

func (e *dErrImpl) Error() string {
	if e.len() == 0 {
		return ""
	}
	if e.len() == 1 {
		errStr := e.errs[0].Error()
		if len(e.errsType) == 1 && e.errsType[0] != "" {
			return fmt.Sprintf("%s: %s", e.errsType[0], errStr)
		}
		return errStr
	}

	// Multiple error handling.
	pre := "* "
	lines := make([]string, e.len())
	for i, err := range e.errs {
		lines[i] = pre + err.Error()
	}

	return "Multiple errors:\n" + strings.Join(lines, "\n")
}

func (e *dErrImpl) AnonymizedErrs() []string {
	return e.anonymizedErrs
}

func (e *dErrImpl) len() int {
	return len(e.errs)
}

func (e *dErrImpl) merge(e2 *dErrImpl) {
	if e2.len() > 0 {
		e.errs = append(e.errs, e2.errs...)
		e.errsType = append(e.errsType, e2.errsType...)
		e.anonymizedErrs = append(e.anonymizedErrs, e2.anonymizedErrs...)
	}
}

func (e *dErrImpl) etype() string {
	if e.len() > 1 {
		return multiError
	} else if e.len() == 1 && len(e.errsType) == 1 {
		return e.errsType[0]
	} else {
		return ""
	}
}

func (e *dErrImpl) errorsType() []string {
	return e.errsType
}

func (e *dErrImpl) errors() []error {
	return e.errs
}

func (e *dErrImpl) CausedByErrType(t string) bool {
	for _, et := range e.errsType {
		if et == t {
			return true
		}
	}
	return false
}

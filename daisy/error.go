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

// dErr is a Daisy internal error type.
// It has:
// - optional error typing
// - multiple error aggregation
//
// Default implementation:
// The default dErr implementation is flat, dErr.add(anotherDErr) will merge the two dErrs
// into a single, flat dErr instead of making anotherDErr a child to dErr.
type dErr interface {
	error

	// add shouldn't be called directly, instead call addErrs(dErr, error).
	// This assists with nil dErrs. addErrs(nil, e) will return a new dErr.
	add(error)
	Type() string
}

// addErrs adds an error to a dErr.
// The dErr can be nil. If both the dErr and errors are nil, a nil dErr is returned.
// If dErr is nil, but errors are not nil, a new dErr is instantiated, the errors are added,
// and the new dErr is returned.
// Any nil error in errs is disregarded. Therefore, `var e dErr; e = addErrs(e, nil)`
// preserves e's nil-ness.
func addErrs(e dErr, errs ...error) dErr {
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

func errf(format string, a ...interface{}) dErr {
	return newErr(fmt.Errorf(format, a...))
}

// newErr returns a dErr. newErr is used to wrap another error as a dErr.
// If e is already a dErr, e is copied and returned.
// If e is nil, nil is returned.
func newErr(e error) dErr {
	if e == nil {
		return nil
	}
	if dE, ok := e.(*dErrImpl); ok {
		return dE
	}
	return &dErrImpl{errs: []error{e}}
}

func typedErr(errType string, e error) dErr {
	if e == nil {
		return nil
	}
	dE := newErr(e)
	dE.(*dErrImpl).errType = errType
	return dE
}

func typedErrf(errType, format string, a ...interface{}) dErr {
	return typedErr(errType, fmt.Errorf(format, a...))
}

type dErrImpl struct {
	errs    []error
	errType string
}

func (e *dErrImpl) add(err error) {
	if e2, ok := err.(*dErrImpl); ok {
		e.merge(e2)
	} else if !ok {
		// This is some other error type. Add it.
		e.errs = append(e.errs, err)
	}
	if e.len() > 1 {
		e.errType = multiError
	}
}

func (e *dErrImpl) Error() string {
	if e.len() == 0 {
		return ""
	}
	if e.len() == 1 {
		errStr := e.errs[0].Error()
		if e.errType != "" {
			return fmt.Sprintf("%s: %s", e.errType, errStr)
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

func (e *dErrImpl) len() int {
	return len(e.errs)
}

func (e *dErrImpl) merge(e2 *dErrImpl) {
	if e2.len() > 0 {
		e.errs = append(e.errs, e2.errs...)
		// Take e2's type. This solves the situation of e having 0 errors, and e2 having 1.
		// Of course, there is a possibility of len(e) > 0 and len(e2) > 1, in which case,
		// the type should be a multiError.
		e.errType = e2.errType
		if e.len() > 1 {
			e.errType = multiError
		}
	}
}

func (e *dErrImpl) Type() string {
	return e.errType
}

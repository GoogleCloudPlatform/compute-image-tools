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
	untypedError     = ""
	resourceDNEError = "ResourceDoesNotExist"
)

type dError struct {
	Msg, ErrType string
}

func (e *dError) cast() error {
	if e == nil {
		return nil
	}
	return e
}

func (e *dError) Error() string {
	if e.ErrType != untypedError {
		return fmt.Sprintf("%s: %s", e.ErrType, e.Msg)
	}
	return e.Msg
}

func errorf(format string, a ...interface{}) *dError {
	return &dError{Msg: fmt.Sprintf(format, a...)}
}

func typedErrorf(errType, format string, a ...interface{}) *dError {
	err := errorf(format, a...)
	err.ErrType = errType
	return err
}

type dErrors []*dError

func (e *dErrors) cast() error {
	if e == nil || *e == nil {
		return nil
	}
	return e
}

func (e *dErrors) add(errs ...*dError) {
	for _, err := range errs {
		if err != nil {
			*e = append(*e, err)
		}
	}
}

func (e *dErrors) Error() string {
	var errStrs []string
	for _, err := range *e {
		errStrs = append(errStrs, fmt.Sprintf("  * %s", err))
	}
	return fmt.Sprintf("Errors:\n%s", strings.Join(errStrs, "\n"))
}

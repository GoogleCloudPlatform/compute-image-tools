//  Copyright 2018 Google Inc. All Rights Reserved.
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

// Package junitxml provides helpers around creating junit XML data.
package junitxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// NewTestSuite creates a new TestSuite.
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		Name:  name,
		start: time.Now(),
	}
}

// TestSuite is a junitxml TestSuite.
type TestSuite struct {
	XMLName  xml.Name `xml:"testsuite"`
	Name     string   `xml:"name,attr"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Errors   int      `xml:"errors,attr"`
	Disabled int      `xml:"disabled,attr"`
	Skipped  int      `xml:"skipped,attr"`
	Time     float64  `xml:"time,attr"`

	TestCase []*TestCase `xml:"testcase"`

	start time.Time
}

// Finish marks a TestSuite as finished and sends it in the provided channel.
func (s *TestSuite) Finish(tests chan *TestSuite) {
	s.Time = time.Since(s.start).Seconds()
	s.Tests = len(s.TestCase)
	for _, tc := range s.TestCase {
		if tc.Failure != nil {
			s.Failures++
		}
		if tc.Skipped != nil {
			s.Skipped++
		}
	}
	tests <- s
}

// NewTestCase creates a new TestCase.
func NewTestCase(classname, name string) *TestCase {
	return &TestCase{
		Classname: classname,
		Name:      name,
		ID:        uuid.New().String(),
		start:     time.Now(),
	}
}

// TestCase is a junitxml TestCase.
type TestCase struct {
	Classname string        `xml:"classname,attr"`
	ID        string        `xml:"id,attr"`
	Name      string        `xml:"name,attr"`
	Time      float64       `xml:"time,attr"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`

	start time.Time
	buf   bytes.Buffer
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

type junitFailure struct {
	FailMessage string `xml:",chardata"`
	FailType    string `xml:"type,attr"`
}

// Logf logs to the TestCase SystemOut.
func (c *TestCase) Logf(msg string, args ...interface{}) {
	c.buf.WriteString(fmt.Sprintf(msg, args...) + "\n")
}

// WriteFailure marks a TestCase as failed with the provided message.
func (c *TestCase) WriteFailure(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	c.Logf(msg)
	c.Failure = &junitFailure{
		FailMessage: msg,
		FailType:    "Failure",
	}
}

// WriteSkipped marks a TestCase as skipped with the provided message.
func (c *TestCase) WriteSkipped(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	c.Skipped = &junitSkipped{
		Message: msg,
	}
}

// Finish marks a TestCase as finished and sends it in the provided channel.
func (c *TestCase) Finish(tests chan *TestCase) {
	c.Time = time.Since(c.start).Seconds()
	c.SystemOut = c.buf.String()
	tests <- c
}

// FilterTestCase markes a TestCase as skipped if the name matches the regex.
func (c *TestCase) FilterTestCase(regex *regexp.Regexp) bool {
	if regex != nil && !regex.MatchString(c.Name) {
		c.WriteSkipped("Test does not match filter: %q", regex.String())
		return true
	}

	return false
}

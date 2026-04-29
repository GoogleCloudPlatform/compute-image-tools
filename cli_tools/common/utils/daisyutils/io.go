//  Copyright 2019 Google Inc. All Rights Reserved.
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

package daisyutils

import "io"

// ByteCountingReader forwards calls to a delegate reader, keeping
// track of the number bytes that have been read in `BytesRead`.
// Errors are propagated unchanged.
type ByteCountingReader struct {
	r         io.Reader
	BytesRead int64
}

// NewByteCountingReader is a contructor for ByteCountingReader.
func NewByteCountingReader(r io.Reader) *ByteCountingReader {
	return &ByteCountingReader{r, 0}
}

func (l *ByteCountingReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	l.BytesRead += int64(n)
	return n, err
}

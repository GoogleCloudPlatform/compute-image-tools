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

package pathutils

import (
	"math/rand"
	"net/url"
	"path"
	"strings"
	"time"
)

// RandString generates a random string of n length.
func RandString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := "bdghjlmnpqrstvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}

func JoinUrl(urlStr string, pathStr string) string {
	u, _ := url.Parse(urlStr)
	u.Path = path.Join(u.Path, pathStr)
	return u.String()
}

// Ensures url ends with a /
func ToDirectoryUrl(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

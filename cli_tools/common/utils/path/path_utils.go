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
	"path/filepath"
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

// JoinURL extends URL with additional path
func JoinURL(urlStr string, pathStr string) string {
	u, _ := url.Parse(urlStr)
	u.Path = path.Join(u.Path, pathStr)
	return u.String()
}

// ToDirectoryURL ensures url ends with a /
func ToDirectoryURL(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

// ToWorkingDir gets absolute path from given relative path of the directory
func ToWorkingDir(relDir string, currentExecutablePath string) string {
	wd, err := filepath.Abs(filepath.Dir(currentExecutablePath))
	if err == nil {
		return path.Join(wd, relDir)
	}
	return relDir
}
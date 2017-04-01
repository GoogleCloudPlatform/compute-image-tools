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

package workflow

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var (
	bucket = `([a-z0-9][-_.a-z0-9]*)`
	object = `(.+)`
	// Many of the Google Storage URLs are supported below.
	// It is preferred that customers specify their object using
	// its gs://<bucket>/<object> URL.
	bucketRegex = regexp.MustCompile(fmt.Sprintf(`^gs://%s$`, bucket))
	gsRegex     = regexp.MustCompile(fmt.Sprintf(`^gs://%s/%s$`, bucket, object))
	// Check for the Google Storage URLs:
	// http://<bucket>.storage.googleapis.com/<object>
	// https://<bucket>.storage.googleapis.com/<object>
	gsHTTPRegex = regexp.MustCompile(fmt.Sprintf(`^http[s]?://%s\.storage\.googleapis\.com/%s$`, bucket, object))
	// Check for the other possible Google Storage URLs:
	// http://storage.googleapis.com/<bucket>/<object>
	// https://storage.googleapis.com/<bucket>/<object>
	//
	// The following are deprecated but checked:
	// http://commondatastorage.googleapis.com/<bucket>/<object>
	// https://commondatastorage.googleapis.com/<bucket>/<object>
	gsHTTPRegex2 = regexp.MustCompile(fmt.Sprintf(`^http[s]?://(?:commondata)?storage\.googleapis\.com/%s/%s$`, bucket, object))
)

func containsString(s string, ss []string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}

// filter creates a copy of ss, excluding any instances of s.
func filter(ss []string, s string) []string {
	result := []string{}
	for _, element := range ss {
		if element != s {
			result = append(result, element)
		}
	}
	return result
}

func isLink(s string) bool {
	return strings.Contains(s, "/")
}

func randString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}

func splitGCSPath(p string) (string, string, error) {
	for _, rgx := range []*regexp.Regexp{gsRegex, gsHTTPRegex, gsHTTPRegex2} {
		matches := rgx.FindStringSubmatch(p)
		if matches != nil {
			return matches[1], matches[2], nil
		}
	}
	matches := bucketRegex.FindStringSubmatch(p)
	if matches != nil {
		return matches[1], "", nil
	}
	return "", "", fmt.Errorf("%q is not a valid GCS path", p)
}

// substitute analyzes an element for string values and replaces
// found instances of indicated substrings with given replacement strings.
// Private fields of a struct are not modified.
func substitute(v reflect.Value, replacer *strings.Replacer) {
	if !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Ptr:
		// A nil entry will cause additional reflect operations to panic.
		if v.IsNil() {
			return
		}
	}

	if v.Kind() == reflect.Ptr {
		// Dereference me.
		substitute(v.Elem(), replacer)
	}

	// If this is a string, run the replacer on it.
	switch v.Interface().(type) {
	case string:
		v.SetString(replacer.Replace(v.String()))
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			substitute(v.Field(i), replacer)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			substitute(v.Index(i), replacer)
		}
	case reflect.Map:
		kvs := v.MapKeys()
		for _, kv := range kvs {
			vv := v.MapIndex(kv)

			// Create new mutable copies of the key and value.
			// Modify the copies.
			newKv := reflect.New(kv.Type()).Elem()
			newKv.Set(kv)
			newVv := reflect.New(vv.Type()).Elem()
			newVv.Set(vv)
			substitute(newKv, replacer)
			substitute(newVv, replacer)

			// Delete the old key-value.
			v.SetMapIndex(kv, reflect.Value{})
			// Set the new key-value.
			v.SetMapIndex(newKv, newVv)
		}
	}
}

func xor(x, y bool) bool {
	return x != y
}

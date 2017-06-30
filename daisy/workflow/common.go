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
	"path"
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
	bucketRegex = regexp.MustCompile(fmt.Sprintf(`^gs://%s/?$`, bucket))
	gsRegex     = regexp.MustCompile(fmt.Sprintf(`^gs://%s/%s$`, bucket, object))
	// Check for the Google Storage URLs:
	// http://<bucket>.storage.googleapis.com/<object>
	// https://<bucket>.storage.googleapis.com/<object>
	gsHTTPRegex1 = regexp.MustCompile(fmt.Sprintf(`^http[s]?://%s\.storage\.googleapis\.com/%s$`, bucket, object))
	// http://storage.cloud.google.com/<bucket>/<object>
	// https://storage.cloud.google.com/<bucket>/<object>
	gsHTTPRegex2 = regexp.MustCompile(fmt.Sprintf(`^http[s]?://storage\.cloud\.google\.com/%s/%s$`, bucket, object))
	// Check for the other possible Google Storage URLs:
	// http://storage.googleapis.com/<bucket>/<object>
	// https://storage.googleapis.com/<bucket>/<object>
	//
	// The following are deprecated but checked:
	// http://commondatastorage.googleapis.com/<bucket>/<object>
	// https://commondatastorage.googleapis.com/<bucket>/<object>
	gsHTTPRegex3 = regexp.MustCompile(fmt.Sprintf(`^http[s]?://(?:commondata)?storage\.googleapis\.com/%s/%s$`, bucket, object))

	gcsAPIBase = "https://storage.cloud.google.com"
)

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

func getGCSAPIPath(p string) (string, error) {
	b, o, e := splitGCSPath(p)
	if e != nil {
		return "", e
	}
	return fmt.Sprintf("%s/%s", gcsAPIBase, path.Join(b, o)), nil
}

func minInt(x int, ys ...int) int {
	for _, y := range ys {
		if y < x {
			x = y
		}
	}
	return x
}

func randString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := "bdghjlmnpqrstvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}

func splitGCSPath(p string) (string, string, error) {
	for _, rgx := range []*regexp.Regexp{gsRegex, gsHTTPRegex1, gsHTTPRegex2, gsHTTPRegex3} {
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

func strIn(s string, ss []string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func strLitPtr(s string) *string {
	return &s
}

func strOr(s string, ss ...string) string {
	ss = append([]string{s}, ss...)
	for _, st := range ss {
		if st != "" {
			return st
		}
	}
	return ""
}

// substitute runs replacer on string elements within a complex data structure
// (except those contained in private data structure fields).
func substitute(v reflect.Value, replacer *strings.Replacer) {
	traverseData(v, func(val reflect.Value) error {
		switch val.Interface().(type) {
		case string:
			val.SetString(replacer.Replace(val.String()))
		}
		return nil
	})
}

// traverseData traverses complex data structures and runs
// a function, f, on its basic data types.
// Traverses arrays, maps, slices, and public fields of structs.
// For example, f will be run on bool, int, string, etc.
// Slices, maps, and structs will not have f called on them, but will
// traverse their subelements.
// Errors returned from f will be returned by traverseDataStructure.
func traverseData(v reflect.Value, f func(reflect.Value) error) error {
	if !v.CanSet() {
		// Don't run on private fields.
		return nil
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Func:
		return nil
	case reflect.Interface, reflect.Ptr, reflect.UnsafePointer:
		if v.IsNil() {
			return nil
		}
		// I'm a pointer, dereference me.
		return traverseData(v.Elem(), f)
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if err := traverseData(v.Index(i), f); err != nil {
				return err
			}
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
			if err := traverseData(newKv, f); err != nil {
				return err
			}
			if err := traverseData(newVv, f); err != nil {
				return err
			}

			// Delete the old key-value.
			v.SetMapIndex(kv, reflect.Value{})
			// Set the new key-value.
			v.SetMapIndex(newKv, newVv)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := traverseData(v.Field(i), f); err != nil {
				return err
			}
		}
	default:
		// As far as I can tell, this is a basic data type. Run f on it.
		return f(v)
	}
	return nil
}

func xor(x, y bool) bool {
	return x != y
}

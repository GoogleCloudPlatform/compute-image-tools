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
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var (
	bucket = `([a-z0-9][-_.a-z0-9]*)`
	object = `(.+)`
	// Many of the Google Storage URLs are supported below.
	// It is preferred that customers specify their object using
	// its gs://<bucket>/<object> URL.
	gsRegex = regexp.MustCompile(fmt.Sprintf(`^gs://%s/%s`, bucket, object))
	// Check for the Google Storage URLs:
	// http://<bucket>.storage.googleapis.com/<object>
	// https://<bucket>.storage.googleapis.com/<object>
	gsHTTPRegex = regexp.MustCompile(fmt.Sprintf(`^http[s]?://%s\.storage\.googleapis\.com/%s`, bucket, object))
	// Check for the other possible Google Storage URLs:
	// http://storage.googleapis.com/<bucket>/<object>
	// https://storage.googleapis.com/<bucket>/<object>
	//
	// The following are deprecated but checked:
	// http://commondatastorage.googleapis.com/<bucket>/<object>
	// https://commondatastorage.googleapis.com/<bucket>/<object>
	gsHTTPRegex2 = regexp.MustCompile(fmt.Sprintf(`^http[s]?://(?:commondata)?storage\.googleapis\.com/%s/%s`, bucket, object))
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
	return "", "", fmt.Errorf("%q is not a valid GCS path", p)
}

// substitute iterates through the public fields of the struct represented
// by s, if the field type matches one of the known types the provided
// replacer.Replace is run on all string values replacing the original value
// in the underlying struct.
// Exceptions: will not change Vars fields or Workflow fields of SubWorkflow types.
func substitute(s reflect.Value, replacer *strings.Replacer) {
	if s.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < s.NumField(); i++ {
		// Skip the Vars field as thats where the replacer gets populated from.
		if s.Type().Field(i).Name == "Vars" {
			continue
		}

		f := s.Field(i)
		// Don't attempt to modify private fields.
		if !f.CanSet() {
			continue
		}

		switch f.Kind() {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
			// A nil entry will cause additional reflect operations to panic.
			if f.IsNil() {
				continue
			}
		}

		raw := f.Interface()
		switch raw.(type) {
		case string:
			f.SetString(replacer.Replace(f.String()))
		case []string:
			var newSlice []string
			for _, v := range raw.([]string) {
				newSlice = append(newSlice, replacer.Replace(v))
			}
			f.Set(reflect.ValueOf(newSlice))
		case map[string]string:
			newMap := map[string]string{}
			for k, v := range raw.(map[string]string) {
				newMap[replacer.Replace(k)] = replacer.Replace(v)
			}
			f.Set(reflect.ValueOf(newMap))
		case map[string][]string:
			newMap := map[string][]string{}
			for k, v := range raw.(map[string][]string) {
				var newSlice []string
				for _, sv := range v {
					newSlice = append(newSlice, replacer.Replace(sv))
				}
				newMap[replacer.Replace(k)] = newSlice
			}
			f.Set(reflect.ValueOf(newMap))
		case map[string]*Step:
			newMap := map[string]*Step{}
			for k, v := range raw.(map[string]*Step) {
				substitute(reflect.ValueOf(v).Elem(), replacer)
				newMap[replacer.Replace(k)] = v
			}
			f.Set(reflect.ValueOf(newMap))
		case *compute.Client, *storage.Client, context.Context, context.CancelFunc:
			// We specifically do not want to change fields with these types.
			continue
		default:
			if f.Kind() != reflect.Ptr {
				continue
			}
			switch e := f.Elem(); e.Kind() {
			case reflect.Slice:
				// Iterate through then run them back through substitute.
				for i := 0; i < e.Len(); i++ {
					substitute(e.Index(i), replacer)
				}
			case reflect.Struct:
				// Run structs right back through substitute.
				substitute(e, replacer)
			}
		}
	}
}

func xor(x, y bool) bool {
	return x != y
}

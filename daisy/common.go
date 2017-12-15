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
	"math/rand"
	"os"
	"os/user"
	"reflect"
	"regexp"
	"strings"
	"time"
)

func getUser() string {
	if cu, err := user.Current(); err == nil {
		return cu.Username
	}
	if hn, err := os.Hostname(); err == nil {
		return hn
	}
	return "unknown"
}

func namedSubexp(re *regexp.Regexp, s string) map[string]string {
	match := re.FindStringSubmatch(s)
	if match == nil {
		return nil
	}
	result := make(map[string]string)
	l := len(match)
	for i, name := range re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		result[name] = ""
		if i < l {
			result[name] = match[i]
		}
	}
	return result
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
	traverseData(v, func(val reflect.Value) dErr {
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
func traverseData(v reflect.Value, f func(reflect.Value) dErr) dErr {
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

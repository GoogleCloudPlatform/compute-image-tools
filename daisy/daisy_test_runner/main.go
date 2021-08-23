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

// daisy_test_runner is a tool for testing using Daisy workflows.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/google/uuid"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

const (
	defaultParallelCount = 5
	timeFormat           = time.RFC3339
)

var (
	oauth         = flag.String("oauth", "", "path to oauth json file")
	projects      = flag.String("projects", "", "comma separated list of projects that can be used for tests, overrides setting in template")
	zone          = flag.String("zone", "", "zone to use for tests, overrides setting in template")
	print         = flag.Bool("print", false, "print out the parsed test cases for debugging")
	printTemplate = flag.Bool("print_template", false, "print out the parsed test template for debugging")
	validate      = flag.Bool("validate", false, "validate all the test cases and exit")
	ce            = flag.String("compute_endpoint_override", "", "API endpoint to override default, will override ComputeEndpoint in template")
	filter        = flag.String("filter", "", "test name filter")
	outPath       = flag.String("out_path", "junit.xml", "junit xml path")
	parallelCount = flag.Int("parallel_count", 0, "TestParallelCount")

	funcMap = map[string]interface{}{
		"randItem": randItem,
		"mkSlice":  mkSlice,
		"mkMap":    mkMap,
		"split":    strings.Split,
		"add":      func(i, a int) int { return i + a },
	}

	testTemplate = template.New("testTemplate").Option("missingkey=zero").Funcs(funcMap)
)

func randItem(args []string) string {
	rand.Seed(time.Now().UnixNano())
	return args[rand.Intn(len(args))]
}

func mkSlice(args ...string) []string {
	return args
}

func mkMap(args ...string) map[string]string {
	m := make(map[string]string)
	for _, arg := range args {
		split := strings.Split(arg, ":")
		if len(split) != 2 {
			continue
		}
		m[split[0]] = split[1]
	}

	return m
}

// A TestSuite describes the tests to run.
type TestSuite struct {
	// Name for this set of tests.
	Name string
	// Project pool to use.
	Projects []string
	// Default zone to use.
	Zone string
	// The test cases to run.
	Tests map[string]*TestCase
	// How many tests to run in parallel.
	TestParallelCount int

	OAuthPath       string
	ComputeEndpoint string
}

// A TestCase is a single test to run.
type TestCase struct {
	// Path to the daisy workflow to use.
	// Each test workflow should manage its own resource creation and cleanup.
	Path   string
	w      *daisy.Workflow
	id     string
	logger *logger
	// Vars to pass to the daisy workflow.
	Vars map[string]string
	// Default timeout is 2 hours.
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	TestTimeout string
	timeout     time.Duration

	// Optional settings that will override those set in the workflow or TestTemplate.
	Zone            string
	OAuthPath       string
	ComputeEndpoint string

	// If set this test will be the only test allowed to run in the project.
	// This is required for any test that changes project level settings that may
	// impact other concurrent test runs.
	ProjectLock       bool
	CustomProjectLock string
}

type logger struct {
	buf bytes.Buffer
	mx  sync.Mutex
}

func (l *logger) WriteLogEntry(e *daisy.LogEntry) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.buf.WriteString(e.String())
}

func (l *logger) WriteSerialPortLogs(w *daisy.Workflow, instance string, buf bytes.Buffer) {
	return
}

func (l *logger) ReadSerialPortLogs() []string {
	return nil
}

func (l *logger) Flush() { return }

func createTestCase(ctx context.Context, testLogger *logger, path, project, zone, oauthPath, ce string, varMap map[string]string) (*daisy.Workflow, error) {
	w, err := daisy.NewFromFile(path)
	if err != nil {
		return nil, err
	}
	for k, v := range varMap {
		w.AddVar(k, v)
	}

	if oauthPath != "" {
		w.OAuthPath = oauthPath
	}

	if ce != "" {
		w.ComputeEndpoint = ce
	}

	if err := w.PopulateClients(ctx, []option.ClientOption{}); err != nil {
		return nil, err
	}

	w.Project = project
	w.Zone = zone
	w.DisableGCSLogging()
	w.DisableCloudLogging()
	w.DisableStdoutLogging()
	w.Logger = testLogger

	if len(w.Steps) == 0 {
		return nil, nil
	}
	return w, nil
}

func createTestSuite(ctx context.Context, path string, varMap map[string]string, regex *regexp.Regexp) (*TestSuite, error) {
	var t TestSuite

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	templ, err := testTemplate.Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	var buf bytes.Buffer
	if err := templ.Execute(&buf, varMap); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	if *printTemplate {
		fmt.Println(buf.String())
		return nil, nil
	}

	if err := json.Unmarshal(buf.Bytes(), &t); err != nil {
		return nil, daisy.JSONError(path, buf.Bytes(), err)
	}

	if *projects != "" {
		t.Projects = strings.Split(*projects, ",")
	}
	if len(t.Projects) == 0 {
		return nil, errors.New("no projects provided")
	}

	if *zone != "" {
		t.Zone = *zone
	}
	if *oauth != "" {
		t.OAuthPath = *oauth
	}
	if *ce != "" {
		t.ComputeEndpoint = *ce
	}
	if *parallelCount != 0 {
		t.TestParallelCount = *parallelCount
	}

	if t.TestParallelCount == 0 {
		t.TestParallelCount = defaultParallelCount
	}

	fmt.Printf("[TestRunner] Creating test cases for test suite %q\n", t.Name)

	for name, test := range t.Tests {
		test.id = uuid.New().String()
		if test.TestTimeout == "" {
			test.timeout = defaultTimeout
		} else {
			d, err := time.ParseDuration(test.TestTimeout)
			if err != nil {
				test.timeout = defaultTimeout
			} else {
				test.timeout = d
			}
		}

		if regex != nil && !regex.MatchString(name) {
			continue
		}

		fmt.Printf("  - Creating test case for %q\n", name)

		wfPath := filepath.Join(filepath.Dir(path), test.Path)
		for k, v := range test.Vars {
			varMap[k] = v
		}

		zone := t.Zone
		if test.Zone != "" {
			zone = test.Zone
		}
		oauthPath := t.OAuthPath
		if test.OAuthPath != "" {
			oauthPath = test.OAuthPath
		}
		computeEndpoint := t.ComputeEndpoint
		if test.ComputeEndpoint != "" {
			computeEndpoint = test.ComputeEndpoint
		}

		rand.Seed(time.Now().UnixNano())
		test.logger = &logger{}
		w, err := createTestCase(ctx, test.logger, wfPath, t.Projects[rand.Intn(len(t.Projects))], zone, oauthPath, computeEndpoint, varMap)
		if err != nil {
			return nil, err
		}
		test.w = w
	}

	return &t, nil
}

const (
	flgDefValue   = "flag generated for workflow variable"
	varFlagPrefix = "var:"
)

func addFlags(args []string) {
	for _, arg := range args {
		if len(arg) <= 1 || arg[0] != '-' {
			continue
		}

		name := arg[1:]
		if name[0] == '-' {
			name = name[1:]
		}

		if !strings.HasPrefix(name, varFlagPrefix) {
			continue
		}

		name = strings.SplitN(name, "=", 2)[0]

		if flag.Lookup(name) != nil {
			continue
		}

		flag.String(name, "", flgDefValue)
	}
}

func checkError(errors chan error) {
	select {
	case err := <-errors:
		fmt.Fprintln(os.Stderr, "\n[TestRunner] Errors in one or more test cases:")
		fmt.Fprintln(os.Stderr, "\n - ", err)
		for {
			select {
			case err := <-errors:
				fmt.Fprintln(os.Stderr, "\n - ", err)
				continue
			default:
				fmt.Fprintln(os.Stderr, "\n[TestRunner] Exiting with exit code 1")
				os.Exit(1)
			}
		}
	default:
		return
	}
}

type junitTestSuite struct {
	mx sync.Mutex

	XMLName  xml.Name `xml:"testsuite"`
	Name     string   `xml:"name,attr"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Errors   int      `xml:"errors,attr"`
	Disabled int      `xml:"disabled,attr"`
	Skipped  int      `xml:"skipped,attr"`
	Time     float64  `xml:"time,attr"`

	TestCase []*junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Classname string        `xml:"classname,attr"`
	ID        string        `xml:"id,attr"`
	Name      string        `xml:"name,attr"`
	Time      float64       `xml:"time,attr"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

type junitFailure struct {
	FailMessage string `xml:",chardata"`
	FailType    string `xml:"type,attr"`
}

type test struct {
	name     string
	testCase *TestCase
}

func getCommonInstanceMetadata(client daisyCompute.Client, project string) (*compute.Metadata, error) {
	proj, err := client.GetProject(project)
	if err != nil {
		return nil, fmt.Errorf("error getting project: %v", err)
	}

	return proj.CommonInstanceMetadata, nil
}

func delItem(items []*compute.MetadataItems, i int) []*compute.MetadataItems {
	// Delete the element.
	// https://github.com/golang/go/wiki/SliceTricks
	copy(items[i:], items[i+1:])
	items[len(items)-1] = nil
	return items[:len(items)-1]
}

func isExpired(val string) bool {
	t, err := time.Parse(timeFormat, val)
	if err != nil {
		return false
	}
	return time.Now().After(t)
}

const (
	writeLock      = "TestWriteLock-"
	readLock       = "TestReadLock-"
	defaultTimeout = 2 * time.Hour
)

func waitLock(client daisyCompute.Client, project string, prefix ...string) (*compute.Metadata, error) {
	var md *compute.Metadata
	var err error
Loop:
	for {
		md, err = getCommonInstanceMetadata(client, project)
		if err != nil {
			return nil, err
		}

		for i, mdi := range md.Items {
			if mdi != nil {
				for _, p := range prefix {
					if strings.HasPrefix(mdi.Key, p) {
						if isExpired(*mdi.Value) {
							md.Items = delItem(md.Items, i)
						} else {
							r := rand.Intn(10) + 5
							time.Sleep(time.Duration(r) * time.Second)
							continue Loop
						}
					}
				}
			}
		}
		return md, nil
	}
}

func projectReadLock(client daisyCompute.Client, project, key string, timeout time.Duration) (string, error) {
	md, err := waitLock(client, project, writeLock)
	if err != nil {
		return "", err
	}

	lock := readLock + key
	val := time.Now().Add(timeout).Format(timeFormat)
	md.Items = append(md.Items, &compute.MetadataItems{Key: lock, Value: &val})
	if err := client.SetCommonInstanceMetadata(project, md); err != nil {
		return "", err
	}
	return lock, nil
}

func customProjectWriteLock(client daisyCompute.Client, project, custom, key string, timeout time.Duration) (string, error) {
	customLock := readLock + custom
	md, err := waitLock(client, project, writeLock, customLock)
	if err != nil {
		return "", err
	}

	lock := customLock + key
	val := time.Now().Add(timeout).Format(timeFormat)
	md.Items = append(md.Items, &compute.MetadataItems{Key: lock, Value: &val})
	if err := client.SetCommonInstanceMetadata(project, md); err != nil {
		return "", err
	}

	return lock, nil
}

func projectWriteLock(client daisyCompute.Client, project, key string, timeout time.Duration) (string, error) {
	md, err := waitLock(client, project, writeLock)
	if err != nil {
		return "", err
	}

	// This means the project has no current write locks, set the write lock
	// now and then wait till all current read locks are gone.
	lock := writeLock + key
	val := time.Now().Add(timeout).Format(timeFormat)
	md.Items = append(md.Items, &compute.MetadataItems{Key: lock, Value: &val})
	if err := client.SetCommonInstanceMetadata(project, md); err != nil {
		return "", err
	}

	if _, err := waitLock(client, project, readLock); err != nil {
		// Attempt to unlock.
		projectUnlock(client, project, lock)
		return "", err
	}
	return lock, nil
}

func projectUnlock(client daisyCompute.Client, project, lock string) error {
	md, err := getCommonInstanceMetadata(client, project)
	if err != nil {
		return err
	}

	for i, mdi := range md.Items {
		if mdi != nil && lock == mdi.Key {
			md.Items = delItem(md.Items, i)
		}
	}

	return client.SetCommonInstanceMetadata(project, md)
}

var allowedChars = regexp.MustCompile("[^-_a-zA-Z0-9]+")

func runTestCase(ctx context.Context, test *test, tc *junitTestCase, errors chan error, retries int) {
	if err := test.testCase.w.PopulateClients(ctx, []option.ClientOption{}); err != nil {
		errors <- fmt.Errorf("%s: %v", tc.Name, err)
		tc.Failure = &junitFailure{FailMessage: err.Error(), FailType: "Error"}
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			fmt.Printf("\nCtrl-C caught, sending cancel signal to %q...\n", test.name)
			test.testCase.w.CancelWorkflow()
			err := fmt.Errorf("test case %q was canceled", test.name)
			errors <- err
			tc.Failure = &junitFailure{FailMessage: err.Error(), FailType: "Canceled"}
		case <-test.testCase.w.Cancel:
		}
	}()

	project := test.testCase.w.Project
	client := test.testCase.w.ComputeClient
	key := test.testCase.w.ID()
	var lock string
	var err error
	if test.testCase.CustomProjectLock != "" {
		for i := 0; i < retries; i++ {
			lock, err = customProjectWriteLock(client, project, allowedChars.ReplaceAllString(test.testCase.CustomProjectLock, "_"), key, test.testCase.timeout)
			if err == nil {
				break
			}
		}
		if err != nil {
			errors <- err
			return
		}
	} else if test.testCase.ProjectLock {
		for i := 0; i < retries; i++ {
			lock, err = projectWriteLock(client, project, key, test.testCase.timeout)
			if err == nil {
				break
			}
		}
		if err != nil {
			errors <- err
			return
		}
	} else {
		for i := 0; i < retries; i++ {
			lock, err = projectReadLock(client, project, key, test.testCase.timeout)
			if err == nil {
				break
			}
		}
		if err != nil {
			errors <- err
			return
		}
	}
	defer func() {
		for i := 0; i < retries; i++ {
			err := projectUnlock(client, project, lock)
			if err == nil {
				break
			}
		}
		if err != nil {
			fmt.Printf("[TestRunner] Test %q: Error unlocking project: %v\n", test.name, err)
		}
	}()

	select {
	case <-test.testCase.w.Cancel:
		return
	default:
	}

	start := time.Now()
	fmt.Printf("[TestRunner] Running test case %q\n", tc.Name)
	if err := test.testCase.w.Run(ctx); err != nil {
		errors <- fmt.Errorf("%s: %v", tc.Name, err)
		tc.Failure = &junitFailure{FailMessage: err.Error(), FailType: "Failure"}
	}
	tc.Time = time.Since(start).Seconds()
	tc.SystemOut = test.testCase.logger.buf.String()
	fmt.Printf("[TestRunner] Test case %q finished\n", tc.Name)
}

func main() {
	addFlags(os.Args[1:])
	flag.Parse()

	varMap := map[string]string{}
	flag.Visit(func(flg *flag.Flag) {
		if strings.HasPrefix(flg.Name, varFlagPrefix) {
			varMap[strings.TrimPrefix(flg.Name, varFlagPrefix)] = flg.Value.String()
		}
	})

	if len(flag.Args()) == 0 {
		fmt.Println("Not enough args, first arg needs to be the path to a test template.")
		os.Exit(1)
	}
	var regex *regexp.Regexp
	if *filter != "" {
		var err error
		regex, err = regexp.Compile(*filter)
		if err != nil {
			fmt.Println("-filter flag not valid:", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()

	ts, err := createTestSuite(ctx, flag.Arg(0), varMap, regex)
	if err != nil {
		log.Fatalln("test case creation error:", err)
	}
	if ts == nil {
		return
	}

	errors := make(chan error, len(ts.Tests))
	// Retry failed locks 2x as many tests in the test case.
	retries := len(ts.Tests) * 2
	if len(ts.Tests) == 0 {
		fmt.Println("[TestRunner] Nothing to do")
		return
	}

	if *print {
		for n, t := range ts.Tests {
			if t.w == nil {
				continue
			}
			fmt.Printf("[TestRunner] Printing test case %q\n", n)
			t.w.Print(ctx)
		}
		checkError(errors)
		return
	}

	if *validate {
		for n, t := range ts.Tests {
			if t.w == nil {
				continue
			}
			fmt.Printf("[TestRunner] Validating test case %q\n", n)
			if err := t.w.Validate(ctx); err != nil {
				errors <- fmt.Errorf("Error validating test case %s: %v", n, err)
			}
		}
		checkError(errors)
		return
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0770); err != nil {
		log.Fatal(err)
	}

	junit := &junitTestSuite{Name: ts.Name, Tests: len(ts.Tests)}
	tests := make(chan *test, len(ts.Tests))
	var wg sync.WaitGroup
	for i := 0; i < ts.TestParallelCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for test := range tests {
				tc := &junitTestCase{Classname: ts.Name, ID: test.testCase.id, Name: test.name}
				junit.mx.Lock()
				junit.TestCase = append(junit.TestCase, tc)
				junit.mx.Unlock()

				if test.testCase.w == nil {
					junit.mx.Lock()
					junit.Skipped++
					junit.mx.Unlock()
					tc.Skipped = &junitSkipped{Message: fmt.Sprintf("Test does not match filter: %q", regex.String())}
					continue
				}

				runTestCase(ctx, test, tc, errors, retries)
			}
		}()
	}

	start := time.Now()
	for n, t := range ts.Tests {
		tests <- &test{name: n, testCase: t}
	}
	close(tests)
	wg.Wait()

	fmt.Printf("[TestRunner] Creating junit xml file: %q\n", *outPath)
	junit.Time = time.Since(start).Seconds()
	d, err := xml.MarshalIndent(junit, "  ", "   ")
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(*outPath, d, 0644); err != nil {
		log.Fatal(err)
	}

	checkError(errors)
	fmt.Println("[TestRunner] All test cases completed successfully.")
}

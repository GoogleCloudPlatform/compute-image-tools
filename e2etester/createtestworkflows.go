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

// Package e2etester is a library for VM-based e2e tests using Daisy workflows.
package e2etester

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

// Config is a config.
type Config struct {
	Image    string
	Metadata map[string]string
}

var configs map[string]Config
var dependencies map[string]string

// RunTogether runs two tests together by including the dependee into the
// dependent WF.
func RunTogether(dependent, dependee string) {
	if dependencies == nil {
		dependencies = make(map[string]string)
	}
	dependencies[dependent] = dependee
}

// AddTestConfig adds a Config object for a specified test.
func AddTestConfig(test string, config Config) {
	if configs == nil {
		configs = make(map[string]Config)
	}
	configs[test] = config
}

var md map[string]string

// GetMetadata gets instance attributes from metadata.
func GetMetadata() (map[string]string, error) {
	if md != nil {
		return md, nil
	}
	metadataURL := "http://metadata.google.internal/computeMetadata/v1/instance/attributes/?recursive=true&alt=json"
	client := &http.Client{}
	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building HTTP request for metadata: %s", err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting metadata: %s", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error getting metadata: %s", err)
	}
	err = json.Unmarshal(body, &md)
	if err != nil {
		return nil, fmt.Errorf("error parsing metadata: %s", err)
	}
	return md, nil
}

func runLocalTest(tests interface{}) error {
	md, err := GetMetadata()
	if err != nil {
		return err
	}
	testname, ok := md["TestName"]
	if !ok {
		return fmt.Errorf("TestName not in metadata")
	}

	typ := reflect.TypeOf(tests)
	m, ok := typ.MethodByName(testname)
	if !ok {
		return fmt.Errorf("could not locate specified test %q", testname)
	}
	if m.Type.NumIn() != 1 {
		return fmt.Errorf("func %q should only have one input", testname)
	}
	if m.Type.In(0) != typ {
		return fmt.Errorf("func %q should take %v type as first arg, but takes %v", testname, typ, m.Type.In(0))
	}
	if m.Type.NumOut() != 1 {
		return fmt.Errorf("func %q should only have one output", testname)
	}
	errType := reflect.TypeOf((*error)(nil)).Elem() // create pointer to nil error then resolve.
	if !m.Type.Out(0).Implements(errType) {
		return fmt.Errorf("func %q should return %v/%T, but returns %v/%T", testname, errType, errType, m.Type.Out(0), m.Type.Out(0))
	}

	res := m.Func.Call([]reflect.Value{reflect.ValueOf(tests)})
	ierr := res[0].Interface()
	if ierr != nil {
		return ierr.(error)
	}
	return nil
}

func createTestWorkflows(tests interface{}, images []string, regex *regexp.Regexp) (map[string]*daisy.Workflow, error) {
	// Create base test workflows.
	workflows := make(map[string]*daisy.Workflow)
	typ := reflect.TypeOf(tests)
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if regex != nil && !regex.MatchString(m.Name) {
			continue
		}
		for _, image := range images {
			wf, err := createVMWorkflow(m.Name, image)
			if err != nil {
				return nil, fmt.Errorf("failed to create workflow for %q", m.Name)
			}
			nameAndImage := fmt.Sprintf("%s [%s]", m.Name, image)
			workflows[nameAndImage] = wf
		}
	}

	// Include dependency tests as workflows in their dependents.
	for dependentTest, dependeeTest := range dependencies {
		for _, image := range images {
			dependentName := fmt.Sprintf("%s [%s]", dependentTest, image)
			dependentwf, ok := workflows[dependentName]
			if !ok {
				return nil, fmt.Errorf("%q in dependency map, but not found", dependentName)
			}
			dependeeName := fmt.Sprintf("%s [%s]", dependeeTest, image)
			dependeewf, ok := workflows[dependeeName]
			if !ok {
				return nil, fmt.Errorf("%q in dependency map, but not found", dependeeName)
			}

			// Make dependee instance name available to the dependent.
			dependentCreateStep := dependentwf.Steps[dependentTest]                                     // create instance step is named after test.
			dependentInstance := (*dependentCreateStep.CreateInstances)[0]                              // pull out the instance config.
			dependentInstance.Metadata[dependeeTest] = fmt.Sprintf("test-instance-%s", dependeewf.ID()) // add an entry to dependent metadata.

			step, err := dependentwf.NewStep("include")
			if err != nil {
				return nil, fmt.Errorf("Failed to create workflow step: %v", err)
			}
			step.SubWorkflow = &daisy.SubWorkflow{Workflow: dependeewf}
			delete(workflows, dependeeName) // dependee is no longer a top-level WF.

			// dependent create-instance step run after the dependee workflow.
			dependentwf.AddDependency(dependentwf.Steps[dependentTest], dependentwf.Steps["include"])
		}
	}

	// Add wait signals to top-level WFs.
	for _, wf := range workflows {
		waitStepName := fmt.Sprintf("wait-instance-%s", wf.Name)
		step, err := wf.NewStep(waitStepName)
		if err != nil {
			return nil, fmt.Errorf("Failed to create workflow step: %v", err)
		}
		instanceSignal := &daisy.InstanceSignal{
			Name: wf.Name,
			SerialOutput: &daisy.SerialOutput{
				Port:         1,
				SuccessMatch: "TEST-SUCCESS",
				FailureMatch: daisy.FailureMatches{"TEST-FAILURE"},
			},
		}
		step.WaitForInstancesSignal = &daisy.WaitForInstancesSignal{instanceSignal}
		step.Timeout = "2m"
		wf.AddDependency(wf.Steps[waitStepName], wf.Steps[wf.Name])
	}

	return workflows, nil
}

func createVMWorkflow(name, image string) (*daisy.Workflow, error) {
	wf := daisy.New()
	wf.Name = name
	wf.Sources = map[string]string{"startup": os.Args[0]}

	config := configs[name]
	if config.Image != "" {
		image = config.Image
	}
	diskStepName := fmt.Sprintf("create-disk-%s", name)
	bootdisk := &daisy.Disk{
		Disk: compute.Disk{
			SourceImage: image,
			Name:        diskStepName,
		},
	}

	step, err := wf.NewStep(diskStepName)
	if err != nil {
		return nil, err
	}
	step.CreateDisks = &daisy.CreateDisks{bootdisk}

	md := config.Metadata
	if md == nil {
		md = make(map[string]string)
	}
	md["TestName"] = name
	instance := &daisy.Instance{
		StartupScript: "startup",
		Metadata:      md,
		Resource:      daisy.Resource{RealName: fmt.Sprintf("test-instance-%s", wf.ID())},
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/compute",
		},
	}
	instance.Disks = append(instance.Disks, &compute.AttachedDisk{Source: diskStepName})
	step, err = wf.NewStep(name)
	if err != nil {
		return nil, err
	}
	step.CreateInstances = &daisy.CreateInstances{instance}

	wf.AddDependency(wf.Steps[name], wf.Steps[diskStepName])
	return wf, nil
}

//  Copyright 2021 Google Inc. All Rights Reserved.
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

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// CheckWorkflow allows a test to check the fields on the daisy workflow associated with a DaisyWorker.
func CheckWorkflow(worker DaisyWorker, check func(wf *daisy.Workflow)) {
	check(worker.(*defaultDaisyWorker).wf)
}

// CheckEnvironment allows a test to check the fields on the EnvironmentSettings associated with a DaisyWorker.
func CheckEnvironment(worker DaisyWorker, check func(env EnvironmentSettings)) {
	check(worker.(*defaultDaisyWorker).env)
}

// CheckResourceLabeler allows a test to check the fields on the resource labeler associated with a DaisyWorker.
func CheckResourceLabeler(worker DaisyWorker, check func(rl *ResourceLabeler)) {
	for _, hook := range worker.(*defaultDaisyWorker).hooks {
		switch hook.(type) {
		case *ResourceLabeler:
			check(hook.(*ResourceLabeler))
			return
		}
	}
	panic("Didn't find resource labeler")
}

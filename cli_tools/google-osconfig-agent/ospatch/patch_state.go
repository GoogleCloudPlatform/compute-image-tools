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

package ospatch

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
)

var liveState state

const pastJobsNum = 10

type state struct {
	PatchRuns []*patchRun
	PastJobs  []string

	sync.RWMutex `json:"-"`
}

func (s *state) jobComplete(job string) {
	s.Lock()
	defer s.Unlock()
	s.PastJobs = append(s.PastJobs, job)
	if len(s.PastJobs) > pastJobsNum {
		s.PastJobs = s.PastJobs[len(s.PastJobs)-pastJobsNum : len(s.PastJobs)]
	}
}

func (s *state) addPatchRun(pr *patchRun) {
	s.Lock()
	defer s.Unlock()
	(*s).PatchRuns = append((*s).PatchRuns, pr)
}

func (s *state) removePatchRun(pr *patchRun) {
	s.Lock()
	defer s.Unlock()

	for i, r := range s.PatchRuns {
		if r.Job.PatchJob == pr.Job.PatchJob {
			copy(s.PatchRuns[i:], s.PatchRuns[i+1:])
			s.PatchRuns[len(s.PatchRuns)-1] = nil
			s.PatchRuns = s.PatchRuns[:len(s.PatchRuns)-1]
			return
		}
	}
}

func (s *state) alreadyAckedJob(job string) bool {
	s.RLock()
	defer s.RUnlock()
	for _, r := range s.PatchRuns {
		if r.Job.PatchJob == job {
			return true
		}
	}
	for _, j := range s.PastJobs {
		if j == job {
			return true
		}
	}
	return false
}

func (s *state) save(path string) error {
	s.RLock()
	defer s.RUnlock()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if s == nil {
		return writeFile(path, []byte("{}"))
	}

	d, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return writeFile(path, d)
}

func saveState() error {
	return liveState.save(config.PatchStateFile())
}

func loadState(path string) error {
	liveState.Lock()
	defer liveState.Unlock()

	d, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(d, &liveState)
}

func writeFile(path string, data []byte) error {
	// Write state to a temporary file first.
	tmp, err := ioutil.TempFile(filepath.Dir(path), "")
	if err != nil {
		return err
	}
	newStateFile := tmp.Name()

	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// Move the new temp file to the live path.
	return os.Rename(newStateFile, path)
}

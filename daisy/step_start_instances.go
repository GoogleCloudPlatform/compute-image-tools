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

package daisy

import (
	"context"
	"sync"
)

// StartInstances start GCE instances.
type StartInstances struct {
	Instances []string `json:",omitempty"`
}

func (st *StartInstances) populate(ctx context.Context, s *Step) dErr {
	for i, instance := range st.Instances {
		if instanceURLRgx.MatchString(instance) {
			st.Instances[i] = extendPartialURL(instance, s.w.Project)
		}
	}
	return nil
}

func (st *StartInstances) validate(ctx context.Context, s *Step) dErr {
	// Instance checking.
	for _, i := range st.Instances {
		if _, err := s.w.instances.regUse(i, s); err != nil {
			return err
		}
	}
	return nil
}

func (st *StartInstances) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)

	for _, i := range st.Instances {
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			w.LogStepInfo(s.name, "StartInstances", "Starting instance %q.", i)
			if err := w.instances.start(i); err != nil {
				e <- err
			}
		}(i)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		return nil
	}
}

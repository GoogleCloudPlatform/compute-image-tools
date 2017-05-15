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
	"strings"
	"sync"
	"time"
)

const defaultInterval = "5s"

// WaitForInstancesSignal is a Daisy WaitForInstancesSignal workflow step.
type WaitForInstancesSignal []InstanceSignal

// This step will not complete until a line in the serial output matches
// SuccessMatch or FailureMatch. A match with FailureMatch will cause the step
// to fail.
type SerialOutput struct {
	Port         int64
	SuccessMatch string
	FailureMatch string
}

// InstanceSignal waits for a signal from an instance.
type InstanceSignal struct {
	// Instance name to wait for.
	Name string
	// Interval to check for signal (default is 5s).
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	Interval string
	interval time.Duration
	// Wait for the instance to stop.
	Stopped bool
	// Wait for a string match in the serial output.
	SerialOutput *SerialOutput
}

func waitForSerialOutput(w *Workflow, name string, port int64, success, failure string, interval time.Duration) error {
	w.logger.Printf("WaitForInstancesSignal: watching serial port %d, SuccessMatch: %q, FailureMatch: %q.", port, success, failure)
	var start int64
	tick := time.Tick(interval)
	for {
		select {
		case <-w.Cancel:
			return nil
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			if err != nil {
				status, sErr := w.ComputeClient.InstanceStatus(w.Project, w.Zone, name)
				if sErr == nil && (status == "TERMINATED" || status == "STOPPING") {
					w.logger.Printf("WaitForInstancesSignal: instance %q stopped, not waiting for serial output.", name)
					return nil
				}
				return fmt.Errorf("WaitForInstancesSignal: instance %q: error getting serial port: %v", name, err)
			}
			start = resp.Next
			if success != "" && strings.Contains(resp.Contents, success) {
				w.logger.Printf("WaitForInstancesSignal: SuccessMatch instance %q: %q in %q", name, failure, resp.Contents)
				return nil
			}
			if failure != "" && strings.Contains(resp.Contents, failure) {
				return fmt.Errorf("WaitForInstancesSignal: FailureMatch instance %q: %q in %q", name, failure, resp.Contents)
			}
		}
	}
}

func (s *WaitForInstancesSignal) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)

	for _, is := range *s {
		wg.Add(1)
		go func(is InstanceSignal) {
			defer wg.Done()
			i, ok := w.instanceRefs.get(is.Name)
			if !ok {
				e <- fmt.Errorf("unresolved instance %q", is.Name)
				return
			}
			if is.Stopped {
				w.logger.Printf("WaitForInstancesSignal: waiting for instance %q to stop.", i.real)
				if err := w.ComputeClient.WaitForInstanceStopped(w.Project, w.Zone, i.real, is.interval); err != nil {
					e <- err
					return
				}
				w.logger.Printf("WaitForInstancesSignal: instance %q stopped.", i.real)
				return
			}
			if is.SerialOutput != nil {
				if err := waitForSerialOutput(w, i.real, is.SerialOutput.Port, is.SerialOutput.SuccessMatch, is.SerialOutput.FailureMatch, is.interval); err != nil {
					e <- err
					return
				}
			}
		}(is)
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

func (s *WaitForInstancesSignal) validate(w *Workflow) error {
	// Instance checking.
	for _, i := range *s {
		if !instanceValid(w, i.Name) {
			return fmt.Errorf("cannot wait for instance signal. Instance not found: %q", i.Name)
		}
		if i.SerialOutput != nil {
			if i.SerialOutput.Port == 0 {
				return fmt.Errorf("%q: cannot wait for instance signal via SerialOutput, no Port given", i.Name)
			}
			if i.SerialOutput.SuccessMatch == "" && i.SerialOutput.FailureMatch == "" {
				return fmt.Errorf("%q: cannot wait for instance signal via SerialOutput, no SuccessMatch or FailureMatch given", i.Name)
			}
		}
	}
	return nil
}

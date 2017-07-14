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
	"strings"
	"sync"
	"time"

	"google.golang.org/api/googleapi"
)

const defaultInterval = "5s"

// WaitForInstancesSignal is a Daisy WaitForInstancesSignal workflow step.
type WaitForInstancesSignal []*InstanceSignal

// SerialOutput describes text signal strings that will be written to the serial
// port.
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
	msg := fmt.Sprintf("WaitForInstancesSignal: watching serial port %d", port)
	if success != "" {
		msg += fmt.Sprintf(", SuccessMatch: %q", success)
	}
	if failure != "" {
		msg += fmt.Sprintf(", FailureMatch: %q", failure)
	}
	w.logger.Printf(msg + ".")
	var start int64
	var errs int
	tick := time.Tick(interval)
	for {
		select {
		case <-w.Cancel:
			return nil
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			// Retry up to 3 times in a row on a 5xx error.
			if apiErr, ok := err.(*googleapi.Error); ok && errs < 3 && (apiErr.Code >= 500 && apiErr.Code <= 599) {
				errs++
				continue
			}
			if err != nil {
				status, sErr := w.ComputeClient.InstanceStatus(w.Project, w.Zone, name)
				if sErr == nil && (status == "TERMINATED" || status == "STOPPING" || status == "STOPPED") {
					w.logger.Printf("WaitForInstancesSignal: instance %q stopped, not waiting for serial output.", name)
					return nil
				}
				return fmt.Errorf("WaitForInstancesSignal: instance %q: error getting serial port: %v, error getting status: %v", name, err, sErr)
			}
			start = resp.Next
			if success != "" && strings.Contains(resp.Contents, success) {
				w.logger.Printf("WaitForInstancesSignal: SuccessMatch found for instance %q", name)
				return nil
			}
			if failure != "" && strings.Contains(resp.Contents, failure) {
				return fmt.Errorf("WaitForInstancesSignal: FailureMatch found for instance %q", name)
			}
			errs = 0
		}
	}
}

func (w *WaitForInstancesSignal) populate(ctx context.Context, s *Step) error {
	for _, ws := range *w {
		if ws.Interval == "" {
			ws.Interval = defaultInterval
		}
		var err error
		ws.interval, err = time.ParseDuration(ws.Interval)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WaitForInstancesSignal) run(ctx context.Context, s *Step) error {
	var wg sync.WaitGroup
	e := make(chan error)

	for _, is := range *w {
		wg.Add(1)
		go func(is *InstanceSignal) {
			defer wg.Done()
			i, ok := instances[s.w].get(is.Name)
			if !ok {
				e <- fmt.Errorf("unresolved instance %q", is.Name)
				return
			}
			serialSig := make(chan struct{})
			stoppedSig := make(chan struct{})
			if is.Stopped {
				go func() {
					s.w.logger.Printf("WaitForInstancesSignal: waiting for instance %q to stop.", i.real)
					if err := s.w.ComputeClient.WaitForInstanceStopped(s.w.Project, s.w.Zone, i.real, is.interval); err != nil {
						e <- err
					} else {
						s.w.logger.Printf("WaitForInstancesSignal: instance %q stopped.", i.real)
					}
					close(stoppedSig)
				}()
			}
			if is.SerialOutput != nil {
				go func() {
					if err := waitForSerialOutput(s.w, i.real, is.SerialOutput.Port, is.SerialOutput.SuccessMatch, is.SerialOutput.FailureMatch, is.interval); err != nil {
						e <- err
					}
					close(serialSig)
				}()
			}
			select {
			case <-serialSig:
				return
			case <-stoppedSig:
				return
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
	case <-s.w.Cancel:
		return nil
	}
}

func (w *WaitForInstancesSignal) validate(ctx context.Context, s *Step) error {
	// Instance checking.
	for _, i := range *w {
		if _, err := instances[s.w].registerUsage(i.Name, s); err != nil {
			return err
		}
		if i.interval == 0*time.Second {
			return fmt.Errorf("%q: cannot wait for instance signal, no interval given", i.Name)
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

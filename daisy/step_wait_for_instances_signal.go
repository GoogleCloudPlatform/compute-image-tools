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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultInterval = "10s"
)

// WaitForInstancesSignal is a Daisy WaitForInstancesSignal workflow step.
type WaitForInstancesSignal []*InstanceSignal

// SerialOutput describes text signal strings that will be written to the serial
// port.
// A StatusMatch will print out the matching line from the StatusMatch onward.
// This step will not complete until a line in the serial output matches
// SuccessMatch or FailureMatch. A match with FailureMatch will cause the step
// to fail.
type SerialOutput struct {
	Port         int64
	SuccessMatch string
	FailureMatch string
	StatusMatch  string
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

func waitForInstanceStopped(s *Step, project, zone, name string, interval time.Duration) dErr {
	w := s.w
	w.Logger.StepInfo(w, s.name, "WaitForInstancesSignal", "waiting for instance %q to stop.", name)
	tick := time.Tick(interval)
	for {
		select {
		case <-s.w.Cancel:
			return nil
		case <-tick:
			stopped, err := s.w.ComputeClient.InstanceStopped(project, zone, name)
			if err != nil {
				return typedErr(apiError, err)
			}
			if stopped {
				w.Logger.StepInfo(w, s.name, "WaitForInstancesSignal", "instance %q stopped.", name)
				return nil
			}
		}
	}
}

func waitForSerialOutput(s *Step, project, zone, name string, so *SerialOutput, interval time.Duration) dErr {
	w := s.w
	msg := fmt.Sprintf("instance %q: watching serial port %d", name, so.Port)
	if so.SuccessMatch != "" {
		msg += fmt.Sprintf(", SuccessMatch: %q", so.SuccessMatch)
	}
	if so.FailureMatch != "" {
		msg += fmt.Sprintf(", FailureMatch: %q", so.FailureMatch)
	}
	if so.StatusMatch != "" {
		msg += fmt.Sprintf(", StatusMatch: %q", so.StatusMatch)
	}
	w.Logger.StepInfo(w, s.name, "WaitForInstancesSignal", msg+".")
	var start int64
	var errs int
	tick := time.Tick(interval)
	for {
		select {
		case <-s.w.Cancel:
			return nil
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(project, zone, name, so.Port, start)
			if err != nil {
				status, sErr := w.ComputeClient.InstanceStatus(project, zone, name)
				if sErr != nil {
					err = fmt.Errorf("%v, error geting InstanceStatus: %v", err, sErr)
				} else {
					err = fmt.Errorf("%v, InstanceStatus: %q", err, status)
				}

				if status == "TERMINATED" || status == "STOPPED" {
					w.Logger.StepInfo(w, s.name, "WaitForInstancesSignal", "instance %q stopped, not waiting for serial output.", name)
					return nil
				}
				// Keep retrying until the instance is STOPPED.
				if status == "STOPPING" {
					continue
				}
				// Retry up to 3 times in a row on any error if we successfully got InstanceStatus.
				if errs < 3 {
					errs++
					continue
				}

				return errf("WaitForInstancesSignal: instance %q: error getting serial port: %v", name, err)
			}
			start = resp.Next
			for _, ln := range strings.Split(resp.Contents, "\n") {
				if so.StatusMatch != "" {
					if i := strings.Index(ln, so.StatusMatch); i != -1 {
						w.Logger.StepInfo(w, s.name, "WaitForInstancesSignal", "instance %q: StatusMatch found: %q", name, strings.TrimSpace(ln[i:]))
					}
				}
				if so.FailureMatch != "" {
					if i := strings.Index(ln, so.FailureMatch); i != -1 {
						return errf("WaitForInstancesSignal FailureMatch found for %q: %q", name, strings.TrimSpace(ln[i:]))
					}
				}
				if so.SuccessMatch != "" {
					if i := strings.Index(ln, so.SuccessMatch); i != -1 {
						w.Logger.StepInfo(w, s.name, "WaitForInstancesSignal", "instance %q: SuccessMatch found %q", name, strings.TrimSpace(ln[i:]))
						return nil
					}
				}
			}
			errs = 0
		}
	}
}

func (w *WaitForInstancesSignal) populate(ctx context.Context, s *Step) dErr {
	for _, ws := range *w {
		if ws.Interval == "" {
			ws.Interval = defaultInterval
		}
		var err error
		ws.interval, err = time.ParseDuration(ws.Interval)
		if err != nil {
			return newErr(err)
		}
	}
	return nil
}

func (w *WaitForInstancesSignal) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	e := make(chan dErr)

	for _, is := range *w {
		wg.Add(1)
		go func(is *InstanceSignal) {
			defer wg.Done()
			i, ok := s.w.instances.get(is.Name)
			if !ok {
				e <- errf("unresolved instance %q", is.Name)
				return
			}
			m := namedSubexp(instanceURLRgx, i.link)
			serialSig := make(chan struct{})
			stoppedSig := make(chan struct{})
			if is.Stopped {
				go func() {
					if err := waitForInstanceStopped(s, m["project"], m["zone"], m["instance"], is.interval); err != nil {
						e <- err
					}
					close(stoppedSig)
				}()
			}
			if is.SerialOutput != nil {
				go func() {
					if err := waitForSerialOutput(s, m["project"], m["zone"], m["instance"], is.SerialOutput, is.interval); err != nil {
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

func (w *WaitForInstancesSignal) validate(ctx context.Context, s *Step) dErr {
	// Instance checking.
	for _, i := range *w {
		if _, err := s.w.instances.regUse(i.Name, s); err != nil {
			return err
		}
		if i.interval == 0*time.Second {
			return errf("%q: cannot wait for instance signal, no interval given", i.Name)
		}
		if i.SerialOutput == nil && i.Stopped == false {
			return errf("%q: cannot wait for instance signal, nothing to wait for", i.Name)
		}
		if i.SerialOutput != nil {
			if i.SerialOutput.Port == 0 {
				return errf("%q: cannot wait for instance signal via SerialOutput, no Port given", i.Name)
			}
			if i.SerialOutput.SuccessMatch == "" && i.SerialOutput.FailureMatch == "" {
				return errf("%q: cannot wait for instance signal via SerialOutput, no SuccessMatch or FailureMatch given", i.Name)
			}
		}
	}
	return nil
}

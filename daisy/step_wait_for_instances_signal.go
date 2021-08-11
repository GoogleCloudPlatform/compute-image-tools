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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultInterval = "10s"
)

var (
	serialOutputValueRegex = regexp.MustCompile(".*<serial-output key:'(.*)' value:'(.*)'>")
)

// WaitForInstancesSignal is a Daisy WaitForInstancesSignal workflow step.
type WaitForInstancesSignal []*InstanceSignal

// WaitForAnyInstancesSignal is a Daisy WaitForAnyInstancesSignal workflow step.
type WaitForAnyInstancesSignal []*InstanceSignal

// FailureMatches is a list of matching failure strings.
type FailureMatches []string

// UnmarshalJSON unmarshals FailureMatches.
func (fms *FailureMatches) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*fms = []string{s}
		return nil
	}

	//not a string, try unmarshalling into an array. Need a temp type to avoid infinite loop.
	var ss []string
	if err := json.Unmarshal(b, &ss); err != nil {
		return err
	}

	*fms = FailureMatches(ss)
	return nil
}

// SerialOutput describes text signal strings that will be written to the serial
// port.
// A StatusMatch will print out the matching line from the StatusMatch onward.
// This step will not complete until a line in the serial output matches
// SuccessMatch or FailureMatch. A match with FailureMatch will cause the step to fail.
type SerialOutput struct {
	Port         int64          `json:",omitempty"`
	SuccessMatch string         `json:",omitempty"`
	FailureMatch FailureMatches `json:"failureMatch,omitempty"`
	StatusMatch  string         `json:",omitempty"`
}

// InstanceSignal waits for a signal from an instance.
type InstanceSignal struct {
	// Instance name to wait for.
	Name string
	// Interval to check for signal (default is 5s).
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	Interval string `json:",omitempty"`
	interval time.Duration
	// Wait for the instance to stop.
	Stopped bool `json:",omitempty"`
	// Wait for a string match in the serial output.
	SerialOutput *SerialOutput `json:",omitempty"`
}

func waitForInstanceStopped(s *Step, project, zone, name string, interval time.Duration) DError {
	w := s.w
	w.LogStepInfo(s.name, "WaitForInstancesSignal", "Waiting for instance %q to stop.", name)
	tick := time.Tick(interval)
	for {
		select {
		case <-s.w.Cancel:
			return nil
		case <-tick:
			stopped, err := s.w.ComputeClient.InstanceStopped(project, zone, name)
			if err != nil {
				return typedErr(apiError, "failed to check whether instance is stopped", err)
			}
			if stopped {
				w.LogStepInfo(s.name, "WaitForInstancesSignal", "Instance %q stopped.", name)
				return nil
			}
		}
	}
}

func waitForSerialOutput(s *Step, project, zone, name string, so *SerialOutput, interval time.Duration) DError {
	w := s.w
	msg := fmt.Sprintf("Instance %q: watching serial port %d", name, so.Port)
	if so.SuccessMatch != "" {
		msg += fmt.Sprintf(", SuccessMatch: %q", so.SuccessMatch)
	}
	if len(so.FailureMatch) > 0 {
		msg += fmt.Sprintf(", FailureMatch: %q (this is not an error)", so.FailureMatch)
	}
	if so.StatusMatch != "" {
		msg += fmt.Sprintf(", StatusMatch: %q", so.StatusMatch)
	}
	w.LogStepInfo(s.name, "WaitForInstancesSignal", msg+".")
	var start int64
	var errs int
	tailString := ""
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
					err = fmt.Errorf("%v, error getting InstanceStatus: %v", err, sErr)
				} else {
					err = fmt.Errorf("%v, InstanceStatus: %q", err, status)
				}

				// Wait until machine restarts to evaluate SerialOutput.
				if status == "TERMINATED" || status == "STOPPED" || status == "STOPPING" {
					continue
				}

				// Retry up to 3 times in a row on any error if we successfully got InstanceStatus.
				if errs < 3 {
					errs++
					continue
				}

				return Errf("WaitForInstancesSignal: instance %q: error getting serial port: %v", name, err)
			}
			start = resp.Next
			lines := strings.Split(resp.Contents, "\n")
			for i, ln := range lines {
				// If there is a unconsumed tail string from the previous block of content, concat it with the 1st line of the new block of content.
				if i == 0 && tailString != "" {
					ln = tailString + ln
					tailString = ""
				}

				// If the content is not ended with a "\n", we want to store the last line as tail string, so it can be concat with the next block of content.
				if i == len(lines)-1 && ln != "" {
					tailString = lines[len(lines)-1]
					break
				}

				if so.StatusMatch != "" {
					if i := strings.Index(ln, so.StatusMatch); i != -1 {
						w.LogStepInfo(s.name, "WaitForInstancesSignal", "Instance %q: StatusMatch found: %q", name, strings.TrimSpace(ln[i:]))
						extractOutputValue(w, ln)
					}
				}
				if len(so.FailureMatch) > 0 {
					for _, failureMatch := range so.FailureMatch {
						if i := strings.Index(ln, failureMatch); i != -1 {
							errMsg := strings.TrimSpace(ln[i:])
							format := "WaitForInstancesSignal FailureMatch found for %q: %q"
							return newErr(errMsg, fmt.Errorf(format, name, errMsg))
						}
					}
				}
				if so.SuccessMatch != "" {
					if i := strings.Index(ln, so.SuccessMatch); i != -1 {
						w.LogStepInfo(s.name, "WaitForInstancesSignal", "Instance %q: SuccessMatch found %q", name, strings.TrimSpace(ln[i:]))
						return nil
					}
				}
			}
			errs = 0
		}
	}
}

func extractOutputValue(w *Workflow, s string) {
	if matches := serialOutputValueRegex.FindStringSubmatch(s); matches != nil && len(matches) == 3 {
		for w.parent != nil {
			w = w.parent
		}
		w.AddSerialConsoleOutputValue(matches[1], matches[2])
	}
}

func (w *WaitForInstancesSignal) populate(ctx context.Context, s *Step) DError {
	is := (*[]*InstanceSignal)(w)
	return populateForWaitForInstancesSignal(is, "wait_for_instance_signal")
}

func (w *WaitForAnyInstancesSignal) populate(ctx context.Context, s *Step) DError {
	is := (*[]*InstanceSignal)(w)
	return populateForWaitForInstancesSignal(is, "wait_for_any_instance_signal")
}

func populateForWaitForInstancesSignal(w *[]*InstanceSignal, sn string) DError {
	for _, ws := range *w {
		if ws.Interval == "" {
			ws.Interval = defaultInterval
		}
		var err error
		ws.interval, err = time.ParseDuration(ws.Interval)
		if err != nil {
			return newErr(fmt.Sprintf("failed to parse duration for step %v", sn), err)
		}
	}
	return nil
}

func (w *WaitForInstancesSignal) run(ctx context.Context, s *Step) DError {
	is := (*[]*InstanceSignal)(w)
	return runForWaitForInstancesSignal(is, s, true)
}

func (w *WaitForAnyInstancesSignal) run(ctx context.Context, s *Step) DError {
	is := (*[]*InstanceSignal)(w)
	return runForWaitForInstancesSignal(is, s, false)
}

func runForWaitForInstancesSignal(w *[]*InstanceSignal, s *Step, waitAll bool) DError {
	var wg sync.WaitGroup
	e := make(chan DError)
	for _, is := range *w {
		wg.Add(1)
		go func(is *InstanceSignal) {
			defer wg.Done()
			i, ok := s.w.instances.get(is.Name)
			if !ok {
				e <- Errf("unresolved instance %q", is.Name)
				return
			}
			m := NamedSubexp(instanceURLRgx, i.link)
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
					if err := waitForSerialOutput(s, m["project"], m["zone"], m["instance"], is.SerialOutput, is.interval); err != nil || !waitAll {
						// send a signal to end other waiting instances
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

func (w *WaitForInstancesSignal) validate(ctx context.Context, s *Step) DError {
	is := (*[]*InstanceSignal)(w)
	return validateForWaitForInstancesSignal(is, s)
}

func (w *WaitForAnyInstancesSignal) validate(ctx context.Context, s *Step) DError {
	is := (*[]*InstanceSignal)(w)
	return validateForWaitForInstancesSignal(is, s)
}

func validateForWaitForInstancesSignal(w *[]*InstanceSignal, s *Step) DError {
	// Instance checking.
	for _, i := range *w {
		if _, err := s.w.instances.regUse(i.Name, s); err != nil {
			return err
		}
		if i.interval == 0*time.Second {
			return Errf("%q: cannot wait for instance signal, no interval given", i.Name)
		}
		if i.SerialOutput == nil && i.Stopped == false {
			return Errf("%q: cannot wait for instance signal, nothing to wait for", i.Name)
		}
		if i.SerialOutput != nil {
			if i.SerialOutput.Port == 0 {
				return Errf("%q: cannot wait for instance signal via SerialOutput, no Port given", i.Name)
			}
			if i.SerialOutput.SuccessMatch == "" && len(i.SerialOutput.FailureMatch) == 0 {
				return Errf("%q: cannot wait for instance signal via SerialOutput, no SuccessMatch or FailureMatch given", i.Name)
			}
		}
	}
	return nil
}

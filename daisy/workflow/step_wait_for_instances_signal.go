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
	"bytes"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"
)

// WaitForInstancesSignal is a Daisy WaitForInstancesSignal workflow step.
type WaitForInstancesSignal []InstanceSignal

// InstanceSignal waits for a signal from an instance.
type InstanceSignal struct {
	Name         string
	Stopped      bool
	SerialOutput struct {
		Port         int64
		SuccessMatch string
		FailureMatch string
	}
}

func waitForSerialOutput(w *Workflow, name string, port int64, success, failure string) error {
	logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", name, port))
	w.logger.Printf("WaitForInstancesSignal: streaming serial port %d output to gs://%s/%s.", w.bucket, port, logsObj)
	if success != "" || failure != "" {
		w.logger.Printf("WaitForInstancesSignal: watching serial port %d, SuccessMatch: %q, FailureMatch: %q.", name)
	}
	var start int64
	var buf bytes.Buffer
	for {
		tick := time.Tick(1 * time.Second)
		select {
		case <-w.Ctx.Done():
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			if err != nil {
				stopped, sErr := w.ComputeClient.InstanceStopped(w.Project, w.Zone, name)
				if (!stopped || sErr != nil) && success != "" {
					return fmt.Errorf("WaitForInstancesStopped: error getting serial port: %v", err)
				}
				w.logger.Printf("WaitForInstancesSignal: error getting serial port: %v", err)
				return nil
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(w.Ctx)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil {
				w.logger.Println("WaitForInstancesStopped: error writing log to GCS:", err)
			}
			if err := wc.Close(); err != nil {
				w.logger.Println("WaitForInstancesStopped: error writing log to GCS:", err)
			}
			if success != "" && strings.Contains(resp.Contents, success) {
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
				if err := w.ComputeClient.WaitForInstanceStopped(w.Project, w.Zone, i.real); err != nil {
					e <- err
					return
				}
			}
			if is.SerialOutput.Port != 0 {
				if err := waitForSerialOutput(w, i.real, is.SerialOutput.Port, is.SerialOutput.SuccessMatch, is.SerialOutput.FailureMatch); err != nil {
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
			return fmt.Errorf("cannot wait for instance stopped. Instance not found: %s", i)
		}
	}
	return nil
}

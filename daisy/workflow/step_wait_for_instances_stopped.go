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
	"sync"
	"time"
)

// WaitForInstancesStopped is a Daisy WaitForInstancesStopped workflow step.
type WaitForInstancesStopped []string

func streamSerial(w *Workflow, name string) {
	logsObj := path.Join(w.logsPath, name+"-serial.log")
	w.logger.Printf("WaitForInstancesStopped: streaming serial output to gs://%s/%s.", w.bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	for {
		tick := time.Tick(1 * time.Second)
		select {
		case <-w.Ctx.Done():
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, 1, start)
			if err != nil {
				stopped, sErr := w.ComputeClient.InstanceStopped(w.Project, w.Zone, name)
				if stopped || sErr != nil {
					w.logger.Println("WaitForInstancesStopped: error getting serial port:", err)
				}
				return
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(w.Ctx)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil {
				w.logger.Println("WaitForInstancesStopped: error writing log to GCS:", err)
				return
			}
			if err := wc.Close(); err != nil {
				w.logger.Println("WaitForInstancesStopped: error writing log to GCS:", err)
				return
			}
		}
	}
}

func (s *WaitForInstancesStopped) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)

	for _, name := range *s {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			i, ok := w.instanceRefs.get(name)
			if !ok {
				e <- fmt.Errorf("unresolved instance %q", name)
				return
			}
			// Stream serial port output to GCS.
			go streamSerial(w, i.real)
			w.logger.Printf("WaitForInstancesStopped: waiting for instance %q.", i.real)
			if err := w.ComputeClient.WaitForInstanceStopped(w.Project, w.Zone, i.real); err != nil {
				e <- err
			}
		}(name)
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

func (s *WaitForInstancesStopped) validate(w *Workflow) error {
	// Instance checking.
	for _, i := range *s {
		if !instanceValid(w, i) {
			return fmt.Errorf("cannot wait for instance stopped. Instance not found: %s", i)
		}
	}
	return nil
}

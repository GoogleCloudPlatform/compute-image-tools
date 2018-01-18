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
	"bytes"
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"google.golang.org/api/googleapi"
)

// CreateInstances is a Daisy CreateInstances workflow step.
type CreateInstances []*Instance

func logSerialOutput(ctx context.Context, w *Workflow, name string, port int64, interval time.Duration) {
	logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", name, port))
	w.Logger.Printf("CreateInstances: streaming instance %q serial port %d output to gs://%s/%s", name, port, w.bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	var errs int
	tick := time.Tick(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			if err != nil {
				// Instance was deleted by this workflow.
				if _, ok := w.instances.get(name); !ok {
					return
				}
				// Instance is stopped.
				stopped, sErr := w.ComputeClient.InstanceStopped(w.Project, w.Zone, name)
				if stopped && sErr == nil {
					return
				}
				w.Logger.Printf("CreateInstances: instance %q: error getting serial port: %v", name, err)
				return
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(ctx)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil {
				w.Logger.Printf("CreateInstances: instance %q: error writing log to GCS: %v", name, err)
				return
			}
			if err := wc.Close(); err != nil {
				if apiErr, ok := err.(*googleapi.Error); ok && (apiErr.Code >= 500 && apiErr.Code <= 599) {
					errs++
					continue
				}
				w.Logger.Printf("CreateInstances: instance %q: error saving log to GCS: %v", name, err)
				return
			}
			errs = 0
		}
	}
}

// populate preprocesses fields: Name, Project, Zone, Description, MachineType, NetworkInterfaces, Scopes, ServiceAccounts, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateInstances) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	for _, i := range *c {
		errs = addErrs(errs, i.populate(ctx, s))
	}
	return errs
}

func (c *CreateInstances) validate(ctx context.Context, s *Step) dErr {
	var errs dErr
	for _, i := range *c {
		errs = addErrs(errs, i.validate(ctx, s))
	}
	return errs
}

func (c *CreateInstances) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	eChan := make(chan dErr)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci *Instance) {
			defer wg.Done()

			for _, d := range ci.Disks {
				if diskRes, ok := w.disks.get(d.Source); ok {
					d.Source = diskRes.link
				}
			}

			w.Logger.Printf("CreateInstances: creating instance %q.", ci.Name)
			if err := w.ComputeClient.CreateInstance(ci.Project, ci.Zone, &ci.Instance); err != nil {
				eChan <- newErr(err)
				return
			}
			go logSerialOutput(ctx, w, ci.Name, 1, 3*time.Second)
		}(ci)
	}

	go func() {
		wg.Wait()
		eChan <- nil
	}()

	select {
	case err := <-eChan:
		return err
	case <-w.Cancel:
		// Wait so instances being created now can be deleted.
		wg.Wait()
		return nil
	}
}

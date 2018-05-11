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
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/logging"
)

// CreateInstances is a Daisy CreateInstances workflow step.
type CreateInstances []*Instance

func logSerialOutput(ctx context.Context, s *Step, name string, port int64, interval time.Duration) {
	w := s.w
	logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", name, port))
	w.Logger.StepInfo(w, s.name, "CreateInstances", "Streaming instance %q serial port %d output to https://storage.cloud.google.com/%s/%s", name, port, w.bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	var gcsErr bool
	tick := time.Tick(interval)

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			if err != nil {
				// Instance was deleted by this workflow.
				if _, ok := w.instances.get(name); !ok {
					break Loop
				}
				// Instance is stopped.
				stopped, sErr := w.ComputeClient.InstanceStopped(w.Project, w.Zone, name)
				if stopped && sErr == nil {
					break Loop
				}
				w.Logger.StepInfo(w, s.name, "CreateInstances", "Instance %q: error getting serial port: %v", name, err)
				break Loop
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(ctx)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil && !gcsErr {
				gcsErr = true
				w.Logger.StepInfo(w, s.name, "CreateInstances", "Instance %q: error writing log to GCS: %v", name, err)
				continue
			}
			if err := wc.Close(); err != nil && !gcsErr {
				gcsErr = true
				w.Logger.StepInfo(w, s.name, "CreateInstances", "Instance %q: error saving log to GCS: %v", name, err)
				continue
			}
		}
	}

	// Write the output to cloud logging only after instance has stopped.
	// Type assertion check is needed for tests not to panic.
	// Split if output is too long for log entry (100K max, we leave a 2K buffer).
	dl, ok := w.Logger.(*daisyLog)
	if ok {
		ss := strings.SplitAfter(buf.String(), "\n")
		var str string
		cl := func(str string) {
			dl.cloudLogger.Log(logging.Entry{
				Payload: map[string]string{
					"localTimestamp": time.Now().String(),
					"workflowName":   getAbsoluteName(w),
					"message":        fmt.Sprintf("Serial port output for instance %q", name),
					"serialPort1":    str,
					"type":           "Daisy",
				},
			})
		}
		for _, s := range ss {
			if len(str)+len(s) > 98*1024 {
				cl(str)
				str = s
			} else {
				str += s
			}
		}
		cl(str)
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
		go func(i *Instance) {
			defer wg.Done()

			for _, d := range i.Disks {
				if diskRes, ok := w.disks.get(d.Source); ok {
					d.Source = diskRes.link
				}
			}

			for _, n := range i.NetworkInterfaces {
				if netRes, ok := w.networks.get(n.Network); ok {
					n.Network = netRes.link
				}
			}

			w.Logger.StepInfo(w, s.name, "CreateInstances", "Creating instance %q.", i.Name)
			if err := w.ComputeClient.CreateInstance(i.Project, i.Zone, &i.Instance); err != nil {
				eChan <- newErr(err)
				return
			}
			go logSerialOutput(ctx, s, i.Name, 1, 3*time.Second)
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

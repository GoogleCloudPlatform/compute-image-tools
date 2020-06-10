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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/googleapi"
)

const (
	serialPortToArchive    = 1
	serialPortPollInterval = 3 * time.Second
)

// CreateInstances is a Daisy CreateInstances workflow step.
type CreateInstances struct {
	Instances     []*Instance
	InstancesBeta []*InstanceBeta
}

// UnmarshalJSON unmarshals Instance.
func (ci *CreateInstances) UnmarshalJSON(b []byte) error {
	var instancesBeta []*InstanceBeta
	if err := json.Unmarshal(b, &instancesBeta); err != nil {
		return err
	}
	ci.InstancesBeta = instancesBeta

	var instances []*Instance
	if err := json.Unmarshal(b, &instances); err != nil {
		return err
	}
	ci.Instances = instances

	return nil
}

func logSerialOutput(s *Step, name string, watcher *SerialOutputWatcher, wcProvider func() io.WriteCloser) {
	w := s.w
	w.stepWait.Add(1)
	// The buffered channel ensures that we don't lose serial output if we have delays uploading
	// the previous chunk to GCS. Assuming a poll interval of 3 seconds, the buffer of size 1024
	// allows ~51 minutes of backlog before we start losing logs.
	c := make(chan string, 1024)
	(*watcher).Watch(name, serialPortToArchive, c, serialPortPollInterval)
	(*watcher).start(name)
	defer w.stepWait.Done()

	var buf bytes.Buffer
	var gcsErr bool

	for {
		serialOutput := <-c
		if serialOutput == "" {
			break
		}
		buf.WriteString(serialOutput)
		wc := wcProvider()
		if _, err := wc.Write([]byte(serialOutput)); err != nil && !gcsErr {
			gcsErr = true
			w.LogStepInfo(s.name, "CreateInstances", "Instance %q: error writing log to GCS: %v", name, err)
			continue
		} else if err != nil { // dont try to close the writer
			continue
		}
		if err := wc.Close(); err != nil && !gcsErr {
			gcsErr = true
			w.LogStepInfo(s.name, "CreateInstances", "Instance %q: error saving log to GCS: %v", name, err)
			continue
		}

		if w.isCanceled() {
			break
		}
	}

	w.Logger.WriteSerialPortLogs(w, name, buf)
}

// populate preprocesses fields: Name, Project, Zone, Description, MachineType, NetworkInterfaces, Scopes, ServiceAccounts, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (ci *CreateInstances) populate(ctx context.Context, s *Step) DError {
	var errs DError
	if ci.Instances != nil {
		for _, i := range ci.Instances {
			errs = addErrs(errs, (&i.InstanceBase).populate(ctx, i, s))
		}
	}

	if ci.InstancesBeta != nil {
		for _, i := range ci.InstancesBeta {
			errs = addErrs(errs, (&i.InstanceBase).populate(ctx, i, s))
		}
	}

	return errs
}

func (ci *CreateInstances) validate(ctx context.Context, s *Step) DError {
	var errs DError
	if ci.instanceUsesBetaFeatures() {
		for _, i := range ci.InstancesBeta {
			errs = addErrs(errs, (&i.InstanceBase).validate(ctx, i, s))
		}
	} else {
		for _, i := range ci.Instances {
			errs = addErrs(errs, (&i.InstanceBase).validate(ctx, i, s))
		}
	}
	return errs
}

func (ci *CreateInstances) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	eChan := make(chan DError)
	createInstance := func(ii InstanceInterface, ib *InstanceBase) {
		// Just try to delete it, a 404 here indicates the instance doesn't exist.
		if ib.OverWrite {
			if err := ii.delete(w.ComputeClient, true); err != nil {
				if apiErr, ok := err.(*googleapi.Error); !ok || apiErr.Code != 404 {
					eChan <- Errf("error deleting existing instance: %v", err)
					return
				}
			}
		}

		// Get the source machine image link if using a source machine image.
		if ii.getSourceMachineImage() != "" {
			if image, ok := w.machineImages.get(ii.getSourceMachineImage()); ok {
				ii.setSourceMachineImage(image.link)
			}
		}
		defer wg.Done()
		ii.updateDisksAndNetworksBeforeCreate(w)

		w.LogStepInfo(s.name, "CreateInstances", "Creating instance %q.", ii.getName())

		if err := ii.create(w.ComputeClient); err != nil {
			// Fallback to no-external-ip mode to workaround organization policy.
			if ib.RetryWhenExternalIPDenied && isExternalIPDeniedByOrganizationPolicy(err) {
				w.LogStepInfo(s.name, "CreateInstances", "Falling back to no-external-ip mode "+
					"for creating instance %v due to the fact that external IP is denied by organization policy.", ii.getName())

				UpdateInstanceNoExternalIP(s)
				err = ii.create(w.ComputeClient)
			}

			if err != nil {
				eChan <- newErr("failed to create instances", err)
				return
			}
		}

		ib.createdInWorkflow = true
		logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", ii.getName(), serialPortToArchive))
		w.LogStepInfo(s.name, "CreateInstances", "Streaming instance %q serial port %d output to https://storage.cloud.google.com/%s/%s",
			ii.getName(), serialPortToArchive, w.bucket, logsObj)
		// The provider is used since logSerialOutput repeatedly closes the writer
		// to flush the logs to GCS.
		provider := func() io.WriteCloser {
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(ctx)
			wc.ContentType = "text/plain"
			return wc
		}
		go logSerialOutput(s, ii.getName(), w.SerialOutputWatcher(), provider)
	}

	if ci.instanceUsesBetaFeatures() {
		for _, i := range ci.InstancesBeta {
			wg.Add(1)
			go createInstance(i, &i.InstanceBase)
		}
	} else {
		for _, i := range ci.Instances {
			wg.Add(1)
			go createInstance(i, &i.InstanceBase)
		}
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

func (ci *CreateInstances) instanceUsesBetaFeatures() bool {
	for _, instanceBeta := range ci.InstancesBeta {
		if instanceBeta != nil && instanceBeta.SourceMachineImage != "" {
			return true
		}
	}
	// if GA instances collection is empty, switch to Beta
	return len(ci.Instances) == 0
}

func isExternalIPDeniedByOrganizationPolicy(err error) bool {
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusPreconditionFailed {
		return strings.Contains(gErr.Message, "constraints/compute.vmExternalIpAccess")
	}
	return false
}

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

package main

import (
	"log"
	"os"
	"sync"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
)

var (
	aw activeWindows
	rc chan *patchWindow
)

func patchInit() {
	aw.windows = map[string]*patchWindow{}
	rc = make(chan *patchWindow)

	// Load current patch state off disk.
	pw, err := loadState(state)
	if err != nil {
		log.Println("ERROR:", err)
	} else if pw != nil && pw.Name != "" && !pw.Complete {
		pw.register()
	}

	// Start the patch runner goroutine.
	go patchRunner()
}

type activeWindows struct {
	windows map[string]*patchWindow
	mx      sync.RWMutex
}

func (a *activeWindows) get(name string) (*patchWindow, bool) {
	a.mx.RLock()
	defer a.mx.RUnlock()
	w, ok := aw.windows[name]
	return w, ok
}

func (a *activeWindows) delete(name string) {
	a.mx.Lock()
	delete(a.windows, name)
	a.mx.Unlock()
}

func (a *activeWindows) add(name string, w *patchWindow) {
	a.mx.Lock()
	a.windows[name] = w
	a.mx.Unlock()
}

type patchPolicy struct {
	*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy
}

// MarshalJSON marchals a patchPolicy using jsonpb.
func (p *patchPolicy) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	s, err := m.MarshalToString(p)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UnmarshalJSON unmarshals apatchPolicy using jsonpb.
func (p *patchPolicy) UnmarshalJSON(b []byte) error {
	return jsonpb.UnmarshalString(string(b), p)
}

type patchWindow struct {
	Name                         string
	Policy                       *patchPolicy
	ScheduledStart, ScheduledEnd time.Time

	StartedAt, EndedAt time.Time `json:",omitempty"`
	Complete           bool
	Errors             []string `json:",omitempty"`

	t      *time.Timer
	cancel chan struct{}
	mx     sync.RWMutex
}

func (w *patchWindow) getName() string {
	w.mx.RLock()
	defer w.mx.RUnlock()
	return w.Name
}

func (w *patchWindow) setName(name string) {
	w.mx.Lock()
	defer w.mx.Unlock()
	w.Name = name
}

func (w *patchWindow) getScheduledStart() time.Time {
	w.mx.RLock()
	defer w.mx.RUnlock()
	return w.ScheduledStart
}

func (w *patchWindow) setScheduledStart(s time.Time) {
	w.mx.Lock()
	defer w.mx.Unlock()
	w.ScheduledStart = s
}

func (w *patchWindow) getScheduledEnd() time.Time {
	w.mx.RLock()
	defer w.mx.RUnlock()
	return w.ScheduledEnd
}

func (w *patchWindow) setScheduledEnd(s time.Time) {
	w.mx.Lock()
	defer w.mx.Unlock()
	w.ScheduledEnd = s
}

func (w *patchWindow) stopTimer() {
	w.mx.RLock()
	defer w.mx.RUnlock()
	w.t.Stop()
}

func (w *patchWindow) setTimer(t *time.Timer) {
	w.mx.Lock()
	defer w.mx.Unlock()
	w.t = t
}

func (w *patchWindow) getPolicy() *patchPolicy {
	w.mx.RLock()
	defer w.mx.RUnlock()
	return w.Policy
}

func (w *patchWindow) setPolicy(p *patchPolicy) {
	w.mx.Lock()
	defer w.mx.Unlock()
	w.Policy = p
}

func (w *patchWindow) String() string {
	return w.getName()
}

func newPatchWindow(pp *osconfigpb.LookupConfigsResponse_EffectivePatchPolicy) (*patchWindow, error) {
	start, end, err := nextWindow(time.Now().UTC(), pp.PatchWindow, 0)
	if err != nil {
		return nil, err
	}
	return &patchWindow{
		Name:           pp.GetFullName(),
		Policy:         &patchPolicy{pp},
		ScheduledStart: start,
		ScheduledEnd:   end,
		cancel:         make(chan struct{}),
	}, nil
}

func (w *patchWindow) in() bool {
	select {
	case <-w.cancel:
		return false
	default:
	}

	now := time.Now()
	return now.After(w.getScheduledStart()) && now.Before(w.getScheduledEnd())
}

// update updates a patchWindow with a PatchPolicy.
func (w *patchWindow) update(p *osconfigpb.LookupConfigsResponse_EffectivePatchPolicy) {
	logger.Println("DEBUG: update", w.Name)
	start, end, err := nextWindow(time.Now().UTC(), p.PatchWindow, 0)
	if err != nil {
		return
	}

	if start != w.getScheduledStart() {
		logger.Println("DEBUG: patchWindow start changed, need to reregister:", w.Name)
		w.stopTimer()
		// Cancel any potential current run if we are no longer in a patch window.
		if w.in() {
			logger.Println("DEBUG: patchWindow running, need to cancel:", w.Name)
			close(w.cancel)
		}

		defer w.register()
	}

	w.setScheduledStart(start)
	w.setScheduledEnd(end)
	w.setPolicy(&patchPolicy{p})

	logger.Println("DEBUG: update end", w.Name)
}

// register registers a patch window as active, this will clobber any existing
// window with the same name.
func (w *patchWindow) register() {
	logger.Println("DEBUG: register", w.Name)
	aw.add(w.getName(), w)

	// Create the Timer that will kick off the patch process.
	// If we happen to be in the patch window now this will start imediately.
	w.setTimer(time.AfterFunc(w.ScheduledStart.Sub(time.Now()), func() { rc <- w }))
	logger.Println(w.Name, "register done")
}

// deregister stops a patch windows timer and removes it from activeWindows
// but does not cancel any current runs.
func (w *patchWindow) deregister() {
	logger.Println("DEBUG: deregister", w.Name)
	w.stopTimer()
	aw.delete(w.getName())
}

func (w *patchWindow) run() (reboot bool) {
	logger.Println("DEBUG: run", w.Name)

	w.StartedAt = time.Now()

	if w.Complete {
		return false
	}

	defer func() {
		if err := saveState(state, w); err != nil {
			log.Println("ERROR:", err)
		}
	}()

	// Make sure we are still in the patch window.
	if !w.in() {
		logger.Println(w.Name, "not in patch window")
		return false
	}

	if err := saveState(state, w); err != nil {
		log.Println("ERROR:", err)
		w.Errors = append(w.Errors, err.Error())
	}

	reboot, err := runUpdates(*w.getPolicy())
	if err != nil {
		// TODO: implement retries
		log.Println("ERROR:", err)
		w.Errors = append(w.Errors, err.Error())
		return false
	}

	// Make sure we are still in the patch window
	if !w.in() {
		logger.Println(w.Name, "timedout")
		w.Errors = append(w.Errors, "Patch window timed out")
		return false
	}

	if !reboot {
		w.Complete = true
		w.EndedAt = time.Now()
	}
	return reboot
}

func setPatchPolicies(efps []*osconfigpb.LookupConfigsResponse_EffectivePatchPolicy) {
	// Deregister any existing patchWindows that no longer exist.
	ppns := make(map[string]struct{})
	for _, pp := range efps {
		ppns[pp.GetFullName()] = struct{}{}
	}

	var toDeregister []*patchWindow
	aw.mx.Lock()
	for _, w := range aw.windows {
		if _, ok := ppns[w.Name]; !ok {
			toDeregister = append(toDeregister, w)
		}
	}
	aw.mx.Unlock()

	logger.Printf("DEBUG: list of patchWindows to deregister: %q\n", toDeregister)
	for _, w := range toDeregister {
		w.deregister()
	}

	// Update or create patchWindows based on provided patch policies.
	for _, pp := range efps {
		logger.Println("DEBUG: setup", pp.GetFullName())
		if w, ok := aw.get(pp.GetFullName()); ok {
			logger.Println("DEBUG: patchWindow to update:", w.Name)
			w.update(pp)
			continue
		}

		w, err := newPatchWindow(pp)
		if err != nil {
			log.Print("ERROR:", err)
			continue
		}
		logger.Println("DEBUG: patchWindow to create:", w.Name)
		w.register()
	}
}

func patchRunner() {
	logger.Println("DEBUG: patchrunner start")
	for {
		logger.Println("DEBUG: waiting for patches to run")
		select {
		case pw := <-rc:
			logger.Println("DEBUG: patchrunner running", pw.Name)
			reboot := pw.run()
			if pw.Policy.RebootConfig == osconfigpb.PatchPolicy_NEVER {
				continue
			}
			if (pw.Policy.RebootConfig == osconfigpb.PatchPolicy_ALWAYS) ||
				(((pw.Policy.RebootConfig == osconfigpb.PatchPolicy_DEFAULT) ||
					(pw.Policy.RebootConfig == osconfigpb.PatchPolicy_REBOOT_CONFIG_UNSPECIFIED)) &&
					reboot) {
				logger.Println("DEBUG: reboot requested", pw.Name)
				if err := rebootSystem(); err != nil {
					logger.Println("ERROR: error running reboot:", err)
				} else {
					// Reboot can take a bit, shutdown the agent so other activities don't start.
					os.Exit(0)
				}
			}
			logger.Println("DEBUG: finished patch window", pw.Name)
		}
	}
}

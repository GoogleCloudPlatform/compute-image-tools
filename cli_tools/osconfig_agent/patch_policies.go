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
	"fmt"
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
	} else if pw != nil && pw.Name != "" {
		pw.register()
	}

	// Start the patch runner goroutine.
	go patchRunner()
}

type activeWindows struct {
	windows map[string]*patchWindow
	mx      sync.Mutex
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
	Name   string
	Policy *patchPolicy

	Start, End time.Time

	t      *time.Timer
	cancel chan struct{}
	mx     sync.RWMutex
}

func (w *patchWindow) String() string {
	return w.Name
}

func newPatchWindow(pp *osconfigpb.LookupConfigsResponse_EffectivePatchPolicy) (*patchWindow, error) {
	start, end, err := nextWindow(time.Now().UTC(), pp.PatchWindow, 0)
	if err != nil {
		return nil, err
	}
	return &patchWindow{
		Name:   pp.GetFullName(),
		Policy: &patchPolicy{pp},
		Start:  start,
		End:    end,
		cancel: make(chan struct{}),
	}, nil
}

func (w *patchWindow) in() bool {
	select {
	case <-w.cancel:
		return false
	default:
	}

	now := time.Now()
	return now.After(w.Start) && now.Before(w.End)
}

// update updates a patchWindow with a PatchPolicy.
func (w *patchWindow) update(p *osconfigpb.LookupConfigsResponse_EffectivePatchPolicy) {
	fmt.Println("DEBUG: update", w.Name)
	start, end, err := nextWindow(time.Now().UTC(), p.PatchWindow, 0)
	if err != nil {
		return
	}

	if start != w.Start {
		fmt.Println("DEBUG: patchWindow start changed, need to reregister:", w.Name)
		w.t.Stop()
		// Cancel any potential current run if we are no longer in a patch window.
		if w.in() {
			fmt.Println("DEBUG: patchWindow running, need to cancel:", w.Name)
			close(w.cancel)
		}

		defer w.register()
	}

	w.mx.Lock()
	w.Start, w.End = start, end
	w.Policy = &patchPolicy{p}
	w.mx.Unlock()

	fmt.Println("DEBUG: update end", w.Name)
}

// register registers a patch window as active, this will clobber any existing
// window with the same name.
func (w *patchWindow) register() {
	fmt.Println("DEBUG: register", w.Name)
	w.mx.Lock()
	defer w.mx.Unlock()

	aw.add(w.Name, w)

	// Create the Timer that will kick off the patch process.
	// If we happen to be in the patch window now this will start imediately.
	w.t = time.AfterFunc(w.Start.Sub(time.Now()), func() { rc <- w })
	fmt.Println(w.Name, "register done")
}

// deregister stops a patch windows timer and removes it from activeWindows
// but does not cancel any current runs.
func (w *patchWindow) deregister() {
	fmt.Println("DEBUG: deregister", w.Name)
	w.t.Stop()
	aw.delete(w.Name)
}

func (w *patchWindow) run() (reboot bool) {
	fmt.Println("DEBUG: run", w.Name)

	w.mx.RLock()
	defer w.mx.RUnlock()

	defer func() {
		if reboot {
			if err := saveState(state, w); err != nil {
				log.Println("ERROR:", err)
			}
		} else {
			if err := saveState(state, nil); err != nil {
				log.Println("ERROR:", err)
			}
		}
	}()

	// Make sure we are still in the patch window.
	if !w.in() {
		fmt.Println(w.Name, "not in patch window")
		return false
	}

	fmt.Println("running patch window", w.Name)
	if err := saveState(state, w); err != nil {
		log.Println("ERROR:", err)
	}

	reboot, err := runUpdates()
	if err != nil {
		// TODO: implement retries
		log.Println("ERROR:", err)
		return false
	}

	// Make sure we are still in the patch window after each step.
	// TODO: Pass this to runUpdates instead?
	if !w.in() {
		fmt.Println(w.Name, "timedout")
		return false
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

	fmt.Printf("DEBUG: list of patchWindows to deregister: %q\n", toDeregister)
	for _, w := range toDeregister {
		defer w.deregister()
	}

	// Update or create patchWindows based on provided patch policies.
	for _, pp := range efps {
		fmt.Println("DEBUG: setup", pp.GetFullName())
		if w, ok := aw.windows[pp.GetFullName()]; ok {
			fmt.Println("DEBUG: patchWindow to update:", w.Name)
			defer w.update(pp)
			continue
		}

		w, err := newPatchWindow(pp)
		if err != nil {
			log.Print("ERROR:", err)
			continue
		}
		fmt.Println("DEBUG: patchWindow to create:", w.Name)
		defer w.register()
	}
	// TODO: Look into simplifying locking, maybe just one lock and no updates
	// during a patch run?
	aw.mx.Unlock()
}

func patchRunner() {
	fmt.Println("DEBUG: patchrunner start")
	for {
		fmt.Println("DEBUG: waiting for patches to run")
		select {
		case pw := <-rc:
			fmt.Println("DEBUG: patchrunner running", pw.Name)
			reboot := pw.run()
			if reboot {
				fmt.Println("DEBUG: reboot requested", pw.Name)
				if err := rebootSystem(); err != nil {
					fmt.Println("ERROR: error running reboot:", err)
				}
				os.Exit(0)
			}
			fmt.Println("DEBUG: finished patch window", pw.Name)
		}
	}
}

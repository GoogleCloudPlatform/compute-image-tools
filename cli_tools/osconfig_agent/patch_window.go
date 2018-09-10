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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

const state = "osconfig_patch.state"

var (
	aw activeWindows
	rc chan *patchWindow
)

func init() {
	aw.windows = map[string]*patchWindow{}
	rc = make(chan *patchWindow)

	// Load current patch state off disk.
	loadState(state)

	// Start the patch runner goroutine.
	go patchRunner()
}

type activeWindows struct {
	windows map[string]*patchWindow
	mx      sync.Mutex
}

func (a *activeWindows) lock() {
	return
	a.mx.Lock()
}

func (a *activeWindows) unlock() {
	return
	a.mx.Unlock()
}

func (a *activeWindows) delete(name string) {
	a.lock()
	delete(a.windows, name)
	a.unlock()
}

func (a *activeWindows) add(name string, w *patchWindow) {
	a.lock()
	a.windows[name] = w
	a.unlock()
}

type patchPolicy struct {
	*osconfigpb.PatchPolicy
}

// MarshalJSON is a hacky workaround to prevent Disk from using compute.Disk's implementation.
func (p *patchPolicy) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	s, err := m.MarshalToString(p)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UnmarshalJSON is a hacky workaround to prevent Disk from using compute.Disk's implementation.
func (p *patchPolicy) UnmarshalJSON(b []byte) error {
	return jsonpb.UnmarshalString(string(b), p)
}

type patchWindow struct {
	Name   string
	Policy *patchPolicy

	Start, End time.Time
	Complete   bool

	t      *time.Timer
	cancel chan struct{}
	mx     sync.RWMutex
}

func newPatchWindow(pp *osconfigpb.PatchPolicy) (*patchWindow, error) {
	start, end, err := nextWindow(time.Now().UTC(), pp.PatchWindow, 0)
	if err != nil {
		return nil, err
	}
	return &patchWindow{
		Name:   pp.GetName(),
		Policy: &patchPolicy{pp},
		Start:  start,
		End:    end,
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
func (w *patchWindow) update(p *osconfigpb.PatchPolicy) {
	fmt.Println("DEBUG: update", w.Name)
	start, end, err := nextWindow(time.Now().UTC(), p.PatchWindow, 0)
	if err != nil {
		return
	}

	w.mx.Lock()
	fmt.Println("DEBUG: update got lock", w.Name)

	w.Policy = &patchPolicy{p}
	if start != w.Start {
		fmt.Println("DEBUG: patchWindow start changed, need to reregister:", w.Name)
		w.t.Stop()
		// Cancel any potential current run if we are no longer in a patch window.
		if w.in() {
			fmt.Println("DEBUG: patchWindow running, need to cancel:", w.Name)
			close(w.cancel)
		}
		w.Start, w.End = start, end

		defer w.register()
	} else {
		w.End = end
	}
	w.mx.Unlock()
	fmt.Println("DEBUG: update unlock", w.Name)

	fmt.Println("DEBUG: update end", w.Name)
}

// register registers a patch window as active, this will clobber any existing
// window with the same name.
func (w *patchWindow) register() {
	fmt.Println("DEBUG: register", w.Name)
	w.mx.Lock()
	defer w.mx.Unlock()
	fmt.Println("DEBUG: register got lock", w.Name)
	defer func() { fmt.Println("DEBUG: register unlock", w.Name) }()

	w.Complete = false
	aw.add(w.Name, w)

	// Create the Timer that will kick off the patch process.
	// If we happen to be in the patch window now this will start imediately.
	w.cancel = make(chan struct{})
	w.t = time.AfterFunc(w.Start.Sub(time.Now()), func() { rc <- w })
	fmt.Println(w.Name, "register done")
}

// deregister stops a patch windows timer and removes it from activeWindows
// but does not cancel any current runs.
func (w *patchWindow) deregister() {
	fmt.Println(w.Name, "deregister")
	w.t.Stop()
	aw.delete(w.Name)
}

func (w *patchWindow) run() bool {
	fmt.Println("DEBUG: run", w.Name)
	w.mx.RLock()
	defer w.mx.RUnlock()
	fmt.Println("DEBUG: run got lock", w.Name)
	defer func() { fmt.Println("DEBUG: run unlock", w.Name) }()

	if w.Complete {
		saveState(state, nil)
		return false
	}

	// Make sure we are still in the patch window.
	if !w.in() {
		fmt.Println(w.Name, "not in patch window")
		saveState(state, nil)
		return false
	}

	fmt.Println("running patch window", w.Name)
	saveState(state, w)

	// Pretend to do work
	time.Sleep(5 * time.Second)

	// Make sure we are still in the patch window after each step.
	if !w.in() {
		fmt.Println(w.Name, "timedout")
		return false
	}

	// ---------------------------
	if w.Name == "flipyflappy" {
		return true
	}
	// ---------------------------

	w.Complete = true
	fmt.Println("finished patch window", w.Name)
	saveState(state, nil)
	return false
}

func patchManager(pps []*osconfigpb.PatchPolicy) {
	// Deregister any existing patchWindows that no longer exist.
	var ppns []string
	for _, pp := range pps {
		ppns = append(ppns, pp.GetName())
	}

	var toDeregister []*patchWindow
	aw.lock()
	defer aw.unlock()
	for _, w := range aw.windows {
		if !strIn(w.Name, ppns) {
			toDeregister = append(toDeregister, w)
		}
	}

	for _, w := range toDeregister {
		fmt.Println("DEBUG: list of patchWindows to deregister:", toDeregister)
		defer w.deregister()
	}

	// Update or create patchWindows based on provided patch policies.
	for _, pp := range pps {
		fmt.Println("DEBUG: setup", pp.GetName())
		if w, ok := aw.windows[pp.GetName()]; ok {
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
}

func loadState(state string) {
	d, err := ioutil.ReadFile(state)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Print("ERROR:", err)
		return
	}

	var pw patchWindow
	if err := json.Unmarshal(d, &pw); err != nil {
		log.Print("ERROR:", err)
		return
	}

	pw.register()
}

func saveState(state string, pw *patchWindow) {
	d, err := json.Marshal(pw)
	if err != nil {
		log.Print("ERROR:", err)
		return
	}

	if err := ioutil.WriteFile(state, d, 0600); err != nil {
		log.Print("ERROR:", err)
		return
	}

	return
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
				os.Exit(0)
			}
		}
	}
}

// nextWindow will return the next applicable time window start relative to now.
func nextWindow(now time.Time, window *osconfigpb.PatchWindow, offset int) (time.Time, time.Time, error) {
	var start time.Time
	var err error
	switch {
	case window.GetDaily() != nil:
		start, err = dailyWindowStart(now, window.GetStartTime(), offset)
	case window.GetWeekly() != nil:
		start, err = weeklyWindowStart(now, window.GetStartTime(), window.GetWeekly(), offset)
	case window.GetMonthly() != nil:
		start, err = monthlyWindowStart(now, window.GetStartTime(), window.GetMonthly(), offset)
	default:
		return now, now, errors.New("no window set in PatchWindow")
	}
	if err != nil {
		return now, now, err
	}

	length := time.Duration(window.GetDuration().GetSeconds()) * time.Second
	end := start.Add(length)
	if now.After(end) {
		return nextWindow(now, window, offset+1)
	}
	return start, end, nil
}

func monthlyWindowStart(now time.Time, start *timeofday.TimeOfDay, window *osconfigpb.PatchWindow_Monthly, offset int) (time.Time, error) {
	var dom int
	month := time.Month(int(now.Month()) + offset)
	firstOfMonth := time.Date(now.Year(), month, 1, 0, 0, 0, 0, now.Location())
	if window.GetOccurrenceOfDay() != nil {
		day := window.GetOccurrenceOfDay().GetDay()
		if day == dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED {
			return now, fmt.Errorf("%q not a valid day", day)
		}
		dom = (7+int(day)-int(firstOfMonth.Weekday()))%7 + 1 + ((int(window.GetOccurrenceOfDay().GetOccurrence()) - 1) * 7)
	} else {
		dom = int(window.GetDayOfMonth())
		if dom == -1 {
			dom = firstOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond).Day()
		}
	}

	// TODO: This will rollover into the following/preceding month, add a check for that.
	return time.Date(now.Year(), month, dom, int(start.Hours), int(start.Minutes), int(start.Seconds), int(start.Nanos), now.Location()), nil
}

func weeklyWindowStart(now time.Time, start *timeofday.TimeOfDay, window *osconfigpb.PatchWindow_Weekly, offset int) (time.Time, error) {
	day := window.GetDay()
	if day == dayofweek.DayOfWeek_DAY_OF_WEEK_UNSPECIFIED {
		return now, fmt.Errorf("%q not a valid day", day)
	}
	t := now.AddDate(0, 0, -int(now.Weekday())).AddDate(0, 0, int(day)+(offset*7))
	return time.Date(t.Year(), t.Month(), t.Day(), int(start.Hours), int(start.Minutes), int(start.Seconds), int(start.Nanos), now.Location()), nil
}

func dailyWindowStart(now time.Time, start *timeofday.TimeOfDay, offset int) (time.Time, error) {
	return time.Date(now.Year(), now.Month(), now.Day()+offset, int(start.Hours), int(start.Minutes), int(start.Seconds), int(start.Nanos), now.Location()), nil
}

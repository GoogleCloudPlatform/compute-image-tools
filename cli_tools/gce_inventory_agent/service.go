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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/service"
)

type program struct {
	run     func(context.Context)
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	timeout time.Duration
}

func (p *program) Start(s service.Service) error {
	go func() {
		p.run(p.ctx)
		close(p.done)
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	p.cancel()
	select {
	case <-p.done:
		return nil
	case <-time.After(p.timeout):
		return fmt.Errorf("failed to shutdown within timeout %s", p.timeout)
	}
}

func usage(name string) {
	fmt.Printf(
		"Usage:\n"+
			"  %[1]s install: install the %[2]s service\n"+
			"  %[1]s remove: remove the %[2]s service\n"+
			"  %[1]s start: start the %[2]s service\n"+
			"  %[1]s stop: stop the %[2]s service\n", filepath.Base(os.Args[0]), name)
}

func register(ctx context.Context, name, displayName, desc string, run func(context.Context), action string) error {
	svcConfig := &service.Config{
		Name:        name,
		DisplayName: displayName,
		Description: desc,
	}

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	prg := &program{
		run:     run,
		ctx:     ctx,
		cancel:  cancel,
		done:    done,
		timeout: 15 * time.Second,
	}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	switch action {
	case "run":
		return svc.Run()
	case "install":
		if err := svc.Install(); err != nil {
			return fmt.Errorf("failed to install service %s: %s", name, err)
		}
	case "remove":
		if err := svc.Uninstall(); err != nil {
			return fmt.Errorf("failed to remove service %s: %s", name, err)
		}
	case "start":
		if err := svc.Start(); err != nil {
			return fmt.Errorf("failed to start service %s: %s", name, err)
		}
	case "stop":
		if err := svc.Stop(); err != nil {
			return fmt.Errorf("failed to stop service %s: %s", name, err)
		}
	case "help":
		usage(name)
	default:
		fmt.Printf("%q is not a valid argument.\n", action)
		usage(name)
	}
	return nil
}

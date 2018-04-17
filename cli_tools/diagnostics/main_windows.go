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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type cmd struct {
	path           string
	args           string
	outputFileName string
	// True when the command produces its own file and doesn't need one
	// created from stdout.
	cmdProducesFile bool
}

type wmiQuery struct {
	class          string
	namespace      string
	outputFileName string
}

func (command cmd) run() (outPath string, err error) {
	outPath = filepath.Join(tmpFolder, command.outputFileName)

	c := exec.Command(command.path)
	argString := command.args

	if command.cmdProducesFile {
		// Replace any output file args with that path in a temp folder
		relPath := command.outputFileName
		argString = strings.Replace(argString, relPath, outPath, -1)
	} else {
		// If the command doesn't produce a file, we need to construct
		// one from Stdout and Stderr
		outFile, err := os.Create(outPath)
		if err != nil {
			log.Printf("Error creating file %s: %v", outPath, err)
			return outPath, err
		}
		defer func() {
			if cErr := outFile.Close(); err != nil {
				err = cErr
			}
		}()
		c.Stdout = outFile
		c.Stderr = outFile
	}

	if command.args != "" {
		c.Args = append(c.Args, strings.Split(argString, " ")...)
	}
	err = c.Run()
	return
}

func (query wmiQuery) run() (string, error) {
	outPath := filepath.Join(tmpFolder, query.outputFileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		return outPath, err
	}
	defer outFile.Close()

	// WMI is somewhat flaky, so we should retry a few times on failures
	var data string
	for i := 0; i < 3; i++ {
		data, err = printWmiObjects(query.class, query.namespace)
		if err == nil {
			break
		}
	}
	if err != nil {
		return outPath, err
	}

	header := fmt.Sprintf("Queried wmi objects [%s] from namespace %s\n\n", query.class, query.namespace)
	if _, err = outFile.WriteString(header); err != nil {
		return outPath, err
	}

	_, err = outFile.WriteString(data)
	return outPath, err
}

func runAll(commands []runner, errCh chan error) []string {
	paths := make([]string, 0, len(commands))

	for _, command := range commands {
		path, err := command.run()
		if err != nil {
			log.Printf("Error: %s while running %v", err, command)
			errCh <- err
		} else {
			paths = append(paths, path)
		}
	}

	return paths
}

func gatherSystemLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		cmd{`C:\Windows\System32\systeminfo.exe`, "", "systeminfo.txt", false},
		cmd{`C:\Windows\System32\bcdedit.exe`, "", "bcdedit.txt", false},
		cmd{`C:\Windows\System32\sc.exe`, "query type=driver", "drivers.txt", false},
		cmd{`C:\Windows\System32\pnputil.exe`, "/e", "pnputil.txt", false},
		cmd{`C:\Windows\System32\msinfo32.exe`, "/report msinfo32.txt", "msinfo32.txt", true},
		wmiQuery{"Win32_UserAccount", `root\CIMv2`, "users.txt"},
	}

	logs <- logFolder{"System", runAll(commands, errs)}
}

func gatherDiskLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		wmiQuery{"MSFT_Disk", `root\Microsoft\Windows\Storage`, "disks.txt"},
		wmiQuery{"MSFT_Volume", `root\Microsoft\Windows\Storage`, "volumes.txt"},
		wmiQuery{"MSFT_Partition", `root\Microsoft\Windows\Storage`, "partitions.txt"},
	}

	logs <- logFolder{"Disk", runAll(commands, errs)}
}

func gatherNetworkLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		cmd{`C:\Windows\System32\nslookup.exe`, "8.8.8.8", "nslookup_dns.txt", false},
		cmd{`C:\Windows\System32\tracert.exe`, "www.gstatic.com", "tracert_gstatic.txt", false},
		cmd{`C:\Windows\System32\ping.exe`, "-n 10 8.8.8.8", "ping_dns.txt", false},
		cmd{`C:\Windows\System32\ping.exe`, "-n 10 www.gstatic.com", "ping_gstatic.txt", false},
		cmd{`C:\Windows\System32\ipconfig.exe`, "/all", "ipconfig.txt", false},
		cmd{`C:\Windows\System32\route.exe`, "print", "route.txt", false},
		cmd{`C:\Windows\System32\netstat.exe`, "-anb", "netstat.txt", false},
		wmiQuery{"MSFT_NetFirewallRule", `root\StandardCimv2`, "firewall.txt"},
	}

	logs <- logFolder{"Network", runAll(commands, errs)}
}

func gatherProgramLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		wmiQuery{"Win32_Process", `root\Cimv2`, "processes.txt"},
		wmiQuery{"Win32_Service", `root\Cimv2`, "services.txt"},
		wmiQuery{"MSFT_ScheduledTask", `root\Microsoft\Windows\TaskScheduler`, "scheduled_tasks.txt"},
	}

	logs <- logFolder{"Program", runAll(commands, errs)}
}

func gatherEventLogs(logs chan logFolder, errs chan error) {
	dirs := []string{`C:\Windows\System32\winevt\Logs`}
	paths := make([]string, 0)
	// Recursively gather all the log files in a directory
	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]
		files, _ := ioutil.ReadDir(dir)
		for _, f := range files {
			path := filepath.Join(dir, f.Name())
			if f.IsDir() {
				dirs = append(dirs, path)
			} else {
				paths = append(paths, path)
			}
		}
	}
	logs <- logFolder{"Event", paths}
}

func gatherTraceLogs(logs chan logFolder, errs chan error) {
	traceStart := cmd{`C:\Windows\System32\wpr.exe`, "-start CPU -start DiskIO -start FileIO -start Network", "trace.etl", true}
	traceStop := cmd{`C:\Windows\System32\wpr.exe`, "-stop trace.etl", "trace.etl", true}

	if _, err := traceStart.run(); err != nil {
		errs <- err
	}

	time.Sleep(10 * time.Minute)
	paths := runAll([]runner{
		traceStop,
	}, errs)
	logs <- logFolder{"Trace", paths}
}

func gatherLogs(trace bool) ([]logFolder, error) {
	runFuncs := []func(logs chan logFolder, errs chan error){
		gatherSystemLogs,
		gatherDiskLogs,
		gatherNetworkLogs,
		gatherProgramLogs,
		gatherEventLogs,
	}
	if trace {
		runFuncs = append(runFuncs, gatherTraceLogs)
	}

	folderCount := len(runFuncs)
	folders := make([]logFolder, 0, folderCount)
	errStrings := make([]string, 0)
	ch := make(chan logFolder, folderCount)
	errs := make(chan error)

	for _, run := range runFuncs {
		go run(ch, errs)
	}

	for {
		select {
		case folder := <-ch:
			folders = append(folders, folder)
		case err := <-errs:
			errStrings = append(errStrings, err.Error())
		}

		if len(folders) == folderCount {
			break
		}
	}

	if len(errs) > 0 {
		return folders, errors.New(strings.Join(errStrings, "\n"))
	}
	return folders, nil
}

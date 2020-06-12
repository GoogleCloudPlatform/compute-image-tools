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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	eventLogsRoot = `C:\Windows\System32\winevt\Logs`
	k8sLogsRoot   = `C:\etc\kubernetes\logs`
	// TODO: user can change the dump path, so better fetch the path from Registry:
	// https://support.microsoft.com/en-us/help/254649/overview-of-memory-dump-file-options-for-windows
	// But it's not likely people will do that.
	crashDump               = `C:\Windows\MEMORY.dmp`
	rdpStatusFileName       = "rdp_status.txt"
	rdpScriptFileName       = "rdp_status.ps1"
	dockerImageListFileName = "docker_images.log"
)

// struct used to construct `Get-WinEvent (-ProviderName/-LogName) ${logName}` cmd string.
type winEvt struct {
	logName string
	// If event logs are from provider i.e. docker, should be True,
	// Otherwise should be false, i.e. Application, System, etc.
	fromProvider bool
}

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
		args := strings.Split(argString, " ")
		for _, arg := range args {
			// Decode the "%20" to space
			c.Args = append(c.Args, strings.ReplaceAll(arg, "%20", " "))
		}
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
			errCh <- fmt.Errorf("Error: %s while running %v", err, command)
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

// gatherRDPSettings invokes rdp_status.ps1 to collect rdp settings,
// return rdp_status.txt file path and errors(if any).
func gatherRDPSettings(logs chan logFolder, errs chan error) {
	pwshPath, err := exec.LookPath("powershell")
	if err != nil {
		errs <- err
		return
	}
	// Get current runtime directory path
	absDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		errs <- err
		return
	}
	rdpScriptFilePath := filepath.Join(absDir, rdpScriptFileName)
	// As we splits the params by spaces, but the rdp script file path has spaces,
	// So encode the space to "%20" to keep the file path as a whole, then decode
	// it back to space later on.
	rdpScriptFilePath = strings.ReplaceAll(rdpScriptFilePath, " ", "%20")
	var commands = []runner{
		cmd{pwshPath, "-ExecutionPolicy Unrestricted -File " + rdpScriptFilePath, rdpStatusFileName, false},
	}

	logs <- logFolder{"RDP", runAll(commands, errs)}
}

// collectFilePaths recursively collect all the file paths under given list of roots,
// return list of file paths and errors(if any).
func collectFilePaths(roots []string) ([]string, []error) {
	filePaths := make([]string, 0)
	errs := make([]error, 0)
	for _, root := range roots {
		// Compared filepath.Walk with orginal BFS folder traversal using Measure-Command cmdlet,
		// looks like almost the same.
		// 		filepath.Walk -> 4s 973ms
		// 		original BFS folder traversal -> 4s 897ms
		// Although filepath.Walk is slower than `find` due to extra lstat calls
		// https://github.com/golang/go/issues/16399, it should be good enough for this scenario.
		err := filepath.Walk(root, func(path string, info os.FileInfo, e error) error {
			if e != nil {
				return e
			}
			if !info.IsDir() {
				filePaths = append(filePaths, path)
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return filePaths, errs
}

// Returns a Get-WinEvent command to get the event log for the specified log source.
func getCommandToConvertEvt(evt winEvt) string {
	cmdStr := `Get-WinEvent %s'%s' | Format-Table -Auto -Wrap`
	source := "-LogName "
	if evt.fromProvider {
		source = "-ProviderName "
	}
	return fmt.Sprintf(cmdStr, source, evt.logName)
}

// getPlainEventLogs generates plain text event logs thru `Get-WinEvent` powershell cmd,
// return list file paths and errors in error channel.
func getPlainEventLogs(evts []winEvt, errs chan error) []string {
	pwshPath, err := exec.LookPath("powershell")
	if err != nil {
		errs <- err
		return []string{}
	}
	commands := make([]runner, 0)
	for _, evt := range evts {
		commands = append(commands, cmd{
			path:            pwshPath,
			args:            getCommandToConvertEvt(evt),
			outputFileName:  evt.logName + ".log",
			cmdProducesFile: false})
	}
	return runAll(commands, errs)
}

// getDockerImagesList put docker images list file in logFolder channel and errors in error channel.
func getDockerImagesList(logs chan logFolder, errs chan error) {
	dockerImageFilePath := make([]string, 0, 1)
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		log.Printf("Docker not installed, couldn't get docker images list.\n")
	} else {
		var commands = []runner{
			cmd{dockerPath, "image list", dockerImageListFileName, false},
		}
		dockerImageFilePath = runAll(commands, errs)
	}
	logs <- logFolder{"Kubernetes", dockerImageFilePath}
}

// gatherEventLogs put all the event log file paths in logFolder channel
// and errors in error channel.
func gatherEventLogs(logs chan logFolder, errs chan error) {
	roots := []string{eventLogsRoot}
	filePaths, ers := collectFilePaths(roots)
	for _, err := range ers {
		errs <- err
	}
	plainEvtLogPaths := getPlainEventLogs([]winEvt{
		{logName: "Application", fromProvider: false},
		{logName: "System", fromProvider: false},
	}, errs)
	filePaths = append(filePaths, plainEvtLogPaths...)
	logs <- logFolder{"Event", filePaths}
}

// gatherKubernetesLogs put all the kubernetes log file paths in logFolder channel
// and errors in error channel.
func gatherKubernetesLogs(logs chan logFolder, errs chan error) {
	roots := make([]string, 0)
	var err error
	_, err = os.Stat(k8sLogsRoot)
	if err == nil {
		roots = append(roots, k8sLogsRoot)
	} else if os.IsNotExist(err) {
		log.Printf("%s doesn't exists, no need to collect inside logs.\n", k8sLogsRoot)
	} else {
		log.Printf("unexpected error happened when check existence of %s: %v", k8sLogsRoot, err)
	}
	_, err = os.Stat(crashDump)
	if err == nil {
		roots = append(roots, crashDump)
	} else if os.IsNotExist(err) {
		log.Printf("%s doesn't exists, no need to collect.\n", crashDump)
	} else {
		log.Printf("unexpected error happened when check existence of %s: %v", crashDump, err)
	}
	// Collect the kubernetes logs and crash dump
	filePaths := make([]string, 0)
	if len(roots) > 0 {
		var ers []error
		filePaths, ers = collectFilePaths(roots)
		for _, err := range ers {
			errs <- err
		}
	}
	// Collect docker event logs
	_, err = exec.LookPath("docker")
	if err == nil {
		plainEvtLogPaths := getPlainEventLogs([]winEvt{
			{logName: "docker", fromProvider: true},
		}, errs)
		filePaths = append(filePaths, plainEvtLogPaths...)
	} else {
		log.Printf("Docker not installed, no docker logs.\n")
	}
	logs <- logFolder{"Kubernetes", filePaths}
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
		gatherKubernetesLogs,
		gatherRDPSettings,
		getDockerImagesList,
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
	if len(errStrings) > 0 {
		return folders, errors.New(strings.Join(errStrings, "\n"))
	}
	return folders, nil
}

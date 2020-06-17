package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// RunCmd executes the named program with given arguments.
// Logs the running command.
func RunCmd(name string, args []string) error {
	log.Printf("Running command: '%s %s'", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCmdAndGetOutput executes the named program with given arguments.
// Logs the running command. Returns the output of the executed program, and error if any.
func RunCmdAndGetOutput(cmdString string, args []string) ([]byte, error) {
	log.Printf("Running command: '%s %s'", cmdString, strings.Join(args, " "))
	return RunCmdAndGetOutputWithoutLog(cmdString, args)
}

// RunCmdAndGetOutputWithoutLog executes the named program with given arguments.
// Returns the output of the executed program, and error if any.
func RunCmdAndGetOutputWithoutLog(cmdString string, args []string) ([]byte, error) {
	output, err := exec.Command(cmdString, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v%v", string(output), err)
	}
	return output, nil
}

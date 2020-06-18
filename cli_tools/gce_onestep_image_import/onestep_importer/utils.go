//  Copyright 2020 Google Inc. All Rights Reserved.
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

package importer

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
)

// runCmd executes the named program with given arguments.
// Logs the running command.
func runCmd(name string, args []string) error {
	log.Printf("Running command: '%s %s'", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCmdAndGetOutput executes the named program with given arguments.
// Logs the running command. Returns the output of the executed program, and error if any.
func runCmdAndGetOutput(cmdString string, args []string) ([]byte, error) {
	log.Printf("Running command: '%s %s'", cmdString, strings.Join(args, " "))
	return runCmdAndGetOutputWithoutLog(cmdString, args)
}

// runCmdAndGetOutputWithoutLog executes the named program with given arguments.
// Returns the output of the executed program, and error if any.
func runCmdAndGetOutputWithoutLog(cmdString string, args []string) ([]byte, error) {
	output, err := exec.Command(cmdString, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v%v", string(output), err)
	}
	return output, nil
}

// importFromCloudProvider imports image from the specified cloud provider
func importFromCloudProvider(args *OneStepImportArguments) error {
	var exportedGCSPath string

	// 1. import from specified cloud provider
	if args.CloudProvider == "aws" {
		importer, err := newAWSImporter(args.Oauth, newAWSImportArguments(args))
		if err != nil {
			return err
		}

		if exportedGCSPath, err = importer.run(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("import from cloud provider %v is currently not supported", args.CloudProvider)
	}

	// 2. update source file flag
	args.SourceFile = exportedGCSPath

	// 3. run image import
	err := runImageImport(args)
	if err != nil {
		log.Println("Failed to import image.",
			fmt.Sprintf("The image file has been copied to Google Cloud Storage, located at %v.", exportedGCSPath),
			"To resume the import process, please directly use image import from GCS.")
	}
	return err
}

// runImageImport calls image import
func runImageImport(args *OneStepImportArguments) error {
	err := runCmd("gce_vm_image_import", []string{
		fmt.Sprintf("-image_name=%v", args.ImageName),
		fmt.Sprintf("-client_id=%v", args.ClientID),
		fmt.Sprintf("-os=%v", args.OS),
		fmt.Sprintf("-source_file=%v", args.SourceFile),
		fmt.Sprintf("-no_guest_environment=%v", args.NoGuestEnvironment),
		fmt.Sprintf("-family=%v", args.Family),
		fmt.Sprintf("-description=%v", args.Description),
		fmt.Sprintf("-network=%v", args.Network),
		fmt.Sprintf("-subnet=%v", args.Subnet),
		fmt.Sprintf("-zone=%v", args.Zone),
		fmt.Sprintf("-timeout=%v", args.Timeout),
		fmt.Sprintf("-project=%v", args.Project),
		fmt.Sprintf("-scratch_bucket_gcs_path=%v", args.ScratchBucketGcsPath),
		fmt.Sprintf("-oauth=%v", args.Oauth),
		fmt.Sprintf("-compute_endpoint_override=%v", args.ComputeEndpoint),
		fmt.Sprintf("-disable_gcs_logging=%v", args.GcsLogsDisabled),
		fmt.Sprintf("-disable_cloud_logging=%v", args.CloudLogsDisabled),
		fmt.Sprintf("-disable_stdout_logging=%v", args.StdoutLogsDisabled),
		fmt.Sprintf("-no_external_ip=%v", args.NoExternalIP),
		fmt.Sprintf("-labels=%v", keyValueString(args.Labels).String()),
		fmt.Sprintf("-storage_location=%v", args.StorageLocation)})
	if err != nil {
		return err
	}
	return nil
}

// keyValueString is an implementation of flag.Value that creates a map
// from the user's argument prior to storing it. It expects the argument
// is in the form KEY1=AB,KEY2=CD. For more info on the format, see
// param.ParseKeyValues.
type keyValueString map[string]string

func (s keyValueString) String() string {
	parts := []string{}
	for k, v := range s {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

func (s *keyValueString) Set(input string) error {
	if *s != nil {
		return fmt.Errorf("only one instance of this flag is allowed")
	}

	*s = make(map[string]string)
	if input != "" {
		var err error
		*s, err = param.ParseKeyValues(input)
		if err != nil {
			return err
		}
	}
	return nil
}

// trimmedString is an implementation of flag.Value that trims whitespace
// from the incoming argument prior to storing it.
type trimmedString string

func (s trimmedString) String() string { return (string)(s) }
func (s *trimmedString) Set(input string) error {
	*s = trimmedString(strings.TrimSpace(input))
	return nil
}

// lowerTrimmedString is an implementation of flag.Value that trims whitespace
// and converts to lowercase the incoming argument prior to storing it.
type lowerTrimmedString string

func (s lowerTrimmedString) String() string { return (string)(s) }
func (s *lowerTrimmedString) Set(input string) error {
	*s = lowerTrimmedString(strings.ToLower(strings.TrimSpace(input)))
	return nil
}

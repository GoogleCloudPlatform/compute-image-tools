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

package args

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/importer"
)

// Flags that are validated.
const (
	imageFlag          = "image_name"
	clientFlag         = "client_id"
	dataDiskFlag       = "data_disk"
	osFlag             = "os"
	customWorkflowFlag = "custom_translate_workflow"
)

// ParsedArguments holds the structured results of parsing CLI arguments.
type ParsedArguments struct {
	Img         importer.ImageSpec
	Env         importer.Environment
	Translation importer.TranslationSpec
}

// ParseArgs parses, validates, and populates the CLI arguments that are
// used by the importer tool.
func ParseArgs(args []string, populator param.Populator) (ParsedArguments, error) {
	flagSet := flag.NewFlagSet("image-import", flag.ContinueOnError)
	// Don't write parse errors to stdout, instead propagate them via an
	// exception since we use flag.ContinueOnError.
	flagSet.SetOutput(ioutil.Discard)

	img := importer.ImageSpec{}
	env := importer.Environment{
		CurrentExecutablePath: os.Args[0],
	}
	translation := importer.TranslationSpec{}

	registerImageSpec(flagSet, &img)
	registerEnvironment(flagSet, &env)
	registerTranslationSpec(flagSet, &translation)

	err := flagSet.Parse(args)
	if err != nil {
		return ParsedArguments{}, err
	}

	if err := populator.PopulateMissingParameters(&env.Project, &env.Zone, &env.Region,
		&env.ScratchBucketGcsPath, translation.SourceFile, &img.StorageLocation); err != nil {
		return ParsedArguments{}, err
	}

	if err := populateNetwork(&env); err != nil {
		return ParsedArguments{}, err
	}

	if err := validate(img, env, translation); err != nil {
		return ParsedArguments{}, err
	}

	return ParsedArguments{img, env, translation}, err
}

func validate(img importer.ImageSpec, env importer.Environment, rules importer.TranslationSpec) error {
	if env.ClientID == "" {
		return fmt.Errorf("-%s has to be specified", clientFlag)
	}
	if img.Name == "" {
		return fmt.Errorf("-%s has to be specified", imageFlag)
	}
	if !rules.DataDisk && rules.OS == "" && rules.CustomWorkflow == "" {
		return fmt.Errorf("-%s, -%s, or -%s has to be specified",
			dataDiskFlag, osFlag, customWorkflowFlag)
	}
	if rules.DataDisk && (rules.OS != "" || rules.CustomWorkflow != "") {
		return fmt.Errorf("when -%s is specified, -%s and -%s should be empty",
			dataDiskFlag, osFlag, customWorkflowFlag)
	}
	if rules.OS != "" && rules.CustomWorkflow != "" {
		return fmt.Errorf("-%s and -%s can't be both specified",
			osFlag, customWorkflowFlag)
	}
	if rules.OS != "" {
		if err := daisy_utils.ValidateOS(rules.OS); err != nil {
			return err
		}
	}
	return nil
}

func populateNetwork(e *importer.Environment) error {
	// Populate Network and Subnet. Two goals:
	//
	// a. Explicitly use the 'default' network only when
	//    network is omitted and subnet is empty.
	// b. Convert bare identifiers to URIs.
	//
	// Rules: https://cloud.google.com/vpc/docs/vpc
	if e.Network == "" && e.Subnet == "" {
		e.Network = "default"
	}
	if e.Subnet != "" {
		e.Subnet = param.GetRegionalResourcePath(e.Region, "subnetworks", e.Subnet)
	}
	if e.Network != "" {
		e.Network = param.GetGlobalResourcePath("networks", e.Network)
	}

	return nil
}

func registerEnvironment(flagSet *flag.FlagSet, e *importer.Environment) {
	flagSet.Var((*lowerTrimmedString)(&e.ClientID), clientFlag,
		"Identifies the client of the importer, e.g. 'gcloud' or 'pantheon'")

	flagSet.Var((*trimmedString)(&e.Project), "project",
		"The project where workflows will be run, and where the resulting image will be stored.")

	flagSet.Var((*trimmedString)(&e.Network), "network",
		"Name of the network in your project to use for the image import. "+
			"The network must have access to Google Cloud Storage. "+
			"If not specified, the network named default is used.")

	flagSet.Var((*trimmedString)(&e.Subnet), "subnet",
		"Name of the subnetwork in your project to use for the image import. "+
			"If the network resource is in legacy mode, do not provide this property. "+
			"If the network is in auto subnet mode, providing the subnetwork is optional. "+
			"If the network is in custom subnet mode, then this field should be specified. "+
			"Zone should be specified if this field is specified.")

	flagSet.Var((*lowerTrimmedString)(&e.Zone), "zone",
		"The zone where workflows will be run, and where the resulting image will be stored.")

	flagSet.Var((*trimmedString)(&e.ScratchBucketGcsPath), "scratch_bucket_gcs_path",
		"GCS scratch bucket to use. A bucket will be created if omitted.")

	flagSet.Var((*trimmedString)(&e.Oauth), "oauth",
		"Path to oauth json file.")

	flagSet.Var((*trimmedString)(&e.ComputeEndpoint), "compute_endpoint_override",
		"API endpoint to override default.")

	flagSet.BoolVar(&e.GcsLogsDisabled, "disable_gcs_logging", false,
		"Do not store logs in GCS.")

	flagSet.BoolVar(&e.CloudLogsDisabled, "disable_cloud_logging", false,
		"Do not store logs in Cloud Logging.")

	flagSet.BoolVar(&e.StdoutLogsDisabled, "disable_stdout_logging", false,
		"Do not write logs to stdout.")

	flagSet.BoolVar(&e.NoExternalIP, "no_external_ip", false,
		"VPC doesn't allow external IPs.")

}

func registerImageSpec(flagSet *flag.FlagSet, i *importer.ImageSpec) {
	flagSet.Var((*lowerTrimmedString)(&i.Name), imageFlag,
		"Name of the disk image to create.")

	flagSet.Var((*trimmedString)(&i.Family), "family",
		"Family to set for the imported image.")

	flagSet.Var((*trimmedString)(&i.Description), "description",
		"Description to set for the imported image.")

	flagSet.Var((*keyValueString)(&i.Labels), "labels",
		"List of label KEY=VALUE pairs to add. "+
			"For more information, see: https://cloud.google.com/compute/docs/labeling-resources")

	flagSet.Var((*lowerTrimmedString)(&i.StorageLocation), "storage_location",
		"Specifies a Cloud Storage location, either regional or multi-regional, "+
			"where image content is to be stored. If not specified, the multi-region "+
			"location closest to the source is chosen automatically.")
}

func registerTranslationSpec(flagSet *flag.FlagSet, t *importer.TranslationSpec) {
	flagSet.Var((*trimmedString)(&t.SourceFile), "source_file",
		"The Cloud Storage URI of the virtual disk file to import.")

	flagSet.Var((*trimmedString)(&t.SourceImage), "source_image",
		"An existing Compute Engine image from which to import.")

	flagSet.BoolVar(&t.DataDisk, dataDiskFlag, false,
		"Specifies that the disk has no bootable OS installed on it. "+
			"Imports the disk without making it bootable or installing Google tools on it.")

	flagSet.Var((*lowerTrimmedString)(&t.OS), osFlag,
		"Specifies the OS of the image being imported. OS must be one of: "+
			"centos-6, centos-7, debian-8, debian-9, opensuse-15, sles-12-byol, "+
			"sles-15-byol, rhel-6, rhel-6-byol, rhel-7, rhel-7-byol, ubuntu-1404, "+
			"ubuntu-1604, ubuntu-1804, windows-10-byol, windows-2008r2, windows-2008r2-byol, "+
			"windows-2012, windows-2012-byol, windows-2012r2, windows-2012r2-byol, "+
			"windows-2016, windows-2016-byol, windows-7-byol.")

	flagSet.BoolVar(&t.NoGuestEnvironment, "no_guest_environment", false,
		"When enabled, the Google Guest Environment will not be installed.")

	flagSet.DurationVar(&t.Timeout, "timeout", time.Hour*2,
		"Maximum time a build can last before it is failed as TIMEOUT. For example, "+
			"specifying 2h will fail the process after 2 hours. See $ gcloud topic datetimes "+
			"for information on duration formats.")

	flagSet.Var((*trimmedString)(&t.CustomWorkflow), customWorkflowFlag,
		"A Daisy workflow JSON file to use for translation.")

	flagSet.BoolVar(&t.UefiCompatible, "uefi_compatible", false,
		"Enables UEFI booting, which is an alternative system boot method. "+
			"Most public images use the GRUB bootloader as their primary boot method.")

	flagSet.BoolVar(&t.SysprepWindows, "sysprep_windows", false,
		"Whether to generalize image using Windows Sysprep. Only applicable to Windows.")
}

// keyValueString is an implementation of flag.Value that creates a map
// from the user's argument prior to storing it. It expects the argument
// is in the form KEY1=AB,KEY2=CD. For more info on the format, see
// param.ParseKeyValues.
type keyValueString map[string]string

func (l keyValueString) String() string {
	parts := []string{}
	for k, v := range l {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

func (l *keyValueString) Set(s string) error {
	if *l != nil {
		return fmt.Errorf("only one instance of this flag is allowed")
	}

	*l = make(map[string]string, 0)
	if s != "" {
		var err error
		*l, err = param.ParseKeyValues(s)
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

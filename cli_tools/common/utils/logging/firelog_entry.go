//  Copyright 2019 Google Inc. All Rights Reserved.
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

package logging

// LogRequest is a server-side pre-defined data structure
type LogRequest struct {
	ClientInfo    ClientInfo `json:"client_info"`
	LogSource     int64      `json:"log_source"`
	RequestTimeMs int64      `json:"request_time_ms"`
	LogEvent      []LogEvent `json:"log_event"`
}

// ClientInfo is a server-side pre-defined data structure
type ClientInfo struct {
	ClientType        string            `json:"client_type"`
	DesktopClientInfo map[string]string `json:"desktop_client_info"`
}

// LogEvent is a server-side pre-defined data structure
type LogEvent struct {
	EventTimeMs         int64  `json:"event_time_ms"`
	EventUptimeMs       int64  `json:"event_uptime_ms"`
	SourceExtensionJSON string `json:"source_extension_json"`
}

// LogResponse is a server-side pre-defined data structure
type LogResponse struct {
	NextRequestWaitMillis int64                `json:"nextRequestWaitMillis,string"`
	LogResponseDetails    []LogResponseDetails `json:"logResponseDetails"`
}

// LogResponseDetails is a server-side pre-defined data structure
type LogResponseDetails struct {
	ResponseAction ResponseAction `json:"responseAction"`
}

// ResponseAction is a server-side pre-defined data structure
type ResponseAction string

const (
	// ResponseActionUnknown - If the client sees this, it should delete the LogRequest (not retry).
	// It may indicate that a new response action was added, which the client
	// doesn't yet understand.  (Deleting rather than retrying will prevent
	// infinite loops.)  The server will do whatever it can to prevent this
	// occurring (by not indicating an action to clients that are behind the
	// requisite version for the action).
	ResponseActionUnknown ResponseAction = "RESPONSE_ACTION_UNKNOWN"
	// RetryRequestLater - The client should retry the request later, via normal scheduling.
	RetryRequestLater ResponseAction = "RETRY_REQUEST_LATER"
	// DeleteRequest - The client should delete the request.  This action will apply for
	// successful requests, and non-retryable requests.
	DeleteRequest ResponseAction = "DELETE_REQUEST"
)

// ComputeImageToolsLogExtension contains all log info, which should be align with sawmill server side configuration.
type ComputeImageToolsLogExtension struct {
	// This id is a random guid for correlation among multiple log lines of a single call
	ID           string       `json:"id"`
	CloudBuildID string       `json:"cloud_build_id"`
	ToolAction   string       `json:"tool_action"`
	Status       string       `json:"status"`
	InputParams  *InputParams `json:"input_params,omitempty"`
	OutputInfo   *OutputInfo  `json:"output_info,omitempty"`
}

// InputParams contains the union of all APIs' param info. To simplify logging service, we
// avoid defining different schemas for each API.
type InputParams struct {
	ImageImportParams    *ImageImportParams    `json:"image_import_input_params,omitempty"`
	ImageExportParams    *ImageExportParams    `json:"image_export_input_params,omitempty"`
	InstanceImportParams *InstanceImportParams `json:"instance_import_input_params,omitempty"`
}

// ImageImportParams contains all input params for image import
type ImageImportParams struct {
	*CommonParams

	ImageName          string `json:"image_name,omitempty"`
	DataDisk           bool   `json:"data_disk"`
	OS                 string `json:"os,omitempty"`
	SourceFile         string `json:"source_file,omitempty"`
	SourceImage        string `json:"source_image,omitempty"`
	NoGuestEnvironment bool   `json:"no_guest_environment"`
	Family             string `json:"family,omitempty"`
	Description        string `json:"description,omitempty"`
	NoExternalIP       bool   `json:"no_external_ip"`
	HasKmsKey          bool   `json:"has_kms_key"`
	HasKmsKeyring      bool   `json:"has_kms_keyring"`
	HasKmsLocation     bool   `json:"has_kms_location"`
	HasKmsProject      bool   `json:"has_kms_project"`
	StorageLocation    string `json:"storage_location,omitempty"`
}

// ImageExportParams contains all input params for image export
type ImageExportParams struct {
	*CommonParams

	DestinationURI string `json:"destination_uri,omitempty"`
	SourceImage    string `json:"source_image,omitempty"`
	Format         string `json:"format,omitempty"`
}

// InstanceImportParams contains all input params for instance import
type InstanceImportParams struct {
	*CommonParams

	InstanceName                string `json:"instance_name,omitempty"`
	SourceURI                   string `json:"source_uri,omitempty"`
	CanIPForward                bool   `json:"can_ip_forward"`
	DeletionProtection          bool   `json:"deletion_protection"`
	MachineType                 string `json:"machine_type,omitempty"`
	NetworkInterface            string `json:"network_interface,omitempty"`
	NetworkTier                 string `json:"network_tier,omitempty"`
	PrivateNetworkIP            string `json:"private_network_ip,omitempty"`
	NoExternalIP                bool   `json:"no_external_ip,omitempty"`
	NoRestartOnFailure          bool   `json:"no_restart_on_failure"`
	OS                          string `json:"os,omitempty"`
	ShieldedIntegrityMonitoring bool   `json:"shielded_integrity_monitoring"`
	ShieldedSecureBoot          bool   `json:"shielded_secure_boot"`
	ShieldedVtpm                bool   `json:"shielded_vtpm"`
	Tags                        string `json:"tags,omitempty"`
	HasBootDiskKmsKey           bool   `json:"has_boot_disk_kms_key"`
	HasBootDiskKmsKeyring       bool   `json:"has_boot_disk_kms_keyring"`
	HasBootDiskKmsLocation      bool   `json:"has_boot_disk_kms_location"`
	HasBootDiskKmsProject       bool   `json:"has_boot_disk_kms_project"`
	NoGuestEnvironment          bool   `json:"no_guest_environment"`
	Node                        string `json:"node,omitempty"`
	NodeAffinityLabel           string `json:"node_affinity_label,omitempty"`
}

// CommonParams is only used to organize the code without impacting hierarchy of data
type CommonParams struct {
	ClientID                string `json:"client_id,omitempty"`
	Network                 string `json:"network,omitempty"`
	Subnet                  string `json:"subnet,omitempty"`
	Zone                    string `json:"zone,omitempty"`
	Timeout                 string `json:"timeout,omitempty"`
	Project                 string `json:"project,omitempty"`
	ObfuscatedProject       string `json:"obfuscated_project,omitempty"`
	Labels                  string `json:"labels,omitempty"`
	ScratchBucketGcsPath    string `json:"scratch_bucket_gcs_path,omitempty"`
	Oauth                   string `json:"oauth,omitempty"`
	ComputeEndpointOverride string `json:"compute_endpoint_override,omitempty"`
	DisableGcsLogging       bool   `json:"disable_gcs_logging"`
	DisableCloudLogging     bool   `json:"disable_cloud_logging"`
	DisableStdoutLogging    bool   `json:"disable_stdout_logging"`
}

// OutputInfo contains output values from the tools execution
type OutputInfo struct {
	// Size of import/export source (image or file)
	SourceSizeGb int64 `json:"source_size_gb,omitempty"`
	// Size of import/export target (image or file)
	TargetSizeGb int64 `json:"target_size_gb,omitempty"`
	// Failure message of the command
	FailureMessage string `json:"failure_message,omitempty"`
	// Failure message of the command without privacy info
	FailureMessageWithoutPrivacyInfo string `json:"failure_message_without_privacy_info,omitempty"`
	// ElapsedTimeMs represents the time elapsed for the tool execution
	ElapsedTimeMs int64 `json:"elapsed_time_ms"`
}

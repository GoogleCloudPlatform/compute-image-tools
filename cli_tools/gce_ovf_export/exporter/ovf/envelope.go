/*
Copyright (c) 2015 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ovf

import "encoding/xml"

// Envelope represents OVF descriptor
type Envelope struct {
	XMLName   xml.Name `xml:"http://schemas.dmtf.org/ovf/envelope/1 Envelope"`
	XMLNSCIM  string   `xml:"xmlns:cim,attr"`
	XMLNSOVF  string   `xml:"xmlns:ovf,attr"`
	XMLNSRASD string   `xml:"xmlns:rasd,attr"`
	XMLNSVMW  string   `xml:"xmlns:vmw,attr"`
	XMLNSVSSD string   `xml:"xmlns:vssd,attr"`
	XMLNSXSI  string   `xml:"xmlns:xsi,attr"`

	References []File `xml:"References>File"`

	// Package level meta-data
	Annotation         *AnnotationSection         `xml:"AnnotationSection"`
	Product            *ProductSection            `xml:"ProductSection"`
	Network            *NetworkSection            `xml:"NetworkSection"`
	Disk               *DiskSection               `xml:"DiskSection"`
	OperatingSystem    *OperatingSystemSection    `xml:"OperatingSystemSection"`
	Eula               *EulaSection               `xml:"EulaSection"`
	VirtualHardware    *VirtualHardwareSection    `xml:"VirtualHardwareSection"`
	ResourceAllocation *ResourceAllocationSection `xml:"ResourceAllocationSection"`
	DeploymentOption   *DeploymentOptionSection   `xml:"DeploymentOptionSection"`

	// Content: A VirtualSystem or a VirtualSystemCollection
	VirtualSystem *VirtualSystem `xml:"VirtualSystem"`
}

// VirtualSystem represents OVF virtual system
type VirtualSystem struct {
	Content

	Annotation      []AnnotationSection      `xml:"AnnotationSection"`
	Product         []ProductSection         `xml:"ProductSection"`
	OperatingSystem []OperatingSystemSection `xml:"OperatingSystemSection"`
	Eula            []EulaSection            `xml:"EulaSection"`
	VirtualHardware []VirtualHardwareSection `xml:"VirtualHardwareSection"`
}

// File represents file element
type File struct {
	ID          string  `xml:"id,attr"`
	Href        string  `xml:"href,attr"`
	Size        uint    `xml:"size,attr"`
	Compression *string `xml:"compression,attr"`
	ChunkSize   *int    `xml:"chunkSize,attr"`
}

// Content is a base struct for other named OVF elements
type Content struct {
	ID   string  `xml:"id,attr"`
	Info string  `xml:"Info"`
	Name *string `xml:"Name"`
}

// Section is a base struct representing unnamed sections
type Section struct {
	Required *bool  `xml:"required,attr"`
	Info     string `xml:"Info"`
}

// AnnotationSection is annotation section
type AnnotationSection struct {
	Section

	Annotation string `xml:"Annotation"`
}

// ProductSection is product section
type ProductSection struct {
	Section

	Class    *string `xml:"class,attr"`
	Instance *string `xml:"instance,attr"`

	Product     string     `xml:"Product"`
	Vendor      string     `xml:"Vendor"`
	Version     string     `xml:"Version"`
	FullVersion string     `xml:"FullVersion"`
	ProductURL  string     `xml:"ProductUrl"`
	VendorURL   string     `xml:"VendorUrl"`
	AppURL      string     `xml:"AppUrl"`
	Property    []Property `xml:"Property"`
}

// Property represents a property
type Property struct {
	Key              string  `xml:"key,attr"`
	Type             string  `xml:"type,attr"`
	Qualifiers       *string `xml:"qualifiers,attr"`
	UserConfigurable *bool   `xml:"userConfigurable,attr"`
	Default          *string `xml:"value,attr"`
	Password         *bool   `xml:"password,attr"`

	Label       *string `xml:"Label"`
	Description *string `xml:"Description"`

	Values []PropertyConfigurationValue `xml:"Value"`
}

// PropertyConfigurationValue represents property configuration value
type PropertyConfigurationValue struct {
	Value         string  `xml:"value,attr"`
	Configuration *string `xml:"configuration,attr"`
}

// NetworkSection represents network section
type NetworkSection struct {
	Section

	Networks []Network `xml:"Network"`
}

// Network represents network
type Network struct {
	Name string `xml:"name,attr"`

	Description string `xml:"Description"`
}

// DiskSection represents disk section
type DiskSection struct {
	Section

	Disks []VirtualDiskDesc `xml:"Disk"`
}

//VirtualDiskDesc represents virtual disk description
type VirtualDiskDesc struct {
	DiskID                  string  `xml:"ovf:diskId,attr"`
	FileRef                 *string `xml:"ovf:fileRef,attr"`
	Capacity                string  `xml:"ovf:capacity,attr"`
	CapacityAllocationUnits *string `xml:"ovf:capacityAllocationUnits,attr"`
	Format                  *string `xml:"ovf:format,attr"`
	PopulatedSize           *int    `xml:"ovf:populatedSize,attr"`
	ParentRef               *string `xml:"ovf:parentRef,attr"`
}

// OperatingSystemSection represents operating system section
type OperatingSystemSection struct {
	Section

	ID      int16   `xml:"id,attr"`
	Version *string `xml:"version,attr"`
	OSType  *string `xml:"osType,attr"`

	Description *string `xml:"Description"`
}

// EulaSection represents EULA section
type EulaSection struct {
	Section

	License string `xml:"License"`
}

// VirtualHardwareSection represents virtual hardware section
type VirtualHardwareSection struct {
	Section

	ID        *string `xml:"id,attr"`
	Transport *string `xml:"transport,attr"`

	System      *VirtualSystemSettingData       `xml:"System"`
	Item        []ResourceAllocationSettingData `xml:"Item"`
	StorageItem []StorageAllocationSettingData  `xml:"StorageItem"`
}

// VirtualSystemSettingData represents virtual system settings
type VirtualSystemSettingData struct {
	CIMVirtualSystemSettingData
}

// ResourceAllocationSettingData represents resource allocation settings
type ResourceAllocationSettingData struct {
	CIMResourceAllocationSettingData

	Bound         *string `xml:"bound,attr"`
	Configuration *string `xml:"configuration,attr"`
	Required      *bool   `xml:"required,attr"`
}

// StorageAllocationSettingData represents storage allocation settings
type StorageAllocationSettingData struct {
	CIMStorageAllocationSettingData

	Required      *bool   `xml:"required,attr"`
	Configuration *string `xml:"configuration,attr"`
	Bound         *string `xml:"bound,attr"`
}

// ResourceAllocationSection represents resource allocations
type ResourceAllocationSection struct {
	Section

	Item []ResourceAllocationSettingData `xml:"Item"`
}

// DeploymentOptionSection represents deployment options
type DeploymentOptionSection struct {
	Section

	Configuration []DeploymentOptionConfiguration `xml:"Configuration"`
}

// DeploymentOptionConfiguration represents deployment options
type DeploymentOptionConfiguration struct {
	ID      string `xml:"id,attr"`
	Default *bool  `xml:"default,attr"`

	Label       string `xml:"Label"`
	Description string `xml:"Description"`
}

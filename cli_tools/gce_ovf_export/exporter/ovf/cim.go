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

import (
	"github.com/vmware/govmomi/vim25/types"
)

/*
Source: http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.24.0/CIM_VirtualSystemSettingData.xsd
*/

// CIMVirtualSystemSettingData represents VirtualSystemSetting element
type CIMVirtualSystemSettingData struct {
	AutomaticRecoveryAction              *uint8   `xml:"vssd:AutomaticRecoveryAction"`
	AutomaticShutdownAction              *uint8   `xml:"vssd:AutomaticShutdownAction"`
	AutomaticStartupAction               *uint8   `xml:"vssd:AutomaticStartupAction"`
	AutomaticStartupActionDelay          *string  `xml:"vssd:AutomaticStartupActionDelay>Interval"`
	AutomaticStartupActionSequenceNumber *uint16  `xml:"vssd:AutomaticStartupActionSequenceNumber"`
	Caption                              *string  `xml:"vssd:Caption"`
	ConfigurationDataRoot                *string  `xml:"vssd:ConfigurationDataRoot"`
	ConfigurationFile                    *string  `xml:"vssd:ConfigurationFile"`
	ConfigurationID                      *string  `xml:"vssd:ConfigurationID"`
	CreationTime                         *string  `xml:"vssd:CreationTime"`
	Description                          *string  `xml:"vssd:Description"`
	ElementName                          string   `xml:"vssd:ElementName"`
	InstanceID                           string   `xml:"vssd:InstanceID"`
	LogDataRoot                          *string  `xml:"vssd:LogDataRoot"`
	Notes                                []string `xml:"vssd:Notes"`
	RecoveryFile                         *string  `xml:"vssd:RecoveryFile"`
	SnapshotDataRoot                     *string  `xml:"vssd:SnapshotDataRoot"`
	SuspendDataRoot                      *string  `xml:"vssd:SuspendDataRoot"`
	SwapFileDataRoot                     *string  `xml:"vssd:SwapFileDataRoot"`
	VirtualSystemIdentifier              *string  `xml:"vssd:VirtualSystemIdentifier"`
	VirtualSystemType                    *string  `xml:"vssd:VirtualSystemType"`
}

/*
Source: http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.24.0/CIM_ResourceAllocationSettingData.xsd
*/

//CIMResourceAllocationSettingData represents ResourceAllocationSetting element
type CIMResourceAllocationSettingData struct {
	AddressOnParent       *string  `xml:"rasd:AddressOnParent"`
	Address               *string  `xml:"rasd:Address"`
	AllocationUnits       *string  `xml:"rasd:AllocationUnits"`
	AutomaticAllocation   *bool    `xml:"rasd:AutomaticAllocation"`
	AutomaticDeallocation *bool    `xml:"rasd:AutomaticDeallocation"`
	Caption               *string  `xml:"rasd:Caption"`
	Connection            []string `xml:"rasd:Connection"`
	ConsumerVisibility    *uint16  `xml:"rasd:ConsumerVisibility"`
	Description           *string  `xml:"rasd:Description"`
	ElementName           string   `xml:"rasd:ElementName"`
	HostResource          []string `xml:"rasd:HostResource"`
	InstanceID            string   `xml:"rasd:InstanceID"`
	Limit                 *uint64  `xml:"rasd:Limit"`
	MappingBehavior       *uint    `xml:"rasd:MappingBehavior"`
	OtherResourceType     *string  `xml:"rasd:OtherResourceType"`
	Parent                *string  `xml:"rasd:Parent"`
	PoolID                *string  `xml:"rasd:PoolID"`
	Reservation           *uint64  `xml:"rasd:Reservation"`
	ResourceSubType       *string  `xml:"rasd:ResourceSubType"`
	ResourceType          *uint16  `xml:"rasd:ResourceType"`
	VirtualQuantity       *uint    `xml:"rasd:VirtualQuantity"`
	VirtualQuantityUnits  *string  `xml:"rasd:VirtualQuantityUnits"`
	Weight                *uint    `xml:"rasd:Weight"`
}

/*
Source: http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2.24.0/CIM_StorageAllocationSettingData.xsd
*/

//CIMStorageAllocationSettingData represents StorageAllocationSetting
type CIMStorageAllocationSettingData struct {
	ElementName string `xml:"rasd:ElementName"`
	InstanceID  string `xml:"rasd:InstanceID"`

	ResourceType      *uint16 `xml:"rasd:ResourceType"`
	OtherResourceType *string `xml:"rasd:OtherResourceType"`
	ResourceSubType   *string `xml:"rasd:ResourceSubType"`

	Access                       *uint16         `xml:"rasd:Access"`
	Address                      *string         `xml:"rasd:Address"`
	AddressOnParent              *string         `xml:"rasd:AddressOnParent"`
	AllocationUnits              *string         `xml:"rasd:AllocationUnits"`
	AutomaticAllocation          *bool           `xml:"rasd:AutomaticAllocation"`
	AutomaticDeallocation        *bool           `xml:"rasd:AutomaticDeallocation"`
	Caption                      *string         `xml:"rasd:Caption"`
	ChangeableType               *uint16         `xml:"rasd:ChangeableType"`
	ComponentSetting             []types.AnyType `xml:"rasd:ComponentSetting"`
	ConfigurationName            *string         `xml:"rasd:ConfigurationName"`
	Connection                   []string        `xml:"rasd:Connection"`
	ConsumerVisibility           *uint16         `xml:"rasd:ConsumerVisibility"`
	Description                  *string         `xml:"rasd:Description"`
	Generation                   *uint64         `xml:"rasd:Generation"`
	HostExtentName               *string         `xml:"rasd:HostExtentName"`
	HostExtentNameFormat         *uint16         `xml:"rasd:HostExtentNameFormat"`
	HostExtentNameNamespace      *uint16         `xml:"rasd:HostExtentNameNamespace"`
	HostExtentStartingAddress    *uint64         `xml:"rasd:HostExtentStartingAddress"`
	HostResource                 []string        `xml:"rasd:HostResource"`
	HostResourceBlockSize        *uint64         `xml:"rasd:HostResourceBlockSize"`
	Limit                        *uint64         `xml:"rasd:Limit"`
	MappingBehavior              *uint           `xml:"rasd:MappingBehavior"`
	OtherHostExtentNameFormat    *string         `xml:"rasd:OtherHostExtentNameFormat"`
	OtherHostExtentNameNamespace *string         `xml:"rasd:OtherHostExtentNameNamespace"`
	Parent                       *string         `xml:"rasd:Parent"`
	PoolID                       *string         `xml:"rasd:PoolID"`
	Reservation                  *uint64         `xml:"rasd:Reservation"`
	SoID                         *string         `xml:"rasd:SoID"`
	SoOrgID                      *string         `xml:"rasd:SoOrgID"`
	VirtualQuantity              *uint           `xml:"rasd:VirtualQuantity"`
	VirtualQuantityUnits         *string         `xml:"rasd:VirtualQuantityUnits"`
	VirtualResourceBlockSize     *uint64         `xml:"rasd:VirtualResourceBlockSize"`
	Weight                       *uint           `xml:"rasd:Weight"`
}

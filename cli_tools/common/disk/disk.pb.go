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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.11.4
// source: disk.proto

package disk

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type PartitionScheme int32

const (
	PartitionScheme_NONE PartitionScheme = 0
	PartitionScheme_MBR  PartitionScheme = 1
	PartitionScheme_GPT  PartitionScheme = 2
)

// Enum value maps for PartitionScheme.
var (
	PartitionScheme_name = map[int32]string{
		0: "NONE",
		1: "MBR",
		2: "GPT",
	}
	PartitionScheme_value = map[string]int32{
		"NONE": 0,
		"MBR":  1,
		"GPT":  2,
	}
)

func (x PartitionScheme) Enum() *PartitionScheme {
	p := new(PartitionScheme)
	*p = x
	return p
}

func (x PartitionScheme) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (PartitionScheme) Descriptor() protoreflect.EnumDescriptor {
	return file_disk_proto_enumTypes[0].Descriptor()
}

func (PartitionScheme) Type() protoreflect.EnumType {
	return &file_disk_proto_enumTypes[0]
}

func (x PartitionScheme) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use PartitionScheme.Descriptor instead.
func (PartitionScheme) EnumDescriptor() ([]byte, []int) {
	return file_disk_proto_rawDescGZIP(), []int{0}
}

type Disk struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PartitionScheme PartitionScheme `protobuf:"varint,1,opt,name=partitionScheme,proto3,enum=PartitionScheme" json:"partitionScheme,omitempty"`
}

func (x *Disk) Reset() {
	*x = Disk{}
	if protoimpl.UnsafeEnabled {
		mi := &file_disk_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Disk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Disk) ProtoMessage() {}

func (x *Disk) ProtoReflect() protoreflect.Message {
	mi := &file_disk_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Disk.ProtoReflect.Descriptor instead.
func (*Disk) Descriptor() ([]byte, []int) {
	return file_disk_proto_rawDescGZIP(), []int{0}
}

func (x *Disk) GetPartitionScheme() PartitionScheme {
	if x != nil {
		return x.PartitionScheme
	}
	return PartitionScheme_NONE
}

var File_disk_proto protoreflect.FileDescriptor

var file_disk_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x64, 0x69, 0x73, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x42, 0x0a, 0x04,
	0x44, 0x69, 0x73, 0x6b, 0x12, 0x3a, 0x0a, 0x0f, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x10, 0x2e,
	0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x65, 0x52,
	0x0f, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x65,
	0x2a, 0x2d, 0x0a, 0x0f, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x63, 0x68,
	0x65, 0x6d, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00, 0x12, 0x07, 0x0a,
	0x03, 0x4d, 0x42, 0x52, 0x10, 0x01, 0x12, 0x07, 0x0a, 0x03, 0x47, 0x50, 0x54, 0x10, 0x02, 0x42,
	0x08, 0x5a, 0x06, 0x2e, 0x3b, 0x64, 0x69, 0x73, 0x6b, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_disk_proto_rawDescOnce sync.Once
	file_disk_proto_rawDescData = file_disk_proto_rawDesc
)

func file_disk_proto_rawDescGZIP() []byte {
	file_disk_proto_rawDescOnce.Do(func() {
		file_disk_proto_rawDescData = protoimpl.X.CompressGZIP(file_disk_proto_rawDescData)
	})
	return file_disk_proto_rawDescData
}

var file_disk_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_disk_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_disk_proto_goTypes = []interface{}{
	(PartitionScheme)(0), // 0: PartitionScheme
	(*Disk)(nil),         // 1: Disk
}
var file_disk_proto_depIdxs = []int32{
	0, // 0: Disk.partitionScheme:type_name -> PartitionScheme
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_disk_proto_init() }
func file_disk_proto_init() {
	if File_disk_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_disk_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Disk); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_disk_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_disk_proto_goTypes,
		DependencyIndexes: file_disk_proto_depIdxs,
		EnumInfos:         file_disk_proto_enumTypes,
		MessageInfos:      file_disk_proto_msgTypes,
	}.Build()
	File_disk_proto = out.File
	file_disk_proto_rawDesc = nil
	file_disk_proto_goTypes = nil
	file_disk_proto_depIdxs = nil
}

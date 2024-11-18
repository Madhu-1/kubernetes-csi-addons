// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v3.19.6
// source: encryptionkeyrotation.proto

package proto

import (
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

// EncryptionKeyRotateRequest contains the information needed to identify
// the volume by the csi-addons sidecar and access any backend services so that the
// key can be rotated.
type EncryptionKeyRotateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the pv
	// This field is required
	PvName string `protobuf:"bytes,1,opt,name=pv_name,json=pvName,proto3" json:"pv_name,omitempty"`
}

func (x *EncryptionKeyRotateRequest) Reset() {
	*x = EncryptionKeyRotateRequest{}
	mi := &file_encryptionkeyrotation_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EncryptionKeyRotateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EncryptionKeyRotateRequest) ProtoMessage() {}

func (x *EncryptionKeyRotateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_encryptionkeyrotation_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EncryptionKeyRotateRequest.ProtoReflect.Descriptor instead.
func (*EncryptionKeyRotateRequest) Descriptor() ([]byte, []int) {
	return file_encryptionkeyrotation_proto_rawDescGZIP(), []int{0}
}

func (x *EncryptionKeyRotateRequest) GetPvName() string {
	if x != nil {
		return x.PvName
	}
	return ""
}

// EncryptionKeyRotateResponse holds the information about the result of the
// EncryptionKeyRotateRequest call.
type EncryptionKeyRotateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *EncryptionKeyRotateResponse) Reset() {
	*x = EncryptionKeyRotateResponse{}
	mi := &file_encryptionkeyrotation_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EncryptionKeyRotateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EncryptionKeyRotateResponse) ProtoMessage() {}

func (x *EncryptionKeyRotateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_encryptionkeyrotation_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EncryptionKeyRotateResponse.ProtoReflect.Descriptor instead.
func (*EncryptionKeyRotateResponse) Descriptor() ([]byte, []int) {
	return file_encryptionkeyrotation_proto_rawDescGZIP(), []int{1}
}

var File_encryptionkeyrotation_proto protoreflect.FileDescriptor

var file_encryptionkeyrotation_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x6b, 0x65, 0x79, 0x72,
	0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x35, 0x0a, 0x1a, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x4b, 0x65, 0x79, 0x52, 0x6f, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x70, 0x76, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x76, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x1d, 0x0a, 0x1b, 0x45,
	0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x4b, 0x65, 0x79, 0x52, 0x6f, 0x74, 0x61,
	0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x77, 0x0a, 0x15, 0x45, 0x6e,
	0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x4b, 0x65, 0x79, 0x52, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x5e, 0x0a, 0x13, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x4b, 0x65, 0x79, 0x52, 0x6f, 0x74, 0x61, 0x74, 0x65, 0x12, 0x21, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x4b, 0x65, 0x79,
	0x52, 0x6f, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x22, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x4b, 0x65, 0x79, 0x52, 0x6f, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x00, 0x42, 0x3c, 0x5a, 0x3a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x63, 0x73, 0x69, 0x2d, 0x61, 0x64, 0x64, 0x6f, 0x6e, 0x73, 0x2f, 0x6b, 0x75, 0x62,
	0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2d, 0x63, 0x73, 0x69, 0x2d, 0x61, 0x64, 0x64, 0x6f,
	0x6e, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_encryptionkeyrotation_proto_rawDescOnce sync.Once
	file_encryptionkeyrotation_proto_rawDescData = file_encryptionkeyrotation_proto_rawDesc
)

func file_encryptionkeyrotation_proto_rawDescGZIP() []byte {
	file_encryptionkeyrotation_proto_rawDescOnce.Do(func() {
		file_encryptionkeyrotation_proto_rawDescData = protoimpl.X.CompressGZIP(file_encryptionkeyrotation_proto_rawDescData)
	})
	return file_encryptionkeyrotation_proto_rawDescData
}

var file_encryptionkeyrotation_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_encryptionkeyrotation_proto_goTypes = []any{
	(*EncryptionKeyRotateRequest)(nil),  // 0: proto.EncryptionKeyRotateRequest
	(*EncryptionKeyRotateResponse)(nil), // 1: proto.EncryptionKeyRotateResponse
}
var file_encryptionkeyrotation_proto_depIdxs = []int32{
	0, // 0: proto.EncryptionKeyRotation.EncryptionKeyRotate:input_type -> proto.EncryptionKeyRotateRequest
	1, // 1: proto.EncryptionKeyRotation.EncryptionKeyRotate:output_type -> proto.EncryptionKeyRotateResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_encryptionkeyrotation_proto_init() }
func file_encryptionkeyrotation_proto_init() {
	if File_encryptionkeyrotation_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_encryptionkeyrotation_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_encryptionkeyrotation_proto_goTypes,
		DependencyIndexes: file_encryptionkeyrotation_proto_depIdxs,
		MessageInfos:      file_encryptionkeyrotation_proto_msgTypes,
	}.Build()
	File_encryptionkeyrotation_proto = out.File
	file_encryptionkeyrotation_proto_rawDesc = nil
	file_encryptionkeyrotation_proto_goTypes = nil
	file_encryptionkeyrotation_proto_depIdxs = nil
}

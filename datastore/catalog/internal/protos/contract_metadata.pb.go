// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        v5.29.3
// source: protos/datastore/contract_metadata.proto

package datastore

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ContractMetadataKeyFilter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Domain        *wrapperspb.StringValue `protobuf:"bytes,1,opt,name=domain,proto3" json:"domain,omitempty"`
	Environment   *wrapperspb.StringValue `protobuf:"bytes,2,opt,name=environment,proto3" json:"environment,omitempty"`
	ChainSelector *wrapperspb.UInt64Value `protobuf:"bytes,3,opt,name=chain_selector,json=chainSelector,proto3" json:"chain_selector,omitempty"`
	Address       *wrapperspb.StringValue `protobuf:"bytes,4,opt,name=address,proto3" json:"address,omitempty"`
}

func (x *ContractMetadataKeyFilter) Reset() {
	*x = ContractMetadataKeyFilter{}
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContractMetadataKeyFilter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContractMetadataKeyFilter) ProtoMessage() {}

func (x *ContractMetadataKeyFilter) ProtoReflect() protoreflect.Message {
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContractMetadataKeyFilter.ProtoReflect.Descriptor instead.
func (*ContractMetadataKeyFilter) Descriptor() ([]byte, []int) {
	return file_protos_datastore_contract_metadata_proto_rawDescGZIP(), []int{0}
}

func (x *ContractMetadataKeyFilter) GetDomain() *wrapperspb.StringValue {
	if x != nil {
		return x.Domain
	}
	return nil
}

func (x *ContractMetadataKeyFilter) GetEnvironment() *wrapperspb.StringValue {
	if x != nil {
		return x.Environment
	}
	return nil
}

func (x *ContractMetadataKeyFilter) GetChainSelector() *wrapperspb.UInt64Value {
	if x != nil {
		return x.ChainSelector
	}
	return nil
}

func (x *ContractMetadataKeyFilter) GetAddress() *wrapperspb.StringValue {
	if x != nil {
		return x.Address
	}
	return nil
}

type ContractMetadataFindRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	KeyFilter *ContractMetadataKeyFilter `protobuf:"bytes,1,opt,name=key_filter,json=keyFilter,proto3" json:"key_filter,omitempty"`
}

func (x *ContractMetadataFindRequest) Reset() {
	*x = ContractMetadataFindRequest{}
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContractMetadataFindRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContractMetadataFindRequest) ProtoMessage() {}

func (x *ContractMetadataFindRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContractMetadataFindRequest.ProtoReflect.Descriptor instead.
func (*ContractMetadataFindRequest) Descriptor() ([]byte, []int) {
	return file_protos_datastore_contract_metadata_proto_rawDescGZIP(), []int{1}
}

func (x *ContractMetadataFindRequest) GetKeyFilter() *ContractMetadataKeyFilter {
	if x != nil {
		return x.KeyFilter
	}
	return nil
}

type ContractMetadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Domain        string `protobuf:"bytes,1,opt,name=domain,proto3" json:"domain,omitempty"`
	Environment   string `protobuf:"bytes,2,opt,name=environment,proto3" json:"environment,omitempty"`
	ChainSelector uint64 `protobuf:"varint,3,opt,name=chain_selector,json=chainSelector,proto3" json:"chain_selector,omitempty"`
	Address       string `protobuf:"bytes,4,opt,name=address,proto3" json:"address,omitempty"`
	Metadata      string `protobuf:"bytes,5,opt,name=metadata,proto3" json:"metadata,omitempty"`
	RowVersion    int32  `protobuf:"varint,6,opt,name=row_version,json=rowVersion,proto3" json:"row_version,omitempty"`
}

func (x *ContractMetadata) Reset() {
	*x = ContractMetadata{}
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContractMetadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContractMetadata) ProtoMessage() {}

func (x *ContractMetadata) ProtoReflect() protoreflect.Message {
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContractMetadata.ProtoReflect.Descriptor instead.
func (*ContractMetadata) Descriptor() ([]byte, []int) {
	return file_protos_datastore_contract_metadata_proto_rawDescGZIP(), []int{2}
}

func (x *ContractMetadata) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

func (x *ContractMetadata) GetEnvironment() string {
	if x != nil {
		return x.Environment
	}
	return ""
}

func (x *ContractMetadata) GetChainSelector() uint64 {
	if x != nil {
		return x.ChainSelector
	}
	return 0
}

func (x *ContractMetadata) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *ContractMetadata) GetMetadata() string {
	if x != nil {
		return x.Metadata
	}
	return ""
}

func (x *ContractMetadata) GetRowVersion() int32 {
	if x != nil {
		return x.RowVersion
	}
	return 0
}

type ContractMetadataFindResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	References []*ContractMetadata `protobuf:"bytes,1,rep,name=references,proto3" json:"references,omitempty"`
}

func (x *ContractMetadataFindResponse) Reset() {
	*x = ContractMetadataFindResponse{}
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContractMetadataFindResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContractMetadataFindResponse) ProtoMessage() {}

func (x *ContractMetadataFindResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContractMetadataFindResponse.ProtoReflect.Descriptor instead.
func (*ContractMetadataFindResponse) Descriptor() ([]byte, []int) {
	return file_protos_datastore_contract_metadata_proto_rawDescGZIP(), []int{3}
}

func (x *ContractMetadataFindResponse) GetReferences() []*ContractMetadata {
	if x != nil {
		return x.References
	}
	return nil
}

type ContractMetadataEditRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Record    *ContractMetadata `protobuf:"bytes,1,opt,name=record,proto3" json:"record,omitempty"`
	Semantics EditSemantics     `protobuf:"varint,2,opt,name=semantics,proto3,enum=chainlink.catalog.datastore.EditSemantics" json:"semantics,omitempty"`
}

func (x *ContractMetadataEditRequest) Reset() {
	*x = ContractMetadataEditRequest{}
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContractMetadataEditRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContractMetadataEditRequest) ProtoMessage() {}

func (x *ContractMetadataEditRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContractMetadataEditRequest.ProtoReflect.Descriptor instead.
func (*ContractMetadataEditRequest) Descriptor() ([]byte, []int) {
	return file_protos_datastore_contract_metadata_proto_rawDescGZIP(), []int{4}
}

func (x *ContractMetadataEditRequest) GetRecord() *ContractMetadata {
	if x != nil {
		return x.Record
	}
	return nil
}

func (x *ContractMetadataEditRequest) GetSemantics() EditSemantics {
	if x != nil {
		return x.Semantics
	}
	return EditSemantics_SEMANTICS_INSERT
}

type ContractMetadataEditResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Record *ContractMetadata `protobuf:"bytes,1,opt,name=record,proto3" json:"record,omitempty"`
}

func (x *ContractMetadataEditResponse) Reset() {
	*x = ContractMetadataEditResponse{}
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ContractMetadataEditResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContractMetadataEditResponse) ProtoMessage() {}

func (x *ContractMetadataEditResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protos_datastore_contract_metadata_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContractMetadataEditResponse.ProtoReflect.Descriptor instead.
func (*ContractMetadataEditResponse) Descriptor() ([]byte, []int) {
	return file_protos_datastore_contract_metadata_proto_rawDescGZIP(), []int{5}
}

func (x *ContractMetadataEditResponse) GetRecord() *ContractMetadata {
	if x != nil {
		return x.Record
	}
	return nil
}

var File_protos_datastore_contract_metadata_proto protoreflect.FileDescriptor

var file_protos_datastore_contract_metadata_proto_rawDesc = []byte{
	0x0a, 0x28, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x73, 0x74, 0x6f,
	0x72, 0x65, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x5f, 0x6d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x6c, 0x69, 0x6e, 0x6b, 0x2e, 0x63, 0x61, 0x74, 0x61, 0x6c, 0x6f, 0x67, 0x2e, 0x64, 0x61,
	0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1d, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f,
	0x64, 0x61, 0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8e, 0x02, 0x0a, 0x19, 0x43, 0x6f, 0x6e, 0x74, 0x72,
	0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x4b, 0x65, 0x79, 0x46, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x12, 0x34, 0x0a, 0x06, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x52, 0x06, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x12, 0x3e, 0x0a, 0x0b, 0x65, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0b, 0x65,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x43, 0x0a, 0x0e, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x5f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x52, 0x0d, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12,
	0x36, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07,
	0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x74, 0x0a, 0x1b, 0x43, 0x6f, 0x6e, 0x74, 0x72,
	0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x46, 0x69, 0x6e, 0x64, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x55, 0x0a, 0x0a, 0x6b, 0x65, 0x79, 0x5f, 0x66, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x36, 0x2e, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x6c, 0x69, 0x6e, 0x6b, 0x2e, 0x63, 0x61, 0x74, 0x61, 0x6c, 0x6f, 0x67, 0x2e, 0x64,
	0x61, 0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63,
	0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x4b, 0x65, 0x79, 0x46, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x52, 0x09, 0x6b, 0x65, 0x79, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x22, 0xca, 0x01,
	0x0a, 0x10, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x64, 0x6f, 0x6d, 0x61, 0x69, 0x6e, 0x12, 0x20, 0x0a, 0x0b, 0x65, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x25, 0x0a, 0x0e,
	0x63, 0x68, 0x61, 0x69, 0x6e, 0x5f, 0x73, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x0d, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x53, 0x65, 0x6c, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1a, 0x0a,
	0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x6f, 0x77,
	0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a,
	0x72, 0x6f, 0x77, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x6d, 0x0a, 0x1c, 0x43, 0x6f,
	0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x46, 0x69,
	0x6e, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4d, 0x0a, 0x0a, 0x72, 0x65,
	0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2d,
	0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x6c, 0x69, 0x6e, 0x6b, 0x2e, 0x63, 0x61, 0x74, 0x61, 0x6c,
	0x6f, 0x67, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x6f, 0x6e,
	0x74, 0x72, 0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x0a, 0x72,
	0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x73, 0x22, 0xae, 0x01, 0x0a, 0x1b, 0x43, 0x6f,
	0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x64,
	0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x45, 0x0a, 0x06, 0x72, 0x65, 0x63,
	0x6f, 0x72, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x6c, 0x69, 0x6e, 0x6b, 0x2e, 0x63, 0x61, 0x74, 0x61, 0x6c, 0x6f, 0x67, 0x2e, 0x64, 0x61,
	0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x06, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64,
	0x12, 0x48, 0x0a, 0x09, 0x73, 0x65, 0x6d, 0x61, 0x6e, 0x74, 0x69, 0x63, 0x73, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x2a, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x6c, 0x69, 0x6e, 0x6b, 0x2e,
	0x63, 0x61, 0x74, 0x61, 0x6c, 0x6f, 0x67, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x73, 0x74, 0x6f, 0x72,
	0x65, 0x2e, 0x45, 0x64, 0x69, 0x74, 0x53, 0x65, 0x6d, 0x61, 0x6e, 0x74, 0x69, 0x63, 0x73, 0x52,
	0x09, 0x73, 0x65, 0x6d, 0x61, 0x6e, 0x74, 0x69, 0x63, 0x73, 0x22, 0x65, 0x0a, 0x1c, 0x43, 0x6f,
	0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x64,
	0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x45, 0x0a, 0x06, 0x72, 0x65,
	0x63, 0x6f, 0x72, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x6c, 0x69, 0x6e, 0x6b, 0x2e, 0x63, 0x61, 0x74, 0x61, 0x6c, 0x6f, 0x67, 0x2e, 0x64,
	0x61, 0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63,
	0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x06, 0x72, 0x65, 0x63, 0x6f, 0x72,
	0x64, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x73, 0x6d, 0x61, 0x72, 0x74, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x6b, 0x69, 0x74,
	0x2f, 0x63, 0x61, 0x74, 0x61, 0x6c, 0x6f, 0x67, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x2f, 0x64, 0x61, 0x74, 0x61, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_protos_datastore_contract_metadata_proto_rawDescOnce sync.Once
	file_protos_datastore_contract_metadata_proto_rawDescData = file_protos_datastore_contract_metadata_proto_rawDesc
)

func file_protos_datastore_contract_metadata_proto_rawDescGZIP() []byte {
	file_protos_datastore_contract_metadata_proto_rawDescOnce.Do(func() {
		file_protos_datastore_contract_metadata_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_datastore_contract_metadata_proto_rawDescData)
	})
	return file_protos_datastore_contract_metadata_proto_rawDescData
}

var file_protos_datastore_contract_metadata_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_protos_datastore_contract_metadata_proto_goTypes = []any{
	(*ContractMetadataKeyFilter)(nil),    // 0: chainlink.catalog.datastore.ContractMetadataKeyFilter
	(*ContractMetadataFindRequest)(nil),  // 1: chainlink.catalog.datastore.ContractMetadataFindRequest
	(*ContractMetadata)(nil),             // 2: chainlink.catalog.datastore.ContractMetadata
	(*ContractMetadataFindResponse)(nil), // 3: chainlink.catalog.datastore.ContractMetadataFindResponse
	(*ContractMetadataEditRequest)(nil),  // 4: chainlink.catalog.datastore.ContractMetadataEditRequest
	(*ContractMetadataEditResponse)(nil), // 5: chainlink.catalog.datastore.ContractMetadataEditResponse
	(*wrapperspb.StringValue)(nil),       // 6: google.protobuf.StringValue
	(*wrapperspb.UInt64Value)(nil),       // 7: google.protobuf.UInt64Value
	(EditSemantics)(0),                   // 8: chainlink.catalog.datastore.EditSemantics
}
var file_protos_datastore_contract_metadata_proto_depIdxs = []int32{
	6, // 0: chainlink.catalog.datastore.ContractMetadataKeyFilter.domain:type_name -> google.protobuf.StringValue
	6, // 1: chainlink.catalog.datastore.ContractMetadataKeyFilter.environment:type_name -> google.protobuf.StringValue
	7, // 2: chainlink.catalog.datastore.ContractMetadataKeyFilter.chain_selector:type_name -> google.protobuf.UInt64Value
	6, // 3: chainlink.catalog.datastore.ContractMetadataKeyFilter.address:type_name -> google.protobuf.StringValue
	0, // 4: chainlink.catalog.datastore.ContractMetadataFindRequest.key_filter:type_name -> chainlink.catalog.datastore.ContractMetadataKeyFilter
	2, // 5: chainlink.catalog.datastore.ContractMetadataFindResponse.references:type_name -> chainlink.catalog.datastore.ContractMetadata
	2, // 6: chainlink.catalog.datastore.ContractMetadataEditRequest.record:type_name -> chainlink.catalog.datastore.ContractMetadata
	8, // 7: chainlink.catalog.datastore.ContractMetadataEditRequest.semantics:type_name -> chainlink.catalog.datastore.EditSemantics
	2, // 8: chainlink.catalog.datastore.ContractMetadataEditResponse.record:type_name -> chainlink.catalog.datastore.ContractMetadata
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_protos_datastore_contract_metadata_proto_init() }
func file_protos_datastore_contract_metadata_proto_init() {
	if File_protos_datastore_contract_metadata_proto != nil {
		return
	}
	file_protos_datastore_common_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_protos_datastore_contract_metadata_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protos_datastore_contract_metadata_proto_goTypes,
		DependencyIndexes: file_protos_datastore_contract_metadata_proto_depIdxs,
		MessageInfos:      file_protos_datastore_contract_metadata_proto_msgTypes,
	}.Build()
	File_protos_datastore_contract_metadata_proto = out.File
	file_protos_datastore_contract_metadata_proto_rawDesc = nil
	file_protos_datastore_contract_metadata_proto_goTypes = nil
	file_protos_datastore_contract_metadata_proto_depIdxs = nil
}

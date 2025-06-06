// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: blobcast/rollupapis/v1/get_chain_info_response.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetChainInfoResponse struct {
	state                protoimpl.MessageState `protogen:"open.v1"`
	ChainId              string                 `protobuf:"bytes,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	FinalizedHeight      uint64                 `protobuf:"varint,2,opt,name=finalized_height,json=finalizedHeight,proto3" json:"finalized_height,omitempty"`
	CelestiaHeightOffset uint64                 `protobuf:"varint,3,opt,name=celestia_height_offset,json=celestiaHeightOffset,proto3" json:"celestia_height_offset,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *GetChainInfoResponse) Reset() {
	*x = GetChainInfoResponse{}
	mi := &file_blobcast_rollupapis_v1_get_chain_info_response_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetChainInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetChainInfoResponse) ProtoMessage() {}

func (x *GetChainInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_blobcast_rollupapis_v1_get_chain_info_response_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetChainInfoResponse.ProtoReflect.Descriptor instead.
func (*GetChainInfoResponse) Descriptor() ([]byte, []int) {
	return file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescGZIP(), []int{0}
}

func (x *GetChainInfoResponse) GetChainId() string {
	if x != nil {
		return x.ChainId
	}
	return ""
}

func (x *GetChainInfoResponse) GetFinalizedHeight() uint64 {
	if x != nil {
		return x.FinalizedHeight
	}
	return 0
}

func (x *GetChainInfoResponse) GetCelestiaHeightOffset() uint64 {
	if x != nil {
		return x.CelestiaHeightOffset
	}
	return 0
}

var File_blobcast_rollupapis_v1_get_chain_info_response_proto protoreflect.FileDescriptor

const file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDesc = "" +
	"\n" +
	"4blobcast/rollupapis/v1/get_chain_info_response.proto\x12\x16blobcast.rollupapis.v1\"\x92\x01\n" +
	"\x14GetChainInfoResponse\x12\x19\n" +
	"\bchain_id\x18\x01 \x01(\tR\achainId\x12)\n" +
	"\x10finalized_height\x18\x02 \x01(\x04R\x0ffinalizedHeight\x124\n" +
	"\x16celestia_height_offset\x18\x03 \x01(\x04R\x14celestiaHeightOffsetB@Z>github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1b\x06proto3"

var (
	file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescOnce sync.Once
	file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescData []byte
)

func file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescGZIP() []byte {
	file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescOnce.Do(func() {
		file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDesc), len(file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDesc)))
	})
	return file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDescData
}

var file_blobcast_rollupapis_v1_get_chain_info_response_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_blobcast_rollupapis_v1_get_chain_info_response_proto_goTypes = []any{
	(*GetChainInfoResponse)(nil), // 0: blobcast.rollupapis.v1.GetChainInfoResponse
}
var file_blobcast_rollupapis_v1_get_chain_info_response_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_blobcast_rollupapis_v1_get_chain_info_response_proto_init() }
func file_blobcast_rollupapis_v1_get_chain_info_response_proto_init() {
	if File_blobcast_rollupapis_v1_get_chain_info_response_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDesc), len(file_blobcast_rollupapis_v1_get_chain_info_response_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_blobcast_rollupapis_v1_get_chain_info_response_proto_goTypes,
		DependencyIndexes: file_blobcast_rollupapis_v1_get_chain_info_response_proto_depIdxs,
		MessageInfos:      file_blobcast_rollupapis_v1_get_chain_info_response_proto_msgTypes,
	}.Build()
	File_blobcast_rollupapis_v1_get_chain_info_response_proto = out.File
	file_blobcast_rollupapis_v1_get_chain_info_response_proto_goTypes = nil
	file_blobcast_rollupapis_v1_get_chain_info_response_proto_depIdxs = nil
}

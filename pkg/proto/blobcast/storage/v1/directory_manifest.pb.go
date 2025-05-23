// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: blobcast/storage/v1/directory_manifest.proto

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

type DirectoryManifest struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	ManifestVersion string                 `protobuf:"bytes,1,opt,name=manifest_version,json=manifestVersion,proto3" json:"manifest_version,omitempty"`
	DirectoryName   string                 `protobuf:"bytes,2,opt,name=directory_name,json=directoryName,proto3" json:"directory_name,omitempty"`
	Files           []*FileReference       `protobuf:"bytes,3,rep,name=files,proto3" json:"files,omitempty"`
	DirectoryHash   []byte                 `protobuf:"bytes,4,opt,name=directory_hash,json=directoryHash,proto3" json:"directory_hash,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *DirectoryManifest) Reset() {
	*x = DirectoryManifest{}
	mi := &file_blobcast_storage_v1_directory_manifest_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DirectoryManifest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DirectoryManifest) ProtoMessage() {}

func (x *DirectoryManifest) ProtoReflect() protoreflect.Message {
	mi := &file_blobcast_storage_v1_directory_manifest_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DirectoryManifest.ProtoReflect.Descriptor instead.
func (*DirectoryManifest) Descriptor() ([]byte, []int) {
	return file_blobcast_storage_v1_directory_manifest_proto_rawDescGZIP(), []int{0}
}

func (x *DirectoryManifest) GetManifestVersion() string {
	if x != nil {
		return x.ManifestVersion
	}
	return ""
}

func (x *DirectoryManifest) GetDirectoryName() string {
	if x != nil {
		return x.DirectoryName
	}
	return ""
}

func (x *DirectoryManifest) GetFiles() []*FileReference {
	if x != nil {
		return x.Files
	}
	return nil
}

func (x *DirectoryManifest) GetDirectoryHash() []byte {
	if x != nil {
		return x.DirectoryHash
	}
	return nil
}

var File_blobcast_storage_v1_directory_manifest_proto protoreflect.FileDescriptor

const file_blobcast_storage_v1_directory_manifest_proto_rawDesc = "" +
	"\n" +
	",blobcast/storage/v1/directory_manifest.proto\x12\x13blobcast.storage.v1\x1a(blobcast/storage/v1/file_reference.proto\"\xc6\x01\n" +
	"\x11DirectoryManifest\x12)\n" +
	"\x10manifest_version\x18\x01 \x01(\tR\x0fmanifestVersion\x12%\n" +
	"\x0edirectory_name\x18\x02 \x01(\tR\rdirectoryName\x128\n" +
	"\x05files\x18\x03 \x03(\v2\".blobcast.storage.v1.FileReferenceR\x05files\x12%\n" +
	"\x0edirectory_hash\x18\x04 \x01(\fR\rdirectoryHashB=Z;github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1b\x06proto3"

var (
	file_blobcast_storage_v1_directory_manifest_proto_rawDescOnce sync.Once
	file_blobcast_storage_v1_directory_manifest_proto_rawDescData []byte
)

func file_blobcast_storage_v1_directory_manifest_proto_rawDescGZIP() []byte {
	file_blobcast_storage_v1_directory_manifest_proto_rawDescOnce.Do(func() {
		file_blobcast_storage_v1_directory_manifest_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_blobcast_storage_v1_directory_manifest_proto_rawDesc), len(file_blobcast_storage_v1_directory_manifest_proto_rawDesc)))
	})
	return file_blobcast_storage_v1_directory_manifest_proto_rawDescData
}

var file_blobcast_storage_v1_directory_manifest_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_blobcast_storage_v1_directory_manifest_proto_goTypes = []any{
	(*DirectoryManifest)(nil), // 0: blobcast.storage.v1.DirectoryManifest
	(*FileReference)(nil),     // 1: blobcast.storage.v1.FileReference
}
var file_blobcast_storage_v1_directory_manifest_proto_depIdxs = []int32{
	1, // 0: blobcast.storage.v1.DirectoryManifest.files:type_name -> blobcast.storage.v1.FileReference
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_blobcast_storage_v1_directory_manifest_proto_init() }
func file_blobcast_storage_v1_directory_manifest_proto_init() {
	if File_blobcast_storage_v1_directory_manifest_proto != nil {
		return
	}
	file_blobcast_storage_v1_file_reference_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_blobcast_storage_v1_directory_manifest_proto_rawDesc), len(file_blobcast_storage_v1_directory_manifest_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_blobcast_storage_v1_directory_manifest_proto_goTypes,
		DependencyIndexes: file_blobcast_storage_v1_directory_manifest_proto_depIdxs,
		MessageInfos:      file_blobcast_storage_v1_directory_manifest_proto_msgTypes,
	}.Build()
	File_blobcast_storage_v1_directory_manifest_proto = out.File
	file_blobcast_storage_v1_directory_manifest_proto_goTypes = nil
	file_blobcast_storage_v1_directory_manifest_proto_depIdxs = nil
}

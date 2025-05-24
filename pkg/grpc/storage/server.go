package storage

import (
	"context"

	"github.com/forma-dev/blobcast/pkg/api/node"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

type StorageServiceServer struct {
	pbStorageapisV1.UnimplementedStorageServiceServer
}

func NewStorageServiceServer() *StorageServiceServer {
	return &StorageServiceServer{}
}

func (s *StorageServiceServer) GetDirectoryManifest(
	ctx context.Context,
	req *pbStorageapisV1.GetDirectoryManifestRequest,
) (*pbStorageapisV1.GetDirectoryManifestResponse, error) {
	dirManifestId := types.BlobIdentifierFromProto(req.Id)
	dirManifest, err := node.GetDirectoryManifest(ctx, dirManifestId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting directory manifest: %v", err)
	}

	return &pbStorageapisV1.GetDirectoryManifestResponse{Manifest: dirManifest}, nil
}

func (s *StorageServiceServer) GetFileManifest(
	ctx context.Context,
	req *pbStorageapisV1.GetFileManifestRequest,
) (*pbStorageapisV1.GetFileManifestResponse, error) {
	fileManifestId := types.BlobIdentifierFromProto(req.Id)
	fileManifest, err := node.GetFileManifest(ctx, fileManifestId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting file manifest: %v", err)
	}

	return &pbStorageapisV1.GetFileManifestResponse{Manifest: fileManifest}, nil
}

func (s *StorageServiceServer) GetFileData(
	ctx context.Context,
	req *pbStorageapisV1.GetFileDataRequest,
) (*pbStorageapisV1.GetFileDataResponse, error) {
	fileManifestId := types.BlobIdentifierFromProto(req.Id)
	fileData, err := node.GetFileData(ctx, fileManifestId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting file data: %v", err)
	}

	return &pbStorageapisV1.GetFileDataResponse{Data: fileData}, nil
}

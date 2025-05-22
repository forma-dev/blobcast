package storage

import (
	"context"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

type StorageServiceServer struct {
	pbStorageapisV1.UnimplementedStorageServiceServer

	da celestia.BlobStore
}

func NewStorageServiceServer(da celestia.BlobStore) *StorageServiceServer {
	return &StorageServiceServer{da: da}
}

func (s *StorageServiceServer) GetDirectoryManifest(
	ctx context.Context,
	req *pbStorageapisV1.GetDirectoryManifestRequest,
) (*pbStorageapisV1.GetDirectoryManifestResponse, error) {
	dirManifestId := types.BlobIdentifierFromProto(req.Id)
	dirManifest, err := sync.GetDirectoryManifest(ctx, s.da, dirManifestId)
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
	fileManifest, err := sync.GetFileManifest(ctx, s.da, fileManifestId)
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
	fileData, err := sync.GetFileData(ctx, s.da, fileManifestId, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting file data: %v", err)
	}

	return &pbStorageapisV1.GetFileDataResponse{Data: fileData}, nil
}

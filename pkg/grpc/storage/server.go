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

func (s *StorageServiceServer) BatchGetFileManifest(
	ctx context.Context,
	req *pbStorageapisV1.BatchGetFileManifestRequest,
) (*pbStorageapisV1.BatchGetFileManifestResponse, error) {
	if len(req.Ids) > 1000 {
		return nil, status.Errorf(codes.InvalidArgument, "batch size must be less than 1000")
	}

	manifests := make([]*pbStorageapisV1.FileManifest, 0)
	for _, id := range req.Ids {
		fileManifestId, err := types.BlobIdentifierFromString(id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid manifest ID: %v", err)
		}
		fileManifest, err := node.GetFileManifest(ctx, fileManifestId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting file manifest: %v", err)
		}
		manifests = append(manifests, &pbStorageapisV1.FileManifest{
			Id:       id,
			Manifest: fileManifest,
		})
	}

	return &pbStorageapisV1.BatchGetFileManifestResponse{Manifests: manifests}, nil
}

func (s *StorageServiceServer) GetDirectoryManifest(
	ctx context.Context,
	req *pbStorageapisV1.GetDirectoryManifestRequest,
) (*pbStorageapisV1.GetDirectoryManifestResponse, error) {
	dirManifestId, err := types.BlobIdentifierFromString(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid manifest ID: %v", err)
	}
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
	fileManifestId, err := types.BlobIdentifierFromString(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid manifest ID: %v", err)
	}
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
	fileManifestId, err := types.BlobIdentifierFromString(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid manifest ID: %v", err)
	}
	fileData, err := node.GetFileData(ctx, fileManifestId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting file data: %v", err)
	}

	return &pbStorageapisV1.GetFileDataResponse{Data: fileData}, nil
}

func (s *StorageServiceServer) GetChunkReference(
	ctx context.Context,
	req *pbStorageapisV1.GetChunkReferenceRequest,
) (*pbStorageapisV1.GetChunkReferenceResponse, error) {
	chunkId, err := types.BlobIdentifierFromString(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chunk ID: %v", err)
	}
	chunkReference, err := node.GetChunkReference(ctx, chunkId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting chunk reference: %v", err)
	}

	return &pbStorageapisV1.GetChunkReferenceResponse{Reference: chunkReference}, nil
}

func (s *StorageServiceServer) GetChunkData(
	ctx context.Context,
	req *pbStorageapisV1.GetChunkDataRequest,
) (*pbStorageapisV1.GetChunkDataResponse, error) {
	chunkId, err := types.BlobIdentifierFromString(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chunk ID: %v", err)
	}
	chunkData, err := node.GetChunkData(ctx, chunkId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting chunk data: %v", err)
	}

	return &pbStorageapisV1.GetChunkDataResponse{Data: chunkData}, nil
}

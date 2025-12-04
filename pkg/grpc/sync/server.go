package sync

import (
	"github.com/forma-dev/blobcast/pkg/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
	pbSyncapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/syncapis/v1"
)

type SyncServiceServer struct {
	pbSyncapisV1.UnimplementedSyncServiceServer
	chainState *state.ChainState
}

func NewSyncServiceServer(chainState *state.ChainState) *SyncServiceServer {
	return &SyncServiceServer{
		chainState: chainState,
	}
}

func (s *SyncServiceServer) StreamBlocks(req *pbSyncapisV1.StreamBlocksRequest, stream pbSyncapisV1.SyncService_StreamBlocksServer) error {
	startHeight := req.StartHeight
	endHeight := req.EndHeight

	if endHeight == 0 {
		finalizedHeight, err := s.chainState.FinalizedHeight()
		if err != nil {
			return status.Errorf(codes.Internal, "error getting finalized height: %v", err)
		}
		endHeight = finalizedHeight
	}

	if startHeight > endHeight {
		return status.Errorf(codes.InvalidArgument, "start height must be less than or equal to end height")
	}

	for height := startHeight; height <= endHeight; height++ {
		block, err := s.chainState.GetBlock(height)
		if err != nil {
			return status.Errorf(codes.Internal, "error getting block at height %d: %v", height, err)
		}
		if block == nil {
			return status.Errorf(codes.NotFound, "block at height %d not found", height)
		}

		// Only send MMR snapshot if one exists at this exact height
		// (empty blocks don't have their own MMR, they inherit from previous blocks)
		var mmrSnapshot []byte
		if height > 0 {
			hasMMR, err := s.chainState.HasStateMMRAtHeight(height)
			if err != nil {
				return status.Errorf(codes.Internal, "error checking state mmr at height %d: %v", height, err)
			}

			if hasMMR {
				mmr, err := s.chainState.GetStateMMR(height)
				if err != nil {
					return status.Errorf(codes.Internal, "error getting state mmr at height %d: %v", height, err)
				}
				mmrSnapshot = mmr.Snapshot()
			}
		}

		chunks := make([]*pbSyncapisV1.ChunkWithHash, len(block.Body.Chunks))
		files := make([]*pbStorageV1.FileManifest, len(block.Body.Files))
		dirs := make([]*pbStorageV1.DirectoryManifest, len(block.Body.Dirs))

		for i, chunk := range block.Body.Chunks {
			chunkData, exists, err := s.chainState.GetChunk(state.HashKey(chunk.Commitment))
			if err != nil {
				return status.Errorf(codes.Internal, "error getting chunk data at height %d: %v", height, err)
			}
			if !exists {
				return status.Errorf(codes.NotFound, "chunk at height %d not found", height)
			}

			chunkHash, exists, err := s.chainState.GetChunkHash(state.HashKey(chunk.Commitment))
			if err != nil {
				return status.Errorf(codes.Internal, "error getting chunk hash at height %d: %v", height, err)
			}
			if !exists {
				return status.Errorf(codes.NotFound, "chunk hash at height %d not found", height)
			}

			chunks[i] = &pbSyncapisV1.ChunkWithHash{
				ChunkData: chunkData,
				ChunkHash: chunkHash[:],
			}
		}

		for i, fileId := range block.Body.Files {
			fileManifest, exists, err := s.chainState.GetFileManifest(fileId)
			if err != nil {
				return status.Errorf(codes.Internal, "error getting file manifest at height %d: %v", height, err)
			}
			if !exists {
				return status.Errorf(codes.NotFound, "file manifest at height %d not found", height)
			}

			files[i] = fileManifest
		}

		for i, dirId := range block.Body.Dirs {
			dirManifest, exists, err := s.chainState.GetDirectoryManifest(dirId)
			if err != nil {
				return status.Errorf(codes.Internal, "error getting directory manifest at height %d: %v", height, err)
			}
			if !exists {
				return status.Errorf(codes.NotFound, "directory manifest at height %d not found", height)
			}

			dirs[i] = dirManifest
		}

		blockResp := &pbSyncapisV1.StreamBlocksResponse{
			Block:       block.Proto(),
			Chunks:      chunks,
			Files:       files,
			Dirs:        dirs,
			MmrSnapshot: mmrSnapshot,
		}

		if err := stream.Send(blockResp); err != nil {
			return status.Errorf(codes.Internal, "error sending block at height %d: %v", height, err)
		}

		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
	}

	return nil
}

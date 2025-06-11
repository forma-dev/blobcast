package node

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

func GetChunkData(ctx context.Context, chunkId *types.BlobIdentifier) ([]byte, error) {
	slog.Debug("Getting chunk data", "chunk_id", chunkId)

	chainState, err := state.GetChainState()
	if err != nil {
		return nil, fmt.Errorf("error getting chain state: %v", err)
	}

	chunkData, exists, err := chainState.GetChunk(state.HashKey(chunkId.Commitment))
	if err != nil {
		return nil, fmt.Errorf("error getting chunk data from state: %v", err)
	}

	if !exists {
		return nil, fmt.Errorf("chunk not found in state")
	}

	return chunkData, nil
}

func GetChunkReference(ctx context.Context, chunkId *types.BlobIdentifier) (*pbStorageV1.ChunkReference, error) {
	slog.Debug("Getting chunk reference", "chunk_id", chunkId)

	chainState, err := state.GetChainState()
	if err != nil {
		return nil, fmt.Errorf("error getting chain state: %v", err)
	}

	chunkHash, exists, err := chainState.GetChunkHash(state.HashKey(chunkId.Commitment))
	if err != nil {
		return nil, fmt.Errorf("error getting chunk hash: %v", err)
	}

	if !exists {
		return nil, fmt.Errorf("chunk hash not found in state")
	}

	chunkData, exists, err := chainState.GetChunk(state.HashKey(chunkId.Commitment))
	if err != nil {
		return nil, fmt.Errorf("error getting chunk data: %v", err)
	}

	if !exists {
		return nil, fmt.Errorf("chunk not found in state")
	}

	return &pbStorageV1.ChunkReference{
		Id:        chunkId.Proto(),
		ChunkHash: chunkHash.Bytes(),
		ChunkSize: uint64(len(chunkData)),
	}, nil
}

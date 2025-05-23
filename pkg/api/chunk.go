package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/state"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

func GetChunkData(ctx context.Context, chunkRef *pbStorageV1.ChunkReference) ([]byte, error) {
	slog.Debug("Getting chunk data",
		"block_height", chunkRef.Id.Height,
		"namespace", hex.EncodeToString(chunkRef.Id.Commitment),
		"commitment_hash", hex.EncodeToString(chunkRef.ChunkHash),
		"chunk_hash", hex.EncodeToString(chunkRef.ChunkHash),
	)

	chainState, err := state.GetChainState()
	if err != nil {
		return nil, fmt.Errorf("error getting chain state: %v", err)
	}

	chunkData, exists, err := chainState.GetChunk(state.HashKey(chunkRef.Id.Commitment))
	if err != nil {
		return nil, fmt.Errorf("error getting chunk data from state: %v", err)
	}

	if !exists {
		return nil, fmt.Errorf("chunk not found in state")
	}

	slog.Debug("Verifying chunk hash")
	chunkHash := crypto.HashBytes(chunkData)
	if !chunkHash.EqualBytes(chunkRef.ChunkHash) {
		return nil, fmt.Errorf("chunk hash verification failed")
	}

	return chunkData, nil
}

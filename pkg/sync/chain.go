package sync

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/celestiaorg/celestia-node/blob"
	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

type BlobcastChain struct {
	chainID       string
	celestiaDA    celestia.BlobStore
	chainState    *state.ChainState
	chunkState    *state.ChunkState
	manifestState *state.ManifestState
}

func NewBlobcastChain(ctx context.Context, celestiaDA celestia.BlobStore) (*BlobcastChain, error) {
	chainState, err := state.GetChainState()
	if err != nil {
		return nil, fmt.Errorf("error getting chain state: %v", err)
	}

	chainID, err := chainState.ChainID()
	if err != nil {
		return nil, fmt.Errorf("error getting chain id: %v", err)
	}

	chunkState, err := state.GetChunkState()
	if err != nil {
		return nil, fmt.Errorf("error getting chunk state: %v", err)
	}

	manifestState, err := state.GetManifestState()
	if err != nil {
		return nil, fmt.Errorf("error getting manifest state: %v", err)
	}
	return &BlobcastChain{
		chainID:       chainID,
		celestiaDA:    celestiaDA,
		chainState:    chainState,
		chunkState:    chunkState,
		manifestState: manifestState,
	}, nil
}

func (bc *BlobcastChain) SyncChain(ctx context.Context) (err error) {
	defer bc.chainState.Close()

	nextHeight := uint64(0)

	celestiaHeightOffset, err := bc.chainState.CelestiaHeightOffset()
	if err != nil {
		return fmt.Errorf("error getting celestia height offset: %v", err)
	}

	if celestiaHeightOffset == 0 {
		// todo
	}

	finalizedHeight, err := bc.chainState.FinalizedHeight()
	if err != nil {
		return fmt.Errorf("error getting finalized height: %v", err)
	}

	if finalizedHeight > 0 {
		nextHeight = finalizedHeight + 1
	}

	var prevBlock *types.Block

	// genesis block
	if nextHeight == 0 {
		genesisBlock, err := bc.CreateGenesisBlock(ctx, celestiaHeightOffset)
		if err != nil {
			return fmt.Errorf("error creating genesis block: %v", err)
		}
		prevBlock = genesisBlock
		nextHeight++
	}

	// Get the latest Celestia height
	latestCelestiaHeight, err := bc.celestiaDA.LatestHeight(ctx)
	if err != nil {
		return fmt.Errorf("error getting latest celestia height: %v", err)
	}

	// perform historical catchup
	slog.Info("starting historical sync", "height", nextHeight, "latest_celestia_height", latestCelestiaHeight)
	for {
		// refresh the latest height when we've caught up the last fetched height
		if nextHeight+celestiaHeightOffset >= latestCelestiaHeight {
			latestCelestiaHeight, err = bc.celestiaDA.LatestHeight(ctx)
			if err != nil {
				return fmt.Errorf("error getting latest celestia height: %v", err)
			}

			if nextHeight+celestiaHeightOffset > latestCelestiaHeight {
				slog.Info("historical sync complete", "height", nextHeight-1)
				break
			}
		}

		block, err := bc.ProduceBlock(ctx, nextHeight, nextHeight+celestiaHeightOffset)
		if err != nil {
			return fmt.Errorf("error producing block: %v", err)
		}

		if err := bc.ApplyBlock(ctx, block, prevBlock); err != nil {
			return fmt.Errorf("error applying block: %v", err)
		}

		if err := bc.FinalizeBlock(ctx, nextHeight); err != nil {
			return fmt.Errorf("error finalizing block: %v", err)
		}

		prevBlock = block
		nextHeight++
	}

	// subscribe to new blocks
	slog.Info("starting live sync")
	heads, err := bc.celestiaDA.SubscribeToNewHeaders(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing to new celestia headers: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case head := <-heads:
			slog.Info("new celestia block", "height", head.Height)
			blockHeight := head.Height - celestiaHeightOffset
			block, err := bc.ProduceBlock(ctx, blockHeight, head.Height)
			if err != nil {
				return fmt.Errorf("error producing block: %v", err)
			}

			if err := bc.ApplyBlock(ctx, block, prevBlock); err != nil {
				return fmt.Errorf("error applying block: %v", err)
			}

			if err := bc.FinalizeBlock(ctx, blockHeight); err != nil {
				return fmt.Errorf("error finalizing block: %v", err)
			}

			prevBlock = block
		}
	}
}

func (bc *BlobcastChain) CreateGenesisBlock(ctx context.Context, celestiaHeight uint64) (*types.Block, error) {
	slog.Debug("creating genesis block", "celestia_height", celestiaHeight)

	celestiaHeader, err := bc.celestiaDA.GetHeader(ctx, celestiaHeight)
	if err != nil {
		return nil, fmt.Errorf("error getting celestia header: %v", err)
	}

	block := types.NewBlock(nil, nil, nil)
	block.Header.ChainID = bc.chainID
	block.Header.Height = 0
	block.Header.CelestiaBlockHeight = celestiaHeight
	block.Header.Timestamp = celestiaHeader.Time()

	slog.Info("created genesis block", "height", block.Height(), "hash", block.Hash())

	if err := bc.chainState.PutBlock(block.Height(), block); err != nil {
		return nil, fmt.Errorf("error applying genesis block: %v", err)
	}

	return block, nil
}

func (bc *BlobcastChain) ProduceBlock(ctx context.Context, height uint64, celestiaHeight uint64) (*types.Block, error) {
	block, err := bc.ProduceBlockFromCelestiaHeight(ctx, celestiaHeight)
	if err != nil {
		return nil, fmt.Errorf("error producing block from celestia height: %v", err)
	}

	celestiaHeader, err := bc.celestiaDA.GetHeader(ctx, celestiaHeight)
	if err != nil {
		return nil, fmt.Errorf("error getting celestia header: %v", err)
	}

	block.Header.ChainID = bc.chainID
	block.Header.Height = height
	block.Header.CelestiaBlockHeight = celestiaHeight
	block.Header.Timestamp = celestiaHeader.Time()

	return block, nil
}

func (bc *BlobcastChain) ApplyBlock(ctx context.Context, block *types.Block, prevBlock *types.Block) error {
	slog.Debug("applying block", "height", block.Height())

	if block.Height() > 0 {
		if prevBlock == nil {
			var err error
			prevBlock, err = bc.chainState.GetBlock(block.Header.Height - 1)
			if err != nil {
				return fmt.Errorf("error getting previous block: %v", err)
			}
		}

		if prevBlock == nil {
			return fmt.Errorf("previous block not found")
		}

		if prevBlock.Height() != block.Height()-1 {
			return fmt.Errorf("previous block height mismatch")
		}

		block.Header.ParentHash = prevBlock.Hash()
	}

	// update state root
	stateMMR, err := bc.chainState.GetStateMMR(block.Height() - 1)
	if err != nil {
		return fmt.Errorf("error getting state mmr: %v", err)
	}

	// add the dirs, files, and chunks merkle roots to the state root, if they exist
	if !block.Header.DirsRoot.IsZero() {
		stateMMR.AddLeaf(block.Header.DirsRoot.Bytes())
	}
	if !block.Header.FilesRoot.IsZero() {
		stateMMR.AddLeaf(block.Header.FilesRoot.Bytes())
	}
	if !block.Header.ChunksRoot.IsZero() {
		stateMMR.AddLeaf(block.Header.ChunksRoot.Bytes())
	}

	block.Header.StateRoot = stateMMR.Root()

	if err := bc.chainState.PutStateMMR(block.Height(), stateMMR); err != nil {
		return fmt.Errorf("error putting state mmr: %v", err)
	}

	slog.Debug("block header",
		"height", block.Height(),
		"hash", block.Hash(),
		"parent_hash", block.Header.ParentHash,
		"chain_id", block.Header.ChainID,
		"timestamp", block.Header.Timestamp,
		"celestia_height", block.Header.CelestiaBlockHeight,
		"dirs_root", block.Header.DirsRoot,
		"files_root", block.Header.FilesRoot,
		"chunks_root", block.Header.ChunksRoot,
		"state_root", block.Header.StateRoot,
	)
	slog.Info("applied block", "height", block.Height(), "hash", block.Hash(), "parent_hash", block.Header.ParentHash)

	return bc.chainState.PutBlock(block.Height(), block)
}

func (bc *BlobcastChain) FinalizeBlock(ctx context.Context, height uint64) error {
	slog.Debug("finalizing block", "height", height)
	return bc.chainState.SetFinalizedHeight(height)
}

func (bc *BlobcastChain) ProduceBlockFromCelestiaHeight(ctx context.Context, height uint64) (*types.Block, error) {
	chunks, files, dirs, err := bc.SyncCelestiaHeight(ctx, height)
	if err != nil {
		return nil, fmt.Errorf("error syncing celestia height: %v", err)
	}

	return types.NewBlock(dirs, files, chunks), nil
}

func (bc *BlobcastChain) SyncCelestiaHeight(
	ctx context.Context,
	height uint64,
) (chunks []*types.BlobIdentifier, files []*types.BlobIdentifier, dirs []*types.BlobIdentifier, err error) {
	blobs, err := bc.celestiaDA.GetBlobs(ctx, height)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting blobs from Celestia: %v", err)
	}
	return bc.SyncBlobs(ctx, height, blobs)
}

func (bc *BlobcastChain) SyncBlobs(
	ctx context.Context,
	height uint64,
	blobs []*blob.Blob,
) (chunks []*types.BlobIdentifier, files []*types.BlobIdentifier, dirs []*types.BlobIdentifier, err error) {
	for _, blob := range blobs {
		var envelope pbRollupV1.BlobcastEnvelope
		err := proto.Unmarshal(blob.Data(), &envelope)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error unmarshalling blob envelope: %v", err)
		}

		blobId := &types.BlobIdentifier{
			Height:     height,
			Commitment: blob.Commitment,
		}

		var syncErr error
		switch record := envelope.Payload.(type) {
		case *pbRollupV1.BlobcastEnvelope_ChunkData:
			if syncErr = bc.SyncChunk(ctx, height, blob, record.ChunkData); syncErr == nil {
				chunks = append(chunks, blobId)
			}
		case *pbRollupV1.BlobcastEnvelope_FileManifest:
			if syncErr = bc.SyncFileManifest(ctx, height, blob, record.FileManifest); syncErr == nil {
				files = append(files, blobId)
			}
		case *pbRollupV1.BlobcastEnvelope_DirectoryManifest:
			if syncErr = bc.SyncDirectoryManifest(ctx, height, blob, record.DirectoryManifest); syncErr == nil {
				dirs = append(dirs, blobId)
			}
		default:
			return nil, nil, nil, fmt.Errorf("unknown blob type: %v", blob.Data())
		}

		if syncErr != nil {
			return nil, nil, nil, fmt.Errorf("error syncing blob: %v", syncErr)
		}
	}
	return chunks, files, dirs, nil
}

func (bc *BlobcastChain) SyncChunk(ctx context.Context, height uint64, blob *blob.Blob, chunkData *pbStorageV1.ChunkData) error {
	slog.Debug("syncing chunk",
		"height", height,
		"commitment", hex.EncodeToString(blob.Commitment),
		"size", len(chunkData.ChunkData),
	)

	err := bc.chunkState.Set(state.ChunkKey(blob.Commitment), chunkData.ChunkData)
	if err != nil {
		return fmt.Errorf("error saving chunk data: %v", err)
	}
	return bc.chunkState.MarkSeen(state.ChunkKey(blob.Commitment), height)
}

func (bc *BlobcastChain) SyncFileManifest(ctx context.Context, height uint64, blob *blob.Blob, fileManifest *pbStorageV1.FileManifest) error {
	slog.Debug("syncing file manifest",
		"height", height,
		"commitment", hex.EncodeToString(blob.Commitment),
		"file_name", fileManifest.FileName,
		"file_size", fileManifest.FileSize,
		"mime_type", fileManifest.MimeType,
		"num_chunks", len(fileManifest.Chunks),
	)

	manifestId := &types.BlobIdentifier{
		Height:     height,
		Commitment: blob.Commitment,
	}

	// all chunks for this file manifest must be seen
	for _, chunk := range fileManifest.Chunks {
		seen, err := bc.chunkState.SeenAtHeight(state.ChunkKey(chunk.Id.Commitment), height)
		if err != nil {
			return fmt.Errorf("error getting chunk seen height: %v", err)
		}
		if !seen {
			chunkId := types.BlobIdentifierFromProto(chunk.Id)
			slog.Debug("file chunk not seen", "chunk_id", chunkId, "commitment", hex.EncodeToString(chunk.Id.Commitment), "file_id", manifestId)
			slog.Warn("file manifest will not be inlcuded due to missing chunk", "file_id", manifestId)
			return nil
		}
	}

	return bc.manifestState.PutFileManifest(manifestId, fileManifest)
}

func (bc *BlobcastChain) SyncDirectoryManifest(ctx context.Context, height uint64, blob *blob.Blob, directoryManifest *pbStorageV1.DirectoryManifest) error {
	slog.Debug("syncing directory manifest",
		"height", height,
		"name", directoryManifest.DirectoryName,
		"commitment", hex.EncodeToString(blob.Commitment),
		"num_files", len(directoryManifest.Files),
		"dir_hash", hex.EncodeToString(directoryManifest.DirectoryHash),
	)

	manifestId := &types.BlobIdentifier{
		Height:     height,
		Commitment: blob.Commitment,
	}

	// all files for this directory manifest must be seen
	for _, file := range directoryManifest.Files {
		_, found, err := bc.manifestState.GetFileManifest(types.BlobIdentifierFromProto(file.Id))
		if err != nil {
			return fmt.Errorf("error getting file manifest: %v", err)
		}
		if !found {
			fileId := types.BlobIdentifierFromProto(file.Id)
			slog.Debug("file manifest not seen", "file_id", fileId, "dir_id", manifestId)
			slog.Warn("directory manifest will not be included due to missing file manifest", "dir_id", manifestId)
			return nil
		}
	}

	return bc.manifestState.PutDirectoryManifest(manifestId, directoryManifest)
}

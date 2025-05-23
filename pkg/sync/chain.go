package sync

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/celestiaorg/celestia-node/blob"
	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/crypto/merkle"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

type BlobcastChain struct {
	chainID    string
	celestiaDA celestia.BlobStore
	chainState *state.ChainState
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

	return &BlobcastChain{
		chainID:    chainID,
		celestiaDA: celestiaDA,
		chainState: chainState,
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
		tx, err := bc.chainState.BeginTransaction()
		if err != nil {
			return fmt.Errorf("error beginning chain state transaction: %v", err)
		}

		genesisBlock, err := bc.CreateGenesisBlock(ctx, tx, celestiaHeightOffset)
		if err != nil {
			if abortErr := tx.Abort(); abortErr != nil {
				slog.Error("error aborting transaction", "error", abortErr)
			}
			return fmt.Errorf("error creating genesis block: %v", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing transaction: %v", err)
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

		tx, err := bc.chainState.BeginTransaction()
		if err != nil {
			return fmt.Errorf("error beginning chain state transaction: %v", err)
		}

		block, err := bc.ProduceBlock(ctx, tx, nextHeight, nextHeight+celestiaHeightOffset)
		if err != nil {
			if abortErr := tx.Abort(); abortErr != nil {
				slog.Error("error aborting transaction", "error", abortErr)
			}
			return fmt.Errorf("error producing block: %v", err)
		}

		if err := bc.ApplyBlock(ctx, tx, block, prevBlock); err != nil {
			if abortErr := tx.Abort(); abortErr != nil {
				slog.Error("error aborting transaction", "error", abortErr)
			}
			return fmt.Errorf("error applying block: %v", err)
		}

		if err := bc.FinalizeBlock(ctx, tx, nextHeight); err != nil {
			if abortErr := tx.Abort(); abortErr != nil {
				slog.Error("error aborting transaction", "error", abortErr)
			}
			return fmt.Errorf("error finalizing block: %v", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing transaction: %v", err)
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
			latestCelestiaHeight = head.Height
			for {
				if nextHeight+celestiaHeightOffset > latestCelestiaHeight {
					break
				}

				tx, err := bc.chainState.BeginTransaction()
				if err != nil {
					return fmt.Errorf("error beginning chain state transaction: %v", err)
				}

				block, err := bc.ProduceBlock(ctx, tx, nextHeight, nextHeight+celestiaHeightOffset)
				if err != nil {
					if abortErr := tx.Abort(); abortErr != nil {
						slog.Error("error aborting transaction", "error", abortErr)
					}
					return fmt.Errorf("error producing block: %v", err)
				}

				if err := bc.ApplyBlock(ctx, tx, block, prevBlock); err != nil {
					if abortErr := tx.Abort(); abortErr != nil {
						slog.Error("error aborting transaction", "error", abortErr)
					}
					return fmt.Errorf("error applying block: %v", err)
				}

				if err := bc.FinalizeBlock(ctx, tx, nextHeight); err != nil {
					if abortErr := tx.Abort(); abortErr != nil {
						slog.Error("error aborting transaction", "error", abortErr)
					}
					return fmt.Errorf("error finalizing block: %v", err)
				}

				if err := tx.Commit(); err != nil {
					return fmt.Errorf("error committing transaction: %v", err)
				}

				prevBlock = block
				nextHeight++
			}
		}
	}
}

func (bc *BlobcastChain) CreateGenesisBlock(ctx context.Context, tx *state.ChainStateTransaction, celestiaHeight uint64) (*types.Block, error) {
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

	if err := tx.PutBlock(block.Height(), block); err != nil {
		return nil, fmt.Errorf("error applying genesis block: %v", err)
	}

	return block, nil
}

func (bc *BlobcastChain) ProduceBlock(ctx context.Context, tx *state.ChainStateTransaction, height uint64, celestiaHeight uint64) (*types.Block, error) {
	block, err := bc.ProduceBlockFromCelestiaHeight(ctx, tx, celestiaHeight)
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

func (bc *BlobcastChain) ApplyBlock(ctx context.Context, tx *state.ChainStateTransaction, block *types.Block, prevBlock *types.Block) error {
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

	if err := tx.PutStateMMR(block.Height(), stateMMR); err != nil {
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

	return tx.PutBlock(block.Height(), block)
}

func (bc *BlobcastChain) FinalizeBlock(ctx context.Context, tx *state.ChainStateTransaction, height uint64) error {
	slog.Debug("finalizing block", "height", height)
	return tx.SetFinalizedHeight(height)
}

func (bc *BlobcastChain) ProduceBlockFromCelestiaHeight(ctx context.Context, tx *state.ChainStateTransaction, height uint64) (*types.Block, error) {
	chunks, files, dirs, err := bc.SyncCelestiaHeight(ctx, tx, height)
	if err != nil {
		return nil, fmt.Errorf("error syncing celestia height: %v", err)
	}

	return types.NewBlock(dirs, files, chunks), nil
}

func (bc *BlobcastChain) SyncCelestiaHeight(
	ctx context.Context,
	tx *state.ChainStateTransaction,
	height uint64,
) (chunks []*types.BlobIdentifier, files []*types.BlobIdentifier, dirs []*types.BlobIdentifier, err error) {
	blobs, err := bc.celestiaDA.GetBlobs(ctx, height)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting blobs from Celestia: %v", err)
	}
	return bc.SyncBlobs(ctx, tx, height, blobs)
}

func (bc *BlobcastChain) SyncBlobs(
	ctx context.Context,
	tx *state.ChainStateTransaction,
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
			if syncErr = bc.SyncChunk(ctx, tx, height, blob, record.ChunkData); syncErr == nil {
				chunks = append(chunks, blobId)
			}
		case *pbRollupV1.BlobcastEnvelope_FileManifest:
			if syncErr = bc.SyncFileManifest(ctx, tx, height, blob, record.FileManifest); syncErr == nil {
				files = append(files, blobId)
			}
		case *pbRollupV1.BlobcastEnvelope_DirectoryManifest:
			if syncErr = bc.SyncDirectoryManifest(ctx, tx, height, blob, record.DirectoryManifest); syncErr == nil {
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

func (bc *BlobcastChain) SyncChunk(
	ctx context.Context,
	tx *state.ChainStateTransaction,
	height uint64,
	blob *blob.Blob,
	chunkData *pbStorageV1.ChunkData,
) error {
	slog.Debug("syncing chunk",
		"height", height,
		"commitment", hex.EncodeToString(blob.Commitment),
		"size", len(chunkData.ChunkData),
	)
	return tx.PutChunk(state.HashKey(blob.Commitment), chunkData.ChunkData)
}

func (bc *BlobcastChain) SyncFileManifest(
	ctx context.Context,
	tx *state.ChainStateTransaction,
	height uint64,
	blob *blob.Blob,
	fileManifest *pbStorageV1.FileManifest,
) error {
	manifestId := &types.BlobIdentifier{
		Height:     height,
		Commitment: blob.Commitment,
	}

	slog.Debug("syncing file manifest",
		"height", height,
		"commitment", hex.EncodeToString(blob.Commitment),
		"file_id", manifestId,
		"file_name", fileManifest.FileName,
		"file_size", fileManifest.FileSize,
		"mime_type", fileManifest.MimeType,
		"num_chunks", len(fileManifest.Chunks),
	)

	// verify the file hash
	fileData := []byte{}

	// all chunks for this file manifest must be seen
	for _, chunk := range fileManifest.Chunks {
		chunkData, exists, err := tx.GetChunk(state.HashKey(chunk.Id.Commitment))
		if err != nil {
			return fmt.Errorf("error getting chunk data: %v", err)
		}
		if !exists {
			chunkId := types.BlobIdentifierFromProto(chunk.Id)
			slog.Warn(
				"file manifest will not be inlcuded due to missing chunk",
				"file_id", manifestId,
				"chunk_id", chunkId,
				"commitment", hex.EncodeToString(chunk.Id.Commitment),
			)
			return nil
		}

		fileData = append(fileData, chunkData...)
	}

	// Verify file length is correct in manifest
	if uint64(len(fileData)) != fileManifest.FileSize {
		slog.Warn("file manifest will not be inlcuded due to file size mismatch",
			"file_id", manifestId,
			"expected_size", fileManifest.FileSize,
			"actual_size", len(fileData),
		)
		return nil
	}

	// verify the file hash
	fileHash := crypto.HashBytes(fileData)
	if !fileHash.EqualBytes(fileManifest.FileHash) {
		slog.Warn("file manifest will not be inlcuded due to hash mismatch",
			"file_id", manifestId,
			"expected_hash", hex.EncodeToString(fileManifest.FileHash),
			"actual_hash", hex.EncodeToString(fileHash[:]),
		)
		return nil
	}

	return tx.PutFileManifest(manifestId, fileManifest)
}

func (bc *BlobcastChain) SyncDirectoryManifest(
	ctx context.Context,
	tx *state.ChainStateTransaction,
	height uint64,
	blob *blob.Blob,
	directoryManifest *pbStorageV1.DirectoryManifest,
) error {
	manifestId := &types.BlobIdentifier{
		Height:     height,
		Commitment: blob.Commitment,
	}

	slog.Debug("syncing directory manifest",
		"height", height,
		"name", directoryManifest.DirectoryName,
		"commitment", hex.EncodeToString(blob.Commitment),
		"dir_id", manifestId,
		"num_files", len(directoryManifest.Files),
		"dir_hash", hex.EncodeToString(directoryManifest.DirectoryHash),
	)

	// Create merkle root of all file hashes
	// For merkle roots to match, directory files must be sorted by relative path
	merkleLeaves := make([][]byte, 0, len(directoryManifest.Files))

	// all files for this directory manifest must be seen
	for _, file := range directoryManifest.Files {
		fileId := types.BlobIdentifierFromProto(file.Id)
		fileManifest, found, err := tx.GetFileManifest(fileId)
		if err != nil {
			return fmt.Errorf("error getting file manifest: %v", err)
		}
		if !found {
			slog.Warn("directory manifest will not be included due to missing file manifest",
				"dir_id", manifestId,
				"file_id", fileId,
				"commitment", hex.EncodeToString(file.Id.Commitment),
			)
			return nil
		}

		fileHash := crypto.HashBytes([]byte(file.RelativePath), fileManifest.FileHash)
		merkleLeaves = append(merkleLeaves, fileHash.Bytes())
	}

	dirMerkleRoot := merkle.CalculateMerkleRoot(merkleLeaves)

	if !dirMerkleRoot.EqualBytes(directoryManifest.DirectoryHash) {
		slog.Warn("directory manifest will not be included due to hash mismatch",
			"dir_id", manifestId,
			"expected_hash", hex.EncodeToString(directoryManifest.DirectoryHash),
			"actual_hash", hex.EncodeToString(dirMerkleRoot[:]),
		)
		return nil
	}

	return tx.PutDirectoryManifest(manifestId, directoryManifest)
}

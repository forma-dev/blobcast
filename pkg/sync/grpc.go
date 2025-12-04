package sync

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"sync"

	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
	pbSyncapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/syncapis/v1"
)

type processedBlockData struct {
	height        uint64
	block         *types.Block
	chunks        map[state.HashKey][]byte
	chunkHashes   map[state.HashKey]crypto.Hash
	fileManifests map[*types.BlobIdentifier]*pbStorageV1.FileManifest
	dirManifests  map[*types.BlobIdentifier]*pbStorageV1.DirectoryManifest
	mmrSnapshot   []byte
	blockHash     crypto.Hash
}

type blockResult struct {
	data *processedBlockData
	err  error
}

func (bc *BlobcastChain) SyncChainFromGrpc(
	ctx context.Context,
	rollupClient pbRollupapisV1.RollupServiceClient,
	syncClient pbSyncapisV1.SyncServiceClient,
) error {
	nextHeight := uint64(0)

	finalizedHeight, err := bc.chainState.FinalizedHeight()
	if err != nil {
		return fmt.Errorf("error getting finalized height: %v", err)
	}

	if finalizedHeight > 0 {
		nextHeight = finalizedHeight + 1
	}

	chainInfo, err := rollupClient.GetChainInfo(ctx, &pbRollupapisV1.GetChainInfoRequest{})
	if err != nil {
		return fmt.Errorf("error getting chain info: %v", err)
	}

	if nextHeight >= chainInfo.FinalizedHeight {
		slog.Info("chain is up to date", "height", nextHeight)
		return nil
	}

	slog.Info("starting historical sync from remote node", "height", nextHeight, "latest_height", chainInfo.FinalizedHeight)

	return bc.syncWithStream(ctx, syncClient, nextHeight, chainInfo.FinalizedHeight)
}

func (bc *BlobcastChain) syncWithStream(ctx context.Context, syncClient pbSyncapisV1.SyncServiceClient, startHeight, endHeight uint64) error {
	const batchSize = 1000
	maxWorkers := runtime.NumCPU() * 2

	stream, err := syncClient.StreamBlocks(ctx, &pbSyncapisV1.StreamBlocksRequest{
		StartHeight: startHeight,
		EndHeight:   endHeight,
	})
	if err != nil {
		return fmt.Errorf("error streaming blocks: %v", err)
	}

	blockJobs := make(chan *pbSyncapisV1.StreamBlocksResponse, maxWorkers)
	results := make(chan blockResult, maxWorkers)

	// Start worker pool
	var wg sync.WaitGroup
	for i := range maxWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			bc.blockWorker(ctx, blockJobs, results, workerID)
		}(i)
	}

	// Start stream receiver
	streamDone := make(chan error, 1)
	go func() {
		defer close(blockJobs)

		for {
			blockResp, err := stream.Recv()
			if err == io.EOF {
				streamDone <- nil
				return
			}
			if err != nil {
				streamDone <- fmt.Errorf("error receiving block: %v", err)
				return
			}

			select {
			case blockJobs <- blockResp:
			case <-ctx.Done():
				streamDone <- ctx.Err()
				return
			}
		}
	}()

	// Collect results and batch commit
	commitDone := make(chan error, 1)
	go func() {
		commitDone <- bc.batchCommitter(results, batchSize)
	}()

	// Wait for stream to finish
	if err := <-streamDone; err != nil {
		return err
	}

	// Wait for workers
	wg.Wait()
	close(results)

	// Wait for all processing to finish
	return <-commitDone
}

func (bc *BlobcastChain) blockWorker(
	ctx context.Context,
	blockJobs <-chan *pbSyncapisV1.StreamBlocksResponse,
	results chan<- blockResult,
	workerID int,
) {
	for blockResp := range blockJobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		height := blockResp.Block.Header.Height

		data, err := bc.processStreamedBlock(blockResp)
		if err != nil {
			results <- blockResult{err: fmt.Errorf("worker %d failed processing block %d: %v", workerID, height, err)}
			continue
		}

		results <- blockResult{data: data}
	}
}

func (bc *BlobcastChain) processStreamedBlock(blockResp *pbSyncapisV1.StreamBlocksResponse) (*processedBlockData, error) {
	block := types.BlockFromProto(blockResp.Block)

	if len(blockResp.Chunks) != len(block.Body.Chunks) {
		return nil, fmt.Errorf("number of chunks in block does not match number of chunks in streamed response")
	}
	if len(blockResp.Files) != len(block.Body.Files) {
		return nil, fmt.Errorf("number of files in block does not match number of files in streamed response")
	}
	if len(blockResp.Dirs) != len(block.Body.Dirs) {
		return nil, fmt.Errorf("number of dirs in block does not match number of dirs in streamed response")
	}

	data := &processedBlockData{
		height:        block.Height(),
		block:         block,
		chunks:        make(map[state.HashKey][]byte),
		chunkHashes:   make(map[state.HashKey]crypto.Hash),
		fileManifests: make(map[*types.BlobIdentifier]*pbStorageV1.FileManifest),
		dirManifests:  make(map[*types.BlobIdentifier]*pbStorageV1.DirectoryManifest),
		mmrSnapshot:   blockResp.MmrSnapshot,
		blockHash:     crypto.Hash(blockResp.Block.Hash),
	}

	for i, chunk := range blockResp.Chunks {
		chunkId := block.Body.Chunks[i]
		key := state.HashKey(chunkId.Commitment)

		data.chunks[key] = chunk.ChunkData
		data.chunkHashes[key] = crypto.Hash(chunk.ChunkHash)

		slog.Debug("processed chunk",
			"height", block.Height(),
			"commitment", hex.EncodeToString(chunkId.Commitment[:]),
			"size", len(chunk.ChunkData),
		)
	}

	for i, fileManifest := range blockResp.Files {
		fileId := block.Body.Files[i]
		data.fileManifests[fileId] = fileManifest

		slog.Debug("processed file manifest",
			"height", block.Height(),
			"commitment", hex.EncodeToString(fileId.Commitment[:]),
			"file_id", fileId.ID(),
			"file_name", fileManifest.FileName,
		)
	}

	for i, directoryManifest := range blockResp.Dirs {
		dirId := block.Body.Dirs[i]
		data.dirManifests[dirId] = directoryManifest

		slog.Debug("processed directory manifest",
			"height", block.Height(),
			"commitment", hex.EncodeToString(dirId.Commitment[:]),
			"dir_id", dirId.ID(),
			"name", directoryManifest.DirectoryName,
		)
	}

	slog.Debug("processed block",
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

	return data, nil
}

func (bc *BlobcastChain) batchCommitter(results <-chan blockResult, batchSize int) error {
	processedBlocks := make(map[uint64]*processedBlockData)
	var heights []uint64

	for result := range results {
		if result.err != nil {
			return result.err
		}

		data := result.data
		processedBlocks[data.height] = data
		heights = append(heights, data.height)

		if len(processedBlocks) >= batchSize {
			if err := bc.commitBatch(processedBlocks, heights); err != nil {
				return err
			}

			processedBlocks = make(map[uint64]*processedBlockData)
			heights = nil
		}
	}

	if len(processedBlocks) > 0 {
		if err := bc.commitBatch(processedBlocks, heights); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BlobcastChain) commitBatch(processedBlocks map[uint64]*processedBlockData, heights []uint64) error {
	slices.Sort(heights)

	tx, err := bc.chainState.BeginTransaction()
	if err != nil {
		return fmt.Errorf("error beginning batch transaction: %v", err)
	}
	defer tx.Abort()

	slog.Info("committing batch", "blocks", len(heights), "range", fmt.Sprintf("%d-%d", heights[0], heights[len(heights)-1]))

	for _, height := range heights {
		data := processedBlocks[height]

		for key, chunkData := range data.chunks {
			hash := data.chunkHashes[key]
			if err := tx.PutChunk(key, chunkData, hash); err != nil {
				return fmt.Errorf("error storing chunk in batch: %v", err)
			}
		}

		for fileId, fileManifest := range data.fileManifests {
			if err := tx.PutFileManifest(fileId, fileManifest); err != nil {
				return fmt.Errorf("error storing file manifest in batch: %v", err)
			}
		}

		for dirId, dirManifest := range data.dirManifests {
			if err := tx.PutDirectoryManifest(dirId, dirManifest); err != nil {
				return fmt.Errorf("error storing directory manifest in batch: %v", err)
			}
		}

		data.block.Header.UpdateHashCache(data.blockHash)
		if err := tx.PutBlock(data.height, data.block); err != nil {
			return fmt.Errorf("error storing block in batch: %v", err)
		}

		if len(data.mmrSnapshot) > 0 && data.height > 0 {
			if err := tx.PutStateMMRRaw(data.height, data.mmrSnapshot); err != nil {
				return fmt.Errorf("error storing MMR in batch: %v", err)
			}
		}

		slog.Debug("saved block to disk", "height", data.height, "hash", data.block.Hash())
	}

	maxHeight := heights[len(heights)-1]
	if err := tx.SetFinalizedHeight(maxHeight); err != nil {
		return fmt.Errorf("error setting finalized height in batch: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing batch: %v", err)
	}

	slog.Info("committed batch successfully", "blocks", len(heights), "finalized_height", maxHeight)

	// Publish headers for all blocks in the batch to subscribers
	for _, height := range heights {
		data := processedBlocks[height]
		bc.publishNewHeader(data.block.Header)
	}

	return nil
}

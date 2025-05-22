package sync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/fileutil"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbPrimitivesV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/primitives/v1"
	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

// GetChunkData downloads a single chunk of data from Celestia
func GetChunkData(ctx context.Context, celestiaDA celestia.BlobStore, chunkRef *pbStorageV1.ChunkReference) ([]byte, error) {
	slog.Debug("Getting chunk data",
		"block_height", chunkRef.Id.Height,
		"namespace", hex.EncodeToString(chunkRef.Id.Commitment),
		"commitment_hash", hex.EncodeToString(chunkRef.ChunkHash),
		"chunk_hash", hex.EncodeToString(chunkRef.ChunkHash),
	)

	chunkState, err := state.GetChunkState()
	if err != nil {
		return nil, fmt.Errorf("error getting chunk state db: %v", err)
	}

	cachedData, exists, err := chunkState.Get(state.ChunkKey(chunkRef.Id.Commitment))
	if err != nil {
		return nil, fmt.Errorf("error getting chunk data from state: %v", err)
	}

	if exists {
		return cachedData, nil
	}

	existsInCelestia, err := celestiaDA.Has(ctx, chunkRef.Id.Height, chunkRef.Id.Commitment)
	if err != nil {
		return nil, fmt.Errorf("error checking if chunk exists in Celestia: %v", err)
	}

	if !existsInCelestia {
		return nil, fmt.Errorf("chunk does not exist in Celestia")
	}

	chunkData, err := celestiaDA.Read(ctx, chunkRef.Id.Height, chunkRef.Id.Commitment)
	if err != nil {
		return nil, fmt.Errorf("error reading chunk from Celestia: %v", err)
	}

	slog.Debug("Verifying chunk hash")
	chunkHash := sha256.Sum256(chunkData)
	if !bytes.Equal(chunkHash[:], chunkRef.ChunkHash) {
		return nil, fmt.Errorf("chunk hash verification failed")
	}

	chunkState.Set(state.ChunkKey(chunkRef.Id.Commitment), chunkData)

	return chunkData, nil
}

// Download a file using its manifest
func GetFileData(
	ctx context.Context,
	celestiaDA celestia.BlobStore,
	id *types.BlobIdentifier,
	maxConcurrency int,
) ([]byte, error) {
	fileManifest, err := GetFileManifest(ctx, celestiaDA, id)
	if err != nil {
		return nil, fmt.Errorf("error getting file manifest: %v", err)
	}

	slog.Info("Getting file data",
		"file_name", fileManifest.FileName,
		"file_size", fileManifest.FileSize,
		"num_chunks", len(fileManifest.Chunks),
		"blobcast_url", id.URL(),
	)

	// If maxConcurrency is 0 or negative, use a reasonable default
	if maxConcurrency <= 0 {
		maxConcurrency = DefaultMaxConcurrency
	}

	// Download all chunks concurrently and reconstruct the file
	chunks := make([][]byte, len(fileManifest.Chunks))
	errChan := make(chan error, len(fileManifest.Chunks))

	// Create a semaphore channel to limit concurrency
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	// mutex for slice updates
	var mu sync.Mutex

	for i, chunk := range fileManifest.Chunks {
		wg.Add(1)

		go func(index int, chunkRef *pbStorageV1.ChunkReference) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			slog.Debug("Getting chunk data", "chunk_index", index+1, "num_chunks", len(fileManifest.Chunks))

			data, err := GetChunkData(ctx, celestiaDA, chunkRef)
			if err != nil {
				errChan <- fmt.Errorf("error downloading chunk %d: %v", index+1, err)
				return
			}

			slog.Debug("Chunk data retrieved", "chunk_index", index+1, "num_chunks", len(fileManifest.Chunks), "chunk_size", len(data))

			mu.Lock()
			chunks[index] = data
			mu.Unlock()
		}(i, chunk)
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	// Combine chunks in the correct order
	fileData := make([]byte, 0, fileManifest.FileSize)
	for _, chunkData := range chunks {
		fileData = append(fileData, chunkData...)
	}

	// Verify file length is correct in manifest
	if uint64(len(fileData)) != fileManifest.FileSize {
		return nil, fmt.Errorf("file length verification failed")
	}

	// Verify the file hash
	fileHash := sha256.Sum256(fileData)
	slog.Debug("Verifying file hash", "expected_file_hash", hex.EncodeToString(fileManifest.FileHash), "computed_file_hash", hex.EncodeToString(fileHash[:]))

	if !bytes.Equal(fileHash[:], fileManifest.FileHash) {
		return nil, fmt.Errorf("file hash verification failed")
	}

	slog.Info("Successfully obtained file data",
		"file_name", fileManifest.FileName,
		"file_size", fileManifest.FileSize,
		"num_chunks", len(fileManifest.Chunks),
		"blobcast_url", id.URL(),
	)

	return fileData, nil
}

func PutFileData(
	ctx context.Context,
	storageClient pbStorageapisV1.StorageServiceClient,
	da celestia.BlobStore,
	fileName string,
	fileData []byte,
	maxBlobSize int,
) (*types.BlobIdentifier, *pbStorageV1.FileManifest, error) {
	fileHash := sha256.Sum256(fileData)

	// get upload state
	uploadState, err := state.GetUploadState()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting upload state: %v", err)
	}

	// get file state
	fileState, err := uploadState.GetUploadRecord(fileHash)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting file state: %v", err)
	}

	// file is already completed
	if fileState.Completed {
		manifestIdentifier, err := types.BlobIdentifierFromString(fileState.ManifestID)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing manifest identifier: %v", err)
		}

		fileManifestResponse, err := storageClient.GetFileManifest(ctx, &pbStorageapisV1.GetFileManifestRequest{
			Id: manifestIdentifier.Proto(),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("error getting file manifest: %v", err)
		}

		slog.Info("File is already completed",
			"file_name", fileName,
			"file_hash", hex.EncodeToString(fileHash[:]),
			"blobcast_url", manifestIdentifier.URL(),
		)

		return manifestIdentifier, fileManifestResponse.Manifest, nil
	}

	// Get mime type of the original file data
	mimeType, err := fileutil.DetectMimeType(fileData) // Mime type is based on original content
	if err != nil {
		mimeType = "application/octet-stream"
	}

	slog.Debug(
		"Creating file manifest",
		"file_name", fileName,
		"file_size", len(fileData),
		"file_hash", hex.EncodeToString(fileHash[:]),
		"mime_type", mimeType,
	)

	// Create file manifest
	manifest := &pbStorageV1.FileManifest{
		ManifestVersion:      "1.0",
		FileName:             fileName,
		MimeType:             mimeType,
		FileSize:             uint64(len(fileData)), // Size of the (potentially encrypted) data
		FileHash:             fileHash[:],           // Hash of the (potentially encrypted) data
		CompressionAlgorithm: pbStorageV1.CompressionAlgorithm_COMPRESSION_ALGORITHM_NONE,
		Chunks:               make([]*pbStorageV1.ChunkReference, 0),
	}

	// Split the file into chunks (based on actualDataToProcess)
	chunks := fileutil.Split(fileData, maxBlobSize)
	slog.Debug("Split file data into chunks", "num_chunks", len(chunks))

	// Process chunks in batches up to maxTxSize
	batches := make([][]*pbStorageV1.ChunkData, 1)
	currentBatchSize := 0
	currentBatchIdx := 0
	chunkHashes := make([][32]byte, len(chunks))
	for i, chunk := range chunks {
		chunkHashes[i] = sha256.Sum256(chunk)
		chunkState, err := uploadState.GetUploadRecord(chunkHashes[i])
		if err != nil {
			return nil, nil, fmt.Errorf("error getting chunk state: %v", err)
		}

		if chunkState.Completed {
			slog.Debug("Chunk is already completed", "chunk_hash", hex.EncodeToString(chunkHashes[i][:]))
			continue
		}

		estimatedSize := len(chunk) + 100
		if currentBatchSize+estimatedSize > da.Cfg().MaxTxSize {
			currentBatchIdx++
			batches = append(batches, make([]*pbStorageV1.ChunkData, 0))
			currentBatchSize = 0
		}

		chunkData := &pbStorageV1.ChunkData{
			Version:   "1.0",
			ChunkData: chunk,
		}

		batches[currentBatchIdx] = append(batches[currentBatchIdx], chunkData)
		currentBatchSize += estimatedSize
	}

	slog.Debug("Partitioned chunks into batches", "num_batches", len(batches))

	// Upload batches
	chunksSubmitted := 0
	for batchIdx, batch := range batches {
		slog.Debug("Submitting batch of chunks", "batch", batchIdx+1, "num_chunks", len(batch), "num_batches", len(batches))

		blobcastEnvelopes := make([][]byte, len(batch))
		for i, chunk := range batch {
			blobcastEnvelope := &pbRollupV1.BlobcastEnvelope{
				Payload: &pbRollupV1.BlobcastEnvelope_ChunkData{
					ChunkData: chunk,
				},
			}
			blobcastEnvelopeData, err := proto.Marshal(blobcastEnvelope)
			if err != nil {
				return nil, manifest, fmt.Errorf("error marshalling chunk data into blobcast envelope: %v", err)
			}
			blobcastEnvelopes[i] = blobcastEnvelopeData
		}

		commitments, height, err := da.StoreBatch(ctx, blobcastEnvelopes)
		if err != nil {
			return nil, manifest, fmt.Errorf("error submitting batch %d of %d chunks to Celestia: %v", batchIdx+1, len(batch), err)
		}

		// Add all chunks to the manifest
		for i, chunk := range batch {
			chunkHash := sha256.Sum256(chunk.ChunkData)
			commitment := commitments[i]

			slog.Debug(
				"Successfully submitted chunk to Celestia",
				"batch", batchIdx+1,
				"chunk", chunksSubmitted+i+1,
				"chunk_size", len(chunk.ChunkData),
				"chunk_hash", hex.EncodeToString(chunkHash[:]),
				"height", height,
				"commitment", hex.EncodeToString(commitment),
			)

			chunkState, err := uploadState.GetUploadRecord(chunkHash)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting chunk state: %v", err)
			}

			chunkId := &types.BlobIdentifier{
				Height:     height,
				Commitment: commitment,
			}

			chunkState.Completed = true
			chunkState.ManifestID = chunkId.ID()
			err = uploadState.SaveUploadRecord(chunkState)
			if err != nil {
				return nil, nil, fmt.Errorf("error saving chunk state: %v", err)
			}
		}
		chunksSubmitted += len(batch)
	}

	// add chunks to file manifest
	for i := range chunkHashes {
		chunkState, err := uploadState.GetUploadRecord(chunkHashes[i])
		if err != nil {
			return nil, nil, fmt.Errorf("error getting chunk state: %v", err)
		}

		chunkId, err := types.BlobIdentifierFromString(chunkState.ManifestID)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing manifest identifier: %v", err)
		}

		manifest.Chunks = append(manifest.Chunks, &pbStorageV1.ChunkReference{
			Id: &pbPrimitivesV1.BlobIdentifier{
				Height:     chunkId.Height,
				Commitment: chunkId.Commitment,
			},
			ChunkHash: chunkHashes[i][:],
		})
	}

	// submit file manifest
	manifestIdentifier, err := PutFileManifest(ctx, da, manifest)
	if err != nil {
		return nil, manifest, fmt.Errorf("error submitting file manifest to Celestia: %v", err)
	}

	slog.Info(
		"Successfully submitted file manifest to Celestia",
		"file_name", fileName,
		"file_hash", hex.EncodeToString(fileHash[:]),
		"blobcast_url", manifestIdentifier.URL(),
	)

	// save file state
	fileState.Completed = true
	fileState.ManifestID = manifestIdentifier.ID()
	err = uploadState.SaveUploadRecord(fileState)
	if err != nil {
		return nil, manifest, fmt.Errorf("error saving file state: %v", err)
	}

	return manifestIdentifier, manifest, nil
}

package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/fileutil"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

func PutFileData(
	ctx context.Context,
	da celestia.BlobStore,
	fileName string,
	fileData []byte,
	maxBlobSize int,
) (*types.BlobIdentifier, crypto.Hash, error) {
	fileHash := crypto.HashBytes(fileData)

	// get upload state
	uploadState, err := state.GetUploadState()
	if err != nil {
		return nil, fileHash, fmt.Errorf("error getting upload state: %v", err)
	}

	// get file state
	fileState, err := uploadState.GetUploadRecord(state.UploadRecordKey(fileHash))
	if err != nil {
		return nil, fileHash, fmt.Errorf("error getting file state: %v", err)
	}

	// file is already completed
	if fileState.Completed {
		manifestIdentifier, err := types.BlobIdentifierFromString(fileState.ManifestID)
		if err != nil {
			return nil, fileHash, fmt.Errorf("error parsing manifest identifier: %v", err)
		}

		slog.Info("File is already completed",
			"file_name", fileName,
			"file_hash", hex.EncodeToString(fileHash[:]),
			"blobcast_url", manifestIdentifier.URL(),
		)

		return manifestIdentifier, fileHash, nil
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
			return nil, fileHash, fmt.Errorf("error getting chunk state: %v", err)
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
				return nil, fileHash, fmt.Errorf("error marshalling chunk data into blobcast envelope: %v", err)
			}
			blobcastEnvelopes[i] = blobcastEnvelopeData
		}

		commitments, height, err := da.StoreBatch(ctx, blobcastEnvelopes)
		if err != nil {
			return nil, fileHash, fmt.Errorf("error submitting batch %d of %d chunks to Celestia: %v", batchIdx+1, len(batch), err)
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
				return nil, fileHash, fmt.Errorf("error getting chunk state: %v", err)
			}

			chunkId := &types.BlobIdentifier{
				Height:     height,
				Commitment: crypto.Hash(commitment),
			}

			chunkState.Completed = true
			chunkState.ManifestID = chunkId.ID()
			err = uploadState.SaveUploadRecord(chunkState)
			if err != nil {
				return nil, fileHash, fmt.Errorf("error saving chunk state: %v", err)
			}
		}
		chunksSubmitted += len(batch)
	}

	// add chunks to file manifest
	for i := range chunkHashes {
		chunkState, err := uploadState.GetUploadRecord(chunkHashes[i])
		if err != nil {
			return nil, fileHash, fmt.Errorf("error getting chunk state: %v", err)
		}

		chunkId, err := types.BlobIdentifierFromString(chunkState.ManifestID)
		if err != nil {
			return nil, fileHash, fmt.Errorf("error parsing manifest identifier: %v", err)
		}

		manifest.Chunks = append(manifest.Chunks, &pbStorageV1.ChunkReference{
			Id:        chunkId.Proto(),
			ChunkHash: chunkHashes[i][:],
		})
	}

	// submit file manifest
	manifestIdentifier, err := PutFileManifest(ctx, da, manifest)
	if err != nil {
		return nil, fileHash, fmt.Errorf("error submitting file manifest to Celestia: %v", err)
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
		return nil, fileHash, fmt.Errorf("error saving file state: %v", err)
	}

	return manifestIdentifier, fileHash, nil
}

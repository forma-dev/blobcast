package sync

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"

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
	submitter *Submitter,
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

	// Split the file into chunks
	chunks := fileutil.Split(fileData, maxBlobSize)
	slog.Debug("Split file data into chunks", "num_chunks", len(chunks))

	// Prepare chunks for submission
	chunkResults := make([]chan SubmissionResult, len(chunks))
	chunkHashes := make([][32]byte, len(chunks))

	for i, chunk := range chunks {
		chunkHashes[i] = crypto.HashBytes(chunk)

		chunkState, err := uploadState.GetUploadRecord(chunkHashes[i])
		if err != nil {
			return nil, fileHash, fmt.Errorf("error getting chunk state: %v", err)
		}

		if chunkState.Completed {
			slog.Debug("Chunk is already completed", "chunk_hash", hex.EncodeToString(chunkHashes[i][:]))
			continue
		}

		blobcastEnvelope := &pbRollupV1.BlobcastEnvelope{
			Payload: &pbRollupV1.BlobcastEnvelope_ChunkData{
				ChunkData: &pbStorageV1.ChunkData{
					Version:   "1.0",
					ChunkData: chunk,
				},
			},
		}

		blobcastEnvelopeData, err := proto.Marshal(blobcastEnvelope)
		if err != nil {
			return nil, fileHash, fmt.Errorf("error marshalling chunk data into blobcast envelope: %v", err)
		}

		chunkResults[i] = submitter.Submit(blobcastEnvelopeData)
	}

	// Wait for all chunks to be submitted
	for i, chunkResult := range chunkResults {
		// chunk was previously submitted
		if chunkResult == nil {
			continue
		}

		select {
		case result := <-chunkResult:
			if result.Error != nil {
				return nil, fileHash, fmt.Errorf("error submitting chunk %d: %v", i+1, result.Error)
			}

			chunkId := result.BlobID

			slog.Debug(
				"Successfully submitted chunk to Celestia",
				"file_hash", hex.EncodeToString(fileHash[:]),
				"chunk", i+1,
				"chunk_size", len(chunks[i]),
				"chunk_hash", hex.EncodeToString(chunkHashes[i][:]),
				"height", chunkId.Height,
				"commitment", hex.EncodeToString(chunkId.Commitment[:]),
			)

			// save upload record
			chunkState, err := uploadState.GetUploadRecord(state.UploadRecordKey(chunkHashes[i]))
			if err != nil {
				return nil, fileHash, fmt.Errorf("error getting chunk state: %v", err)
			}

			chunkState.Completed = true
			chunkState.ManifestID = chunkId.ID()
			err = uploadState.SaveUploadRecord(chunkState)
			if err != nil {
				return nil, fileHash, fmt.Errorf("error saving chunk state: %v", err)
			}
		case <-ctx.Done():
			return nil, fileHash, fmt.Errorf("context cancelled while waiting for chunk submission")
		}
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
	manifestIdentifier, err := PutFileManifest(ctx, submitter, manifest)
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

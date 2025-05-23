package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/forma-dev/blobcast/pkg/types"
)

func GetFileData(ctx context.Context, id *types.BlobIdentifier) ([]byte, error) {
	fileManifest, err := GetFileManifest(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error getting file manifest: %v", err)
	}

	slog.Info("Getting file data",
		"file_name", fileManifest.FileName,
		"file_size", fileManifest.FileSize,
		"num_chunks", len(fileManifest.Chunks),
		"blobcast_url", id.URL(),
	)

	fileData := make([]byte, 0, fileManifest.FileSize)

	for i, chunk := range fileManifest.Chunks {
		slog.Debug("Getting chunk data", "chunk_index", i+1, "num_chunks", len(fileManifest.Chunks))

		data, err := GetChunkData(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("error downloading chunk %d: %v", i+1, err)
		}

		slog.Debug("Chunk data retrieved", "chunk_index", i+1, "num_chunks", len(fileManifest.Chunks), "chunk_size", len(data))

		fileData = append(fileData, data...)
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

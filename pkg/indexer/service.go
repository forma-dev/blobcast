package indexer

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbPrimitivesV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/primitives/v1"
	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

type IndexerService struct {
	storageClient pbStorageapisV1.StorageServiceClient
	rollupClient  pbRollupapisV1.RollupServiceClient
	indexDB       *state.IndexerDatabase
	lastIndexed   uint64
}

func NewIndexerService(
	storageClient pbStorageapisV1.StorageServiceClient,
	rollupClient pbRollupapisV1.RollupServiceClient,
	startHeight uint64,
	dbConnString string,
) (*IndexerService, error) {
	indexDB, err := state.NewIndexerDatabase(dbConnString)
	if err != nil {
		return nil, fmt.Errorf("error creating index database: %v", err)
	}

	lastIndexed := startHeight
	if startHeight == 0 {
		if dbHeight, err := indexDB.GetLastIndexedHeight(); err == nil {
			lastIndexed = dbHeight
		}
	}

	return &IndexerService{
		storageClient: storageClient,
		rollupClient:  rollupClient,
		indexDB:       indexDB,
		lastIndexed:   lastIndexed,
	}, nil
}

func (is *IndexerService) Start(ctx context.Context) error {
	// Use a labeled loop to restart subscription without recursion
subscriptionLoop:
	for {
		// Subscribe to block headers starting from the next block we need to index
		stream, err := is.rollupClient.SubscribeToHeaders(ctx, &pbRollupapisV1.SubscribeToHeadersRequest{
			StartHeight: is.lastIndexed + 1,
		})
		if err != nil {
			slog.Error("Error subscribing to headers", "error", err)
			// Reconnect with exponential backoff
			if reconnectErr := is.reconnectWithBackoff(ctx); reconnectErr != nil {
				return reconnectErr
			}
			// Retry subscription from the top of the loop
			continue subscriptionLoop
		}

		slog.Info("Subscribed to block headers", "start_height", is.lastIndexed+1)

		// Receive and process headers as they arrive
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Receive next header from stream
				headerResp, err := stream.Recv()
				if err != nil {
					slog.Error("Error receiving header from stream", "error", err)
					// Reconnect with exponential backoff
					if reconnectErr := is.reconnectWithBackoff(ctx); reconnectErr != nil {
						return reconnectErr
					}
					// Restart the subscription from the outer loop
					continue subscriptionLoop
				}

				// Fetch and index the full block
				height := headerResp.Header.Height
				slog.Debug("Received header notification", "height", height)

				if err := is.indexBlock(ctx, height); err != nil {
					slog.Error("Error indexing block, reconnecting to retry", "height", height, "error", err)
					// Reconnect with exponential backoff
					if reconnectErr := is.reconnectWithBackoff(ctx); reconnectErr != nil {
						return reconnectErr
					}
					// Restart subscription from lastIndexed + 1 to retry the failed block
					continue subscriptionLoop
				}

				is.lastIndexed = height
				slog.Info("Indexed new block", "height", height)
			}
		}
	}
}

func (is *IndexerService) reconnectWithBackoff(ctx context.Context) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			slog.Info("Attempting to reconnect to header stream", "backoff", backoff)

			// Try to get chain info to test connection
			_, err := is.rollupClient.GetChainInfo(ctx, &pbRollupapisV1.GetChainInfoRequest{})
			if err == nil {
				slog.Info("Reconnection successful")
				return nil
			}

			slog.Warn("Reconnection failed, retrying", "error", err, "next_backoff", backoff*2)

			// Exponential backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (is *IndexerService) indexBlock(ctx context.Context, height uint64) error {
	blockResp, err := is.rollupClient.GetBlockByHeight(ctx, &pbRollupapisV1.GetBlockByHeightRequest{
		Height: height,
	})
	if err != nil {
		return fmt.Errorf("error fetching block %d: %v", height, err)
	}

	block := blockResp.Block

	storageUsed := uint64(0)

	blockIndex := &state.BlockIndex{
		Height:           block.Header.Height,
		Hash:             hex.EncodeToString(block.Hash),
		Timestamp:        block.Header.Timestamp.AsTime(),
		CelestiaHeight:   block.Header.CelestiaBlockHeight,
		TotalChunks:      len(block.Body.Chunks),
		TotalFiles:       len(block.Body.Files),
		TotalDirectories: len(block.Body.Dirs),
		StorageUsed:      0, // Will be updated after indexing files/dirs
		ParentHash:       hex.EncodeToString(block.Header.ParentHash),
		DirsRoot:         hex.EncodeToString(block.Header.DirsRoot),
		FilesRoot:        hex.EncodeToString(block.Header.FilesRoot),
		ChunksRoot:       hex.EncodeToString(block.Header.ChunksRoot),
		StateRoot:        hex.EncodeToString(block.Header.StateRoot),
	}

	if err := is.indexDB.PutBlockIndex(blockIndex); err != nil {
		return err
	}

	for i, chunkID := range block.Body.Chunks {
		chunkSize, err := is.indexChunk(ctx, chunkID, i, block.Header.Height)
		if err != nil {
			slog.Warn("Error indexing chunk", "chunk_id", chunkID, "error", err)
		} else {
			storageUsed += chunkSize
		}
	}

	for _, fileID := range block.Body.Files {
		fileSize, err := is.indexFile(ctx, fileID, block.Header.Height)
		if err != nil {
			slog.Warn("Error indexing file", "file_id", fileID, "error", err)
		} else {
			storageUsed += fileSize
		}
	}

	for _, dirID := range block.Body.Dirs {
		dirSize, err := is.indexDirectory(ctx, dirID, block.Header.Height)
		if err != nil {
			slog.Warn("Error indexing directory", "dir_id", dirID, "error", err)
		} else {
			storageUsed += dirSize
		}
	}

	blockIndex.StorageUsed = storageUsed
	if err := is.indexDB.PutBlockIndex(blockIndex); err != nil {
		return err
	}

	return nil
}

func (is *IndexerService) indexChunk(ctx context.Context, chunkID *pbPrimitivesV1.BlobIdentifier, index int, blockHeight uint64) (uint64, error) {
	chunkRefResp, err := is.storageClient.GetChunkReference(ctx, &pbStorageapisV1.GetChunkReferenceRequest{
		Id: types.BlobIdentifierFromProto(chunkID).ID(),
	})
	if err != nil {
		return 0, err
	}

	chunkReference := chunkRefResp.Reference

	chunkIndex := &state.ChunkIndex{
		BlobID:      types.BlobIdentifierFromProto(chunkID).String(),
		BlockHeight: blockHeight,
		Index:       index,
		ChunkSize:   chunkReference.ChunkSize,
		ChunkHash:   hex.EncodeToString(chunkReference.ChunkHash),
	}

	if err := is.indexDB.PutChunkIndex(chunkIndex); err != nil {
		return 0, err
	}

	return chunkReference.ChunkSize, nil
}

func (is *IndexerService) indexFile(ctx context.Context, fileID *pbPrimitivesV1.BlobIdentifier, blockHeight uint64) (uint64, error) {
	// Fetch file manifest from node
	fileResp, err := is.storageClient.GetFileManifest(ctx, &pbStorageapisV1.GetFileManifestRequest{
		Id: types.BlobIdentifierFromProto(fileID).ID(),
	})
	if err != nil {
		return 0, err
	}

	manifest := fileResp.Manifest

	// Extract file extension and MIME type for categorization
	fileIndex := &state.FileIndex{
		BlobID:      types.BlobIdentifierFromProto(fileID).String(),
		FileName:    manifest.FileName,
		MimeType:    manifest.MimeType,
		FileSize:    manifest.FileSize,
		FileHash:    hex.EncodeToString(manifest.FileHash),
		BlockHeight: blockHeight,
		ChunkCount:  len(manifest.Chunks),
		Tags:        extractTags(manifest.FileName),
	}

	if err := is.indexDB.PutFileIndex(fileIndex); err != nil {
		return 0, err
	}

	for i, chunk := range manifest.Chunks {
		if err := is.indexDB.PutFileChunk(fileIndex.BlobID, types.BlobIdentifierFromProto(chunk.Id).String(), i); err != nil {
			slog.Error("Error indexing file chunk", "file_id", fileIndex.BlobID, "chunk_id", types.BlobIdentifierFromProto(chunk.Id).String(), "error", err)
			return 0, err
		}
	}

	return uint64(proto.Size(manifest)), nil
}

func (is *IndexerService) indexDirectory(ctx context.Context, dirID *pbPrimitivesV1.BlobIdentifier, blockHeight uint64) (uint64, error) {
	// Fetch directory manifest from node
	dirResp, err := is.storageClient.GetDirectoryManifest(ctx, &pbStorageapisV1.GetDirectoryManifestRequest{
		Id: types.BlobIdentifierFromProto(dirID).ID(),
	})
	if err != nil {
		return 0, err
	}

	manifest := dirResp.Manifest

	// Extract unique file types and calculate total size
	fileTypes := make(map[string]bool)
	totalSize := uint64(0)
	subPaths := make([]string, len(manifest.Files))

	for i, file := range manifest.Files {
		subPaths[i] = file.RelativePath

		// Get file manifest to determine type and size
		fileResp, err := is.storageClient.GetFileManifest(ctx, &pbStorageapisV1.GetFileManifestRequest{
			Id: types.BlobIdentifierFromProto(file.Id).ID(),
		})
		if err == nil {
			fileTypes[fileResp.Manifest.MimeType] = true
			totalSize += fileResp.Manifest.FileSize
		}
	}

	// Convert map to slice
	uniqueFileTypes := make([]string, 0, len(fileTypes))
	for fileType := range fileTypes {
		if fileType != "" {
			uniqueFileTypes = append(uniqueFileTypes, fileType)
		}
	}

	dirIndex := &state.DirectoryIndex{
		BlobID:        types.BlobIdentifierFromProto(dirID).String(),
		DirectoryName: manifest.DirectoryName,
		DirectoryHash: hex.EncodeToString(manifest.DirectoryHash),
		BlockHeight:   blockHeight,
		FileCount:     len(manifest.Files),
		TotalSize:     totalSize,
		FileTypes:     uniqueFileTypes,
		SubPaths:      subPaths,
	}

	if err := is.indexDB.PutDirectoryIndex(dirIndex); err != nil {
		return 0, err
	}

	for _, file := range manifest.Files {
		if err := is.indexDB.PutDirectoryFile(types.BlobIdentifierFromProto(dirID).String(), types.BlobIdentifierFromProto(file.Id).String(), file.RelativePath); err != nil {
			slog.Error(
				"Error indexing directory file",
				"dir_id",
				types.BlobIdentifierFromProto(dirID).String(),
				"file_id",
				types.BlobIdentifierFromProto(file.Id).String(),
				"error",
				err,
			)
			return 0, err
		}
	}

	return uint64(proto.Size(manifest)), nil
}

// extractTags extracts simple tags from filename
func extractTags(filename string) []string {
	tags := []string{}

	// Extract file extension
	if ext := filepath.Ext(filename); ext != "" {
		tags = append(tags, strings.ToLower(ext[1:])) // Remove the dot
	}

	// Extract words from filename (simple tokenization)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	words := strings.FieldsFunc(name, func(c rune) bool {
		return c == '_' || c == '-' || c == ' '
	})

	for _, word := range words {
		if len(word) > 2 { // Only include words longer than 2 characters
			tags = append(tags, strings.ToLower(word))
		}
	}

	return tags
}

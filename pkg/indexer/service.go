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
) (*IndexerService, error) {
	indexDB, err := state.NewIndexerDatabase("")
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

func (is *IndexerService) Start(ctx context.Context, syncInterval time.Duration) error {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	if err := is.catchUp(ctx); err != nil {
		slog.Error("Error during initial catchup", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := is.syncNewBlocks(ctx); err != nil {
				slog.Error("Error syncing new blocks", "error", err)
			}
		}
	}
}

func (is *IndexerService) catchUp(ctx context.Context) error {
	chainInfo, err := is.rollupClient.GetChainInfo(ctx, &pbRollupapisV1.GetChainInfoRequest{})
	if err != nil {
		return fmt.Errorf("error getting chain info: %v", err)
	}

	if is.lastIndexed >= chainInfo.FinalizedHeight {
		slog.Info("Indexer is up to date", "height", is.lastIndexed)
		return nil
	}

	slog.Info("Catching up indexer", "from_height", is.lastIndexed, "to_height", chainInfo.FinalizedHeight)

	for height := is.lastIndexed + 1; height <= chainInfo.FinalizedHeight; height++ {
		if err := is.indexBlock(ctx, height); err != nil {
			return fmt.Errorf("error indexing block %d: %v", height, err)
		}
		is.lastIndexed = height

		if height%100 == 0 {
			slog.Info("Indexing progress", "height", height, "target", chainInfo.FinalizedHeight)
		}
	}

	slog.Info("Indexer catch-up complete", "height", is.lastIndexed)
	return nil
}

func (is *IndexerService) syncNewBlocks(ctx context.Context) error {
	chainInfo, err := is.rollupClient.GetChainInfo(ctx, &pbRollupapisV1.GetChainInfoRequest{})
	if err != nil {
		return err
	}

	for height := is.lastIndexed + 1; height <= chainInfo.FinalizedHeight; height++ {
		if err := is.indexBlock(ctx, height); err != nil {
			return err
		}
		is.lastIndexed = height
		slog.Info("Indexed new block", "height", height)
	}

	return nil
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

package sync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/forma-dev/blobcast/pkg/api/node"
	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/crypto/merkle"
	"github.com/forma-dev/blobcast/pkg/encryption"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/storage"
	"github.com/forma-dev/blobcast/pkg/types"

	pbPrimitivesV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/primitives/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

func ExportFile(ctx context.Context, id *types.BlobIdentifier, target string, encryptionKey []byte) error {
	targetDir := filepath.Dir(target)
	fileName := filepath.Base(target)

	slog.Debug("Exporting file to disk", "target_dir", targetDir, "file_name", fileName, "blobcast_id", id)

	// check if file already exists
	if storage.Exists(target) {
		fileManifest, err := node.GetFileManifest(ctx, id)
		if err != nil {
			return fmt.Errorf("error getting file manifest: %v", err)
		}

		// check if file size matches
		fileData, err := storage.ReadFile(target)
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		// check if file hash matches
		if uint64(len(fileData)) == fileManifest.FileSize {
			fileHash := sha256.Sum256(fileData)
			if bytes.Equal(fileHash[:], fileManifest.FileHash) {
				slog.Info("File already exists", "target_dir", targetDir, "file_name", fileName)
				return nil
			}
		}
	}

	// get file data
	fileData, err := node.GetFileData(ctx, id)
	if err != nil {
		return fmt.Errorf("error getting file data: %v", err)
	}

	// Apply decryption if needed
	dataToWrite := fileData
	if encryptionKey != nil {
		slog.Debug("Decrypting file", "target_dir", targetDir, "file_name", fileName)

		// Read the encrypted file
		encryptedData := fileData // Reuse already read data

		// Decrypt the data
		decryptedData, err := encryption.Decrypt(encryptedData, encryptionKey)
		if err != nil {
			return fmt.Errorf("error decrypting file %s: %v", target, err)
		}

		dataToWrite = decryptedData

		slog.Debug("Decrypted file",
			"target_dir", targetDir,
			"file_name", fileName,
			"encrypted_size", len(encryptedData),
			"original_size", len(dataToWrite),
		)
	}

	// write to disk
	if err := storage.WriteFile(target, dataToWrite); err != nil {
		return fmt.Errorf("error writing file to disk: %v", err)
	}

	slog.Info("Successfully exported file",
		"target_dir", targetDir,
		"file_name", fileName,
		"file_size", len(dataToWrite),
		"blobcast_id", id,
	)

	return nil
}

func ExportDirectory(ctx context.Context, id *types.BlobIdentifier, target string, encryptionKey []byte) error {
	slog.Info("Exporting directory to disk", "target_dir", target, "blobcast_id", id)

	// Create target directory if it doesn't exist
	if err := storage.EnsureDir(target); err != nil {
		return fmt.Errorf("error creating target directory: %v", err)
	}

	directoryManifest, err := node.GetDirectoryManifest(ctx, id)
	if err != nil {
		return fmt.Errorf("error getting directory manifest: %v", err)
	}

	for _, fileRef := range directoryManifest.Files {
		relPath := fileRef.RelativePath
		targetPath := filepath.Join(target, relPath)

		// Get file manifest to determine file size and hash
		fileManifestIdentifier := &types.BlobIdentifier{
			Height:     fileRef.GetId().Height,
			Commitment: fileRef.GetId().Commitment,
		}

		// Download the file
		err = ExportFile(ctx, fileManifestIdentifier, targetPath, encryptionKey)
		if err != nil {
			return fmt.Errorf("error downloading file %s: %v", relPath, err)
		}
	}

	slog.Info("Successfully exported directory", "target_dir", target, "blobcast_id", id)
	return nil
}

func UploadFile(
	ctx context.Context,
	da celestia.BlobStore,
	source string,
	relativePath string,
	maxBlobSize int,
	encryptionKey []byte,
) (*types.BlobIdentifier, crypto.Hash, error) {
	slog.Info("Uploading file", "file_name", source, "relative_path", relativePath, "max_blob_size", maxBlobSize)

	// Read the file data
	data, err := storage.ReadFile(source)
	if err != nil {
		return nil, crypto.Hash{}, fmt.Errorf("error reading file %s: %v", source, err)
	}

	slog.Debug("Successfully read file", "file_name", source, "file_size", len(data))

	// Encrypt the file data if an encryption key is provided
	var dataToProcess []byte = data
	if encryptionKey != nil {
		slog.Debug("Encrypting file", "file_name", relativePath, "file_size", len(data))
		encryptedData, err := encryption.Encrypt(data, encryptionKey)
		if err != nil {
			return nil, crypto.Hash{}, fmt.Errorf("error encrypting file %s: %v", relativePath, err)
		}
		dataToProcess = encryptedData
		slog.Debug("Encrypted file", "file_name", relativePath, "file_size", len(data), "encrypted_size", len(dataToProcess))
	}

	blobIdentifier, fileHash, err := PutFileData(ctx, da, relativePath, dataToProcess, maxBlobSize)
	if err != nil {
		return nil, fileHash, fmt.Errorf("error putting file data: %v", err)
	}

	return blobIdentifier, fileHash, nil
}

func UploadDirectory(
	ctx context.Context,
	da celestia.BlobStore,
	source string,
	maxBlobSize int,
	encryptionKey []byte,
) (*types.BlobIdentifier, crypto.Hash, error) {
	slog.Info("Uploading directory", "directory", source)

	// get upload state
	uploadState, err := state.GetUploadState()
	if err != nil {
		return nil, crypto.Hash{}, fmt.Errorf("error getting upload state: %v", err)
	}

	// get directory state
	dirKey := sha256.Sum256([]byte(source))
	dirState, err := uploadState.GetUploadRecord(dirKey)
	if err != nil {
		return nil, crypto.Hash{}, fmt.Errorf("error getting directory state: %v", err)
	}

	// directory upload already complete
	if dirState.Completed {
		slog.Info("Directory upload already complete", "directory", source)

		manifestIdentifier, err := types.BlobIdentifierFromString(dirState.ManifestID)
		if err != nil {
			return nil, crypto.Hash{}, fmt.Errorf("error parsing manifest identifier: %v", err)
		}

		return manifestIdentifier, crypto.Hash{}, nil
	}

	var filesToUpload []struct {
		path    string
		relPath string
	}

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// Skip common system and temporary files
		filename := filepath.Base(relPath)
		if filename == ".DS_Store" ||
			filename == "Thumbs.db" ||
			filename == "desktop.ini" ||
			strings.HasPrefix(filename, ".git") ||
			strings.HasPrefix(filename, ".idea") ||
			strings.HasPrefix(filename, ".vscode") ||
			strings.HasSuffix(filename, ".tmp") ||
			strings.HasSuffix(filename, ".swp") ||
			strings.HasSuffix(filename, "~") {
			return nil
		}

		filesToUpload = append(filesToUpload, struct {
			path    string
			relPath string
		}{path, relPath})

		return nil
	})

	if err != nil {
		return nil, crypto.Hash{}, fmt.Errorf("error walking directory: %v", err)
	}

	// Create a directory manifest using the new Protocol Buffer format
	dirManifest := &pbStorageV1.DirectoryManifest{
		ManifestVersion: "1.0",
		DirectoryName:   filepath.Base(source),
		Files:           make([]*pbStorageV1.FileReference, 0),
	}

	var fileHashes map[string]crypto.Hash = make(map[string]crypto.Hash)

	// Upload files
	for _, file := range filesToUpload {
		blobIdentifier, fileHash, err := UploadFile(ctx, da, file.path, file.relPath, maxBlobSize, encryptionKey)
		if err != nil {
			return nil, crypto.Hash{}, fmt.Errorf("error uploading file %s: %v", file.path, err)
		}

		// hash with relative path to ensure unique directories with same files have different hashes
		fileHashes[file.relPath] = crypto.HashBytes([]byte(file.relPath), fileHash[:])

		dirManifest.Files = append(dirManifest.Files, &pbStorageV1.FileReference{
			Id: &pbPrimitivesV1.BlobIdentifier{
				Height:     blobIdentifier.Height,
				Commitment: blobIdentifier.Commitment,
			},
			RelativePath: file.relPath,
		})
	}

	// Calculate a hash for the entire directory by combining all file hashes
	// Sort files by path to ensure deterministic hash
	sort.Slice(dirManifest.Files, func(i, j int) bool {
		return dirManifest.Files[i].RelativePath < dirManifest.Files[j].RelativePath
	})

	// Create merkle root of all file hashes
	merkleLeaves := make([][]byte, 0, len(fileHashes))
	for _, fileRef := range dirManifest.Files {
		fileHash, ok := fileHashes[fileRef.RelativePath]
		if !ok {
			return nil, crypto.Hash{}, fmt.Errorf("file hash not found for file %s", fileRef.RelativePath)
		}
		merkleLeaves = append(merkleLeaves, fileHash.Bytes())
	}

	dirMerkleRoot := merkle.CalculateMerkleRoot(merkleLeaves)
	dirManifest.DirectoryHash = dirMerkleRoot.Bytes()

	manifestIdentifier, err := PutDirectoryManifest(ctx, da, dirManifest)
	if err != nil {
		return nil, dirMerkleRoot, fmt.Errorf("error uploading directory manifest: %v", err)
	}

	// save directory state
	dirState.ManifestID = manifestIdentifier.String()
	dirState.Completed = true

	// Save the final state
	if err := uploadState.SaveUploadRecord(dirState); err != nil {
		return nil, dirMerkleRoot, fmt.Errorf("error saving directory state: %v", err)
	}

	return manifestIdentifier, dirMerkleRoot, nil
}

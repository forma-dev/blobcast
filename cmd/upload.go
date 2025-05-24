package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/forma-dev/blobcast/pkg/celestia"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/spf13/cobra"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a directory to Celestia",
	Long: `Upload a directory to Celestia using the blobcast protocol.

All files in the directory will be uploaded as blobs to Celestia, and a manifest will be created.
The manifest URL can be used later to download the directory.`,
	RunE: runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// Add local flags for the upload command
	uploadCmd.Flags().StringVarP(&flagDir, "dir", "d", "", "Directory to upload")
	uploadCmd.Flags().StringVarP(&flagFile, "file", "f", "", "File to upload")
	uploadCmd.Flags().StringVar(&flagMaxBlobSize, "max-blob-size", "384KB", "Max file chunk size (e.g., 1.5MB, 1024KB)")
	uploadCmd.Flags().StringVar(&flagMaxTxSize, "max-tx-size", "2MB", "Max transaction size (e.g., 32MB, 2097152B)")
	uploadCmd.Flags().StringVar(&flagGRPCAddr, "node-grpc", getEnvWithDefault("BLOBCAST_NODE_GRPC", "127.0.0.1:50051"), "gRPC address for a blobcast full node")
}

func runUpload(cmd *cobra.Command, args []string) error {
	if flagDir == "" && flagFile == "" {
		return fmt.Errorf("please supply a directory or file to upload")
	}

	// Validate the directory path
	if flagDir != "" {
		flagDir = filepath.Clean(flagDir)
		fileInfo, err := os.Stat(flagDir)
		if err != nil {
			return fmt.Errorf("error accessing directory: %v", err)
		}
		if !fileInfo.IsDir() {
			return fmt.Errorf("%s is not a directory", flagDir)
		}
	}

	// Validate the file path
	if flagFile != "" {
		flagFile = filepath.Clean(flagFile)
		fileInfo, err := os.Stat(flagFile)
		if err != nil {
			return fmt.Errorf("error accessing file: %v", err)
		}
		if fileInfo.IsDir() {
			return fmt.Errorf("%s is not a file", flagFile)
		}
	}

	// Ensure auth token is provided
	if flagAuth == "" {
		return fmt.Errorf("please supply auth token via --auth flag or BLOBCAST_CELESTIA_NODE_AUTH_TOKEN environment variable")
	}

	// Parse blob and transaction size limits
	maxBlobSize, err := parseSizeString(flagMaxBlobSize)
	if err != nil {
		return fmt.Errorf("error parsing max-blob-size: %v", err)
	}
	if maxBlobSize <= 0 {
		return fmt.Errorf("max-blob-size must be a positive value")
	}

	maxTxSize, err := parseSizeString(flagMaxTxSize)
	if err != nil {
		return fmt.Errorf("error parsing max-tx-size: %v", err)
	}
	if maxTxSize <= 0 {
		return fmt.Errorf("max-tx-size must be a positive value")
	}

	// Process encryption key if provided
	var encryptionKey []byte
	if flagEncryptionKey != "" {
		key, err := hex.DecodeString(flagEncryptionKey)
		if err != nil {
			return fmt.Errorf("error decoding encryption key: %v. Key must be a hex-encoded string", err)
		}
		if len(key) != 32 {
			return fmt.Errorf("encryption key must be 32 bytes (64 hex characters) for AES-256")
		}
		encryptionKey = key
	}

	// initialize storage client w/ nil client (no connection)
	storageClient := pbStorageapisV1.NewStorageServiceClient(nil)

	// Initialize Celestia DA client
	daConfig := celestia.DAConfig{
		Rpc:         flagRPC,
		NamespaceId: flagNamespace,
		AuthToken:   flagAuth,
		MaxTxSize:   maxTxSize,
	}

	celestiaDA, err := celestia.NewCelestiaDA(daConfig)
	if err != nil {
		return fmt.Errorf("error creating Celestia client: %v", err)
	}

	// create filesystem client
	filesystemClient := sync.NewFileSystemClient(storageClient, celestiaDA)

	// do the upload
	switch {
	case flagDir != "":
		manifestIdentifier, _, err := filesystemClient.UploadDirectory(context.Background(), flagDir, maxBlobSize, encryptionKey)
		if err != nil {
			return fmt.Errorf("error uploading directory: %v", err)
		}
		slog.Info("Successfully uploaded directory", "dir", flagDir, "blobcast_url", manifestIdentifier.URL())
	case flagFile != "":
		manifestIdentifier, _, err := filesystemClient.UploadFile(
			context.Background(),
			flagFile,
			filepath.Base(flagFile),
			maxBlobSize,
			encryptionKey,
		)
		if err != nil {
			return fmt.Errorf("error uploading file: %v", err)
		}
		slog.Info("Successfully uploaded file", "file", flagFile, "blobcast_url", manifestIdentifier.URL())
	}

	return nil
}

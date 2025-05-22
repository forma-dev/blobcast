package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
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
	uploadCmd.Flags().StringVar(&flagGRPCAddr, "storage-grpc", getEnvWithDefault("BLOBCAST_STORAGE_GRPC", "127.0.0.1:50051"), "gRPC address for storage service")
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

	if flagGRPCAddr == "" {
		return fmt.Errorf("please supply gRPC address via --storage-grpc flag or BLOBCAST_STORAGE_GRPC environment variable")
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

	// Initialize storage client
	conn, err := grpc.NewClient(
		flagGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("error creating storage client: %v", err)
	}
	defer conn.Close()
	storageClient := pbStorageapisV1.NewStorageServiceClient(conn)

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

	// do the upload
	switch {
	case flagDir != "":
		manifestIdentifier, _, err := sync.UploadDirectory(context.Background(), storageClient, celestiaDA, flagDir, maxBlobSize, encryptionKey)
		if err != nil {
			return fmt.Errorf("error uploading directory: %v", err)
		}
		slog.Info("Successfully uploaded directory", "dir", flagDir, "blobcast_url", manifestIdentifier.URL())
	case flagFile != "":
		manifestIdentifier, _, err := sync.UploadFile(
			context.Background(),
			storageClient,
			celestiaDA,
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

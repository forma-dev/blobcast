package files

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/celestia"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/forma-dev/blobcast/pkg/util"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a file or directory from Blobcast to disk",
	RunE:  runExport,
}

func init() {
	filesCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&flagDir, "dir", "d", "", "Directory to export to")
	exportCmd.Flags().StringVarP(&flagFile, "file", "f", "", "File to export")
	exportCmd.Flags().StringVarP(&flagURL, "url", "u", "", "Blobcast URL of the file or directory manifest (required)")
	exportCmd.Flags().
		StringVar(&flagNodeGRPC, "node-grpc", cmd.GetEnvWithDefault("BLOBCAST_NODE_GRPC", "127.0.0.1:50051"), "gRPC address for a blobcast full node")

	exportCmd.MarkFlagRequired("url")
}

func runExport(command *cobra.Command, args []string) error {
	if flagDir == "" && flagFile == "" {
		return fmt.Errorf("please supply a directory or file to upload")
	}

	// Validate the target directory
	if flagDir != "" {
		flagDir = filepath.Clean(flagDir)
		if _, err := os.Stat(flagDir); os.IsNotExist(err) {
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(flagDir, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %v", flagDir, err)
			}
		} else if err != nil {
			return fmt.Errorf("error checking directory: %v", err)
		}
	}

	// Process encryption key if provided
	var encryptionKey []byte
	if flagEncryptionKey != "" {
		key, err := hex.DecodeString(flagEncryptionKey)
		if err != nil {
			return fmt.Errorf("error decoding encryption key: %v. Key must be a hex-encoded string", err)
		}
		if len(key) != 32 { // AES-256 requires a 32-byte key
			return fmt.Errorf("encryption key must be 32 bytes (64 hex characters) for AES-256")
		}
		encryptionKey = key
	}

	// Parse the manifest identifier from the URL
	id, err := types.BlobIdentifierFromURL(flagURL)
	if err != nil {
		return fmt.Errorf("error parsing manifest identifier: %v", err)
	}

	// initialize gRPC client
	conn, err := util.NewGRPCClient(flagNodeGRPC)
	if err != nil {
		return fmt.Errorf("error creating gRPC client: %v", err)
	}
	defer conn.Close()

	storageClient := pbStorageapisV1.NewStorageServiceClient(conn)

	// Initialize Noop DA client (not used for export)
	noopDA, _ := celestia.NewNoopDA()

	// create filesystem client
	filesystemClient := sync.NewFileSystemClient(storageClient, noopDA)

	switch {
	case flagDir != "":
		err = filesystemClient.ExportDirectory(context.Background(), id, flagDir, encryptionKey)
		if err != nil {
			return fmt.Errorf("error exporting directory: %v", err)
		}
		slog.Info("Successfully exported directory", "dir", flagDir, "blobcast_url", flagURL)
	case flagFile != "":
		err = filesystemClient.ExportFile(context.Background(), id, flagFile, encryptionKey)
		if err != nil {
			return fmt.Errorf("error exporting file: %v", err)
		}
		slog.Info("Successfully exported file", "file", flagFile, "blobcast_url", flagURL)
	}

	return nil
}

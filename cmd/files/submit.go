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
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/forma-dev/blobcast/pkg/util"
	"github.com/spf13/cobra"

	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a file or directory to Blobcast",
	RunE:  runSubmit,
}

func init() {
	filesCmd.AddCommand(submitCmd)

	submitCmd.Flags().StringVarP(&flagDir, "dir", "d", "", "Directory to submit")
	submitCmd.Flags().StringVarP(&flagFile, "file", "f", "", "File to submit")
	submitCmd.Flags().StringVar(&flagMaxBlobSize, "max-blob-size", "384KB", "Max file chunk size (e.g., 1.5MB, 1024KB)")
	submitCmd.Flags().StringVar(&flagMaxTxSize, "max-tx-size", "2MB", "Max transaction size (e.g., 32MB, 2097152B)")
	submitCmd.Flags().
		StringVar(&flagCelestiaAuth, "celestia-auth", cmd.GetEnvWithDefault("BLOBCAST_CELESTIA_NODE_AUTH_TOKEN", ""), "Celestia node auth token")
	submitCmd.Flags().
		StringVar(&flagCelestiaRPC, "celestia-rpc", cmd.GetEnvWithDefault("BLOBCAST_CELESTIA_NODE_RPC", "ws://localhost:26658"), "Celestia RPC node endpoint")
}

func runSubmit(command *cobra.Command, args []string) error {
	if flagDir == "" && flagFile == "" {
		return fmt.Errorf("please supply a directory or file to submit")
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

	// Parse blob and transaction size limits
	maxBlobSize, err := util.ParseSizeString(flagMaxBlobSize)
	if err != nil {
		return fmt.Errorf("error parsing max-blob-size: %v", err)
	}
	if maxBlobSize <= 0 {
		return fmt.Errorf("max-blob-size must be a positive value")
	}

	maxTxSize, err := util.ParseSizeString(flagMaxTxSize)
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
		Rpc:         flagCelestiaRPC,
		NamespaceId: state.CelestiaNamespace,
		AuthToken:   flagCelestiaAuth,
		MaxTxSize:   maxTxSize,
	}

	celestiaDA, err := celestia.NewCelestiaDA(daConfig)
	if err != nil {
		return fmt.Errorf("error creating Celestia client: %v", err)
	}

	latestCelestiaHeight, err := celestiaDA.LatestHeight(context.Background())
	if err != nil {
		return fmt.Errorf("error getting latest celestia height: %v", err)
	}
	slog.Debug("Latest Celestia height", "height", latestCelestiaHeight)

	celestiaHeader, err := celestiaDA.GetHeader(context.Background(), latestCelestiaHeight)
	if err != nil {
		return fmt.Errorf("error getting celestia header: %v", err)
	}

	celestiaChainID := celestiaHeader.ChainID()
	slog.Debug("Celestia chain ID", "chain_id", celestiaChainID)

	var network string
	switch celestiaChainID {
	case "mamo-1":
		network = "mammoth"
	case "mocha-4":
		network = "mocha"
	case "celestia":
		network = "celestia"
	default:
		return fmt.Errorf("unsupported celestia chain ID: %s", celestiaChainID)
	}

	slog.Info("Initializing with celestia network", "network", network)
	state.SetNetwork(network)

	// set app data dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	appDataDir := filepath.Join(homeDir, ".blobcast")

	slog.Info("Initializing with app data directory", "data_dir", appDataDir)
	state.SetDataDir(appDataDir)

	// create filesystem client
	filesystemClient := sync.NewFileSystemClient(storageClient, celestiaDA)

	switch {
	case flagDir != "":
		manifestIdentifier, _, err := filesystemClient.UploadDirectory(context.Background(), flagDir, maxBlobSize, encryptionKey)
		if err != nil {
			return fmt.Errorf("error submitting directory: %v", err)
		}
		slog.Info("Successfully submitted directory", "dir", flagDir, "blobcast_url", manifestIdentifier.URL())
	case flagFile != "":
		manifestIdentifier, _, err := filesystemClient.UploadFile(
			context.Background(),
			flagFile,
			filepath.Base(flagFile),
			maxBlobSize,
			encryptionKey,
		)
		if err != nil {
			return fmt.Errorf("error submitting file: %v", err)
		}
		slog.Info("Successfully submitted file", "file", flagFile, "blobcast_url", manifestIdentifier.URL())
	}

	return nil
}

package indexer

import (
	"os"
	"path/filepath"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagDataDir      string
	flagNodeGRPC     string
	flagStartHeight  uint64
	flagSyncInterval string
)

var indexerCmd = &cobra.Command{
	Use:   "indexer",
	Short: "Blobcast indexer commands",
	Long:  "Commands for running and managing the blobcast indexer service",
}

func init() {
	cmd.RootCmd.AddCommand(indexerCmd)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	defaultAppDataDir := filepath.Join(homeDir, ".blobcast")

	indexerCmd.PersistentFlags().
		StringVar(&flagDataDir, "data-dir", cmd.GetEnvWithDefault("BLOBCAST_DATA_DIR", defaultAppDataDir), "Path to the app data directory")

}

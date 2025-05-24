package node

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/spf13/cobra"
)

var (
	flagNetwork      string
	flagDataDir      string
	flagCelestiaAuth string
	flagCelestiaRPC  string
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Blobcast node commands",
}

func init() {
	cmd.RootCmd.AddCommand(nodeCmd)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	defaultAppDataDir := filepath.Join(homeDir, ".blobcast")

	nodeCmd.PersistentFlags().StringVar(&flagDataDir, "data-dir", cmd.GetEnvWithDefault("BLOBCAST_DATA_DIR", defaultAppDataDir), "Path to the app data directory")
	nodeCmd.PersistentFlags().StringVar(&flagNetwork, "network", cmd.GetEnvWithDefault("BLOBCAST_CELESTIA_NETWORK", "mocha"), "Celestia network")
	nodeCmd.PersistentFlags().
		StringVar(&flagCelestiaAuth, "celestia-auth", cmd.GetEnvWithDefault("BLOBCAST_CELESTIA_NODE_AUTH_TOKEN", ""), "Celestia node auth token")
	nodeCmd.PersistentFlags().
		StringVar(&flagCelestiaRPC, "celestia-rpc", cmd.GetEnvWithDefault("BLOBCAST_CELESTIA_NODE_RPC", "ws://localhost:26658"), "Celestia RPC node endpoint")
}

func initNode() error {
	slog.Info("Initializing with celestia network", "network", flagNetwork)
	state.SetNetwork(flagNetwork)

	slog.Info("Initializing with app data directory", "data_dir", flagDataDir)
	state.SetDataDir(flagDataDir)

	var chainID string
	switch flagNetwork {
	case "mocha":
		chainID = state.ChainIDMocha
	case "mammoth":
		chainID = state.ChainIDMammoth
	case "celestia":
		chainID = state.ChainIDCelestia
	default:
		return fmt.Errorf("invalid network: %s", flagNetwork)
	}

	slog.Info("Initializing blobcast chain", "chain_id", chainID)
	chainState, err := state.GetChainState()
	if err != nil {
		return fmt.Errorf("error getting chain state: %v", err)
	}

	chainState.SetChainID(chainID)

	var heightOffset uint64
	switch flagNetwork {
	case "mocha":
		heightOffset = state.StartHeightMocha
	case "mammoth":
		heightOffset = state.StartHeightMammoth
	case "celestia":
		heightOffset = state.StartHeightCelestia
	default:
		return fmt.Errorf("invalid network: %s", flagNetwork)
	}

	slog.Info("Setting celestia height offset", "height_offset", heightOffset)
	chainState.SetCelestiaHeightOffset(heightOffset)

	return nil
}

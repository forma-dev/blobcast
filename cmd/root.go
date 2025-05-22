package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
)

var (
	flagDir           string
	flagFile          string
	flagNamespace     string
	flagAuth          string
	flagRPC           string
	flagNetwork       string
	flagEncryptionKey string
	flagMaxBlobSize   string
	flagMaxTxSize     string
	flagURL           string
	flagDryRun        bool
	flagVerbose       bool
	flagNoResume      bool
	flagClearState    bool
	flagServePort     string
	flagGRPCAddr      string
)

var rootCmd = &cobra.Command{
	Use:   "blobcast",
	Short: "Blobcast uploads and downloads files using Celestia",
	Long: `Blobcast is a tool for uploading and downloading files using Celestia as a data availability layer.

It can upload an entire directory structure to Celestia and create a manifest that can be used
to download the files later. Files can optionally be encrypted for privacy.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(initLogging)
	// Add persistent flags that are shared across all commands
	rootCmd.PersistentFlags().StringVar(&flagNamespace, "namespace", getEnvWithDefault("BLOBCAST_NAMESPACE", "626C6F62636173742D31"), "Target namespace")
	rootCmd.PersistentFlags().StringVar(&flagNetwork, "network", getEnvWithDefault("BLOBCAST_CELESTIA_NETWORK", "mocha"), "Celestia network")
	rootCmd.PersistentFlags().StringVar(&flagAuth, "auth", getEnvWithDefault("BLOBCAST_CELESTIA_NODE_AUTH_TOKEN", ""), "Celestia node auth token")
	rootCmd.PersistentFlags().StringVar(&flagRPC, "rpc", getEnvWithDefault("BLOBCAST_CELESTIA_NODE_RPC", "ws://localhost:26658"), "Celestia RPC node endpoint")
	rootCmd.PersistentFlags().StringVar(&flagEncryptionKey, "key", "", "Hex-encoded 32-byte AES encryption key for encryption/decryption")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
}

func initLogging() {
	logLevel := slog.LevelInfo
	if flagVerbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: logLevel,
	})))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

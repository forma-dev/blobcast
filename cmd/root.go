package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
)

var (
	flagVerbose bool
)

var RootCmd = &cobra.Command{
	Use:   "blobcast",
	Short: "Blobcast is a minimal based rollup for publishing and retrieving files",
	Long: `Blobcast is a minimal based rollup for publishing and retrieving files on top of Celestia's data availability layer.

Files are chunked, submitted, and tracked by manifests. Blobcast nodes derive blockchain state from
Celestia blocks to provide content-addressable file storage with cryptographic integrity guarantees.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(initLogging)
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
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
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

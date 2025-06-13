package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"

	_ "net/http/pprof"
)

var (
	flagVerbose bool
	flagPprof   bool
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
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
	RootCmd.PersistentFlags().BoolVar(&flagPprof, "pprof", false, "Enable pprof")

	cobra.OnInitialize(initLogging)
	cobra.OnInitialize(initPprof)
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

func initPprof() {
	if flagPprof {
		go func() {
			http.ListenAndServe("localhost:6060", nil)
		}()
	}
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

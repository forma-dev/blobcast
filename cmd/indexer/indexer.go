package indexer

import (
	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagDbConnString string
	flagNodeGRPC     string
	flagStartHeight  uint64
)

var indexerCmd = &cobra.Command{
	Use:   "indexer",
	Short: "Blobcast indexer commands",
	Long:  "Commands for running and managing the blobcast indexer service",
}

func init() {
	cmd.RootCmd.AddCommand(indexerCmd)

	indexerCmd.PersistentFlags().
		StringVar(&flagDbConnString, "db-conn", cmd.GetEnvWithDefault("BLOBCAST_DB_CONNECTION_STRING", "postgres://postgres:secret@127.0.0.1:5432/postgres?sslmode=disable"), "Connection string for the indexer database")
}

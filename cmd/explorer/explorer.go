package explorer

import (
	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagAddr         string
	flagPort         string
	flagGatewayUrl   string
	flagDbConnString string
)

var explorerCmd = &cobra.Command{
	Use:   "explorer",
	Short: "Blobcast explorer commands",
	Long:  "Commands for running and managing the blobcast blockchain explorer web interface",
}

func init() {
	cmd.RootCmd.AddCommand(explorerCmd)

	explorerCmd.PersistentFlags().
		StringVar(&flagDbConnString, "db-conn", cmd.GetEnvWithDefault("BLOBCAST_DB_CONNECTION_STRING", "postgres://postgres:secret@127.0.0.1:5432/postgres?sslmode=disable"), "Connection string for the indexer database")
}

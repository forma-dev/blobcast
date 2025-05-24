package api

import (
	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagAddr     string
	flagPort     string
	flagNodeGRPC string
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Blobcast REST API server commands",
}

func init() {
	cmd.RootCmd.AddCommand(apiCmd)
}

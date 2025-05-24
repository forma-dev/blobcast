package gateway

import (
	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagAddr     string
	flagPort     string
	flagNodeGRPC string
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Blobcast gateway server commands",
}

func init() {
	cmd.RootCmd.AddCommand(gatewayCmd)
}

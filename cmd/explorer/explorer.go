package explorer

import (
	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagAddr       string
	flagPort       string
	flagIndexerDb  string
	flagGatewayUrl string
)

var explorerCmd = &cobra.Command{
	Use:   "explorer",
	Short: "Blobcast explorer commands",
	Long:  "Commands for running and managing the blobcast blockchain explorer web interface",
}

func init() {
	cmd.RootCmd.AddCommand(explorerCmd)
}

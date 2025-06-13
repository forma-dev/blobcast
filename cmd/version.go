package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/forma-dev/blobcast/pkg/version"
	"github.com/spf13/cobra"
)

var (
	outputJSON bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print version information for blobcast",
	Run:   runVersion,
}

func init() {
	RootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVar(&outputJSON, "json", false, "Output version info as JSON")
}

func runVersion(cmd *cobra.Command, args []string) {
	if outputJSON {
		info := version.Get()
		output, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(output))
	} else {
		Banner()
	}
}

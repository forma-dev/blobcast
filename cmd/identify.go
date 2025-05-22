package cmd

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/spf13/cobra"
)

var identifyCmd = &cobra.Command{
	Use:   "identify",
	Short: "Parse a blobcast URL and display its components",
	Long: `Parse a blobcast URL and display information about it.

This command extracts and displays the namespace, height, and commitment
from a blobcast URL, along with validation checks.

Example:
  blobcast identify --url "blobcast://bc8c7d91baz7sb2p8eyt6n33uqhg0m6jn3"`,
	RunE:    runIdentify,
	Example: `  blobcast identify --url "blobcast://bc8c7d91baz7sb2p8eyt6n33uqhg0m6jn3"`,
}

func init() {
	rootCmd.AddCommand(identifyCmd)
	identifyCmd.Flags().StringVarP(&flagURL, "url", "u", "", "Blobcast URL to identify (required)")
	identifyCmd.MarkFlagRequired("url")
}

func runIdentify(cmd *cobra.Command, args []string) error {
	if !strings.HasPrefix(flagURL, "blobcast://") {
		return fmt.Errorf("invalid URL format: URL must start with 'blobcast://'")
	}

	identifier, err := types.BlobIdentifierFromURL(flagURL)
	if err != nil {
		return fmt.Errorf("error parsing blobcast URL: %v", err)
	}

	height := identifier.Height
	commitment := hex.EncodeToString(identifier.Commitment)

	fmt.Println("\nBlobcast URL Information:")
	fmt.Println("========================")
	fmt.Println("URL:         ", flagURL)
	fmt.Println("Celestia Height:      ", height)
	fmt.Println("Share Commitment:  ", commitment)
	fmt.Println("\nURL Validation: ✓ Valid blobcast URL")
	fmt.Println("\nUsage Tips:")
	fmt.Println("  • This URL can be used with the 'download' command to retrieve content")
	fmt.Println("  • Example: blobcast download --url \"" + flagURL + "\" --dir ./output")

	return nil
}

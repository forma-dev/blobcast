package files

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/spf13/cobra"
)

var identifyCmd = &cobra.Command{
	Use:     "identify",
	Short:   "Parse a blobcast URL and display its components",
	RunE:    runIdentify,
	Example: `  blobcast identify --url "blobcast://bc46nekbubvwcyg34in9p85rfbzd38p6tmg3ep9z9asn2jw5eaefznmzv5yyql"`,
}

func init() {
	filesCmd.AddCommand(identifyCmd)
	identifyCmd.Flags().StringVarP(&flagURL, "url", "u", "", "Blobcast URL to identify (required)")
	identifyCmd.MarkFlagRequired("url")
}

func runIdentify(command *cobra.Command, args []string) error {
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

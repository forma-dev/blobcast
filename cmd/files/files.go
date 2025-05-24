package files

import (
	"github.com/forma-dev/blobcast/cmd"
	"github.com/spf13/cobra"
)

var (
	flagCelestiaAuth  string
	flagCelestiaRPC   string
	flagDir           string
	flagEncryptionKey string
	flagFile          string
	flagMaxBlobSize   string
	flagMaxTxSize     string
	flagNodeGRPC      string
	flagURL           string
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Blobcast files commands",
}

func init() {
	cmd.RootCmd.AddCommand(filesCmd)
	filesCmd.PersistentFlags().StringVar(&flagEncryptionKey, "key", "", "Hex-encoded 32-byte AES encryption key for encryption/decryption")
}

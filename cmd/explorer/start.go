package explorer

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/explorer"
	"github.com/forma-dev/blobcast/pkg/net/middleware"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the blobcast explorer web interface",
	Long:  "Start the web interface for browsing the blobcast blockchain",
	RunE:  runStart,
}

func init() {
	explorerCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&flagAddr, "addr", "a", "127.0.0.1", "Address to listen on")
	startCmd.Flags().StringVarP(&flagPort, "port", "p", "8082", "Port to listen on")
	startCmd.Flags().
		StringVar(&flagGatewayUrl, "gateway-url", cmd.GetEnvWithDefault("BLOBCAST_GATEWAY_URL", "http://localhost:8080"), "URL of the blobcast gateway")
}

func runStart(command *cobra.Command, args []string) error {
	indexDB, err := state.NewIndexerDatabase(flagDbConnString)
	if err != nil {
		return fmt.Errorf("error connecting to indexer database: %v", err)
	}
	defer indexDB.Close()

	cmd.Banner()

	server := explorer.NewServer(indexDB, strings.TrimRight(flagGatewayUrl, "/"))

	addr := flagAddr + ":" + flagPort
	slog.Info("Starting blobcast explorer", "addr", addr)

	handler := server.Router()
	handler.Use(middleware.LogRequestMiddleware)

	return http.ListenAndServe(addr, handler)
}

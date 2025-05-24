package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/api/rest"
	"github.com/forma-dev/blobcast/pkg/net/middleware"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a blobcast REST API server",
	RunE:  runStart,
}

func init() {
	apiCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&flagAddr, "addr", "a", "127.0.0.1", "Address to listen on")
	startCmd.Flags().StringVarP(&flagPort, "port", "p", "8081", "Port to listen on")
	startCmd.Flags().
		StringVar(&flagNodeGRPC, "node-grpc", cmd.GetEnvWithDefault("BLOBCAST_NODE_GRPC", "127.0.0.1:50051"), "gRPC address for a blobcast full node")
}

func runStart(command *cobra.Command, args []string) error {
	// initialize storage client
	keepaliveParams := keepalive.ClientParameters{
		Time:                15 * time.Minute,
		Timeout:             60 * time.Second,
		PermitWithoutStream: true,
	}
	conn, err := grpc.NewClient(
		flagNodeGRPC,
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)), // 1GB for now
	)
	if err != nil {
		return fmt.Errorf("error creating storage client: %v", err)
	}
	defer conn.Close()

	storageClient := pbStorageapisV1.NewStorageServiceClient(conn)
	rollupClient := pbRollupapisV1.NewRollupServiceClient(conn)

	// Create and start REST API server
	server := rest.NewServer(storageClient, rollupClient)

	addr := flagAddr + ":" + flagPort
	slog.Info("Starting Blobcast REST API server", "addr", addr)

	handler := server.Router()
	handler.Use(middleware.LogRequestMiddleware)

	return http.ListenAndServe(addr, handler)
}

package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/forma-dev/blobcast/pkg/api/rest"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the REST API server",
	RunE:  runAPI,
}

var (
	flagAPIPort string
)

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().StringVarP(&flagAPIPort, "port", "p", "8081", "Port to listen on")
	apiCmd.Flags().StringVar(&flagGRPCAddr, "node-grpc", getEnvWithDefault("BLOBCAST_NODE_GRPC", "127.0.0.1:50051"), "gRPC address for a blobcast full node")
}

func runAPI(cmd *cobra.Command, args []string) error {
	// initialize storage client
	keepaliveParams := keepalive.ClientParameters{
		Time:                15 * time.Minute,
		Timeout:             60 * time.Second,
		PermitWithoutStream: true,
	}
	conn, err := grpc.NewClient(
		flagGRPCAddr,
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

	addr := ":" + flagAPIPort
	slog.Info("Starting Blobcast REST API server", "addr", addr)

	handler := server.Router()
	handler.Use(logRequestMiddleware)

	return http.ListenAndServe(addr, handler)
}

package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/indexer"
	"github.com/forma-dev/blobcast/pkg/util"
	"github.com/spf13/cobra"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the blobcast indexer service",
	Long:  "Start the indexer service that indexes chain data by connecting to a blobcast node via gRPC",
	RunE:  runStart,
}

func init() {
	indexerCmd.AddCommand(startCmd)
	startCmd.Flags().StringVar(&flagNodeGRPC, "node-grpc",
		cmd.GetEnvWithDefault("BLOBCAST_NODE_GRPC", "127.0.0.1:50051"),
		"gRPC address for a blobcast full node")
	startCmd.Flags().StringVar(&flagSyncInterval, "sync-interval", "5s",
		"How often to check for new blocks")
	startCmd.Flags().Uint64Var(&flagStartHeight, "start-height", 0,
		"Block height to start indexing from (0 = from beginning)")
}

func runStart(command *cobra.Command, args []string) error {
	syncInterval, err := time.ParseDuration(flagSyncInterval)
	if err != nil {
		return fmt.Errorf("invalid sync interval: %v", err)
	}

	conn, err := util.NewGRPCClient(flagNodeGRPC)
	if err != nil {
		return fmt.Errorf("error creating gRPC client: %v", err)
	}
	defer conn.Close()

	storageClient := pbStorageapisV1.NewStorageServiceClient(conn)
	rollupClient := pbRollupapisV1.NewRollupServiceClient(conn)

	ctx := context.Background()

	indexerService, err := indexer.NewIndexerService(storageClient, rollupClient, flagStartHeight, flagDbConnString)
	if err != nil {
		return fmt.Errorf("error creating indexer service: %v", err)
	}

	slog.Info("Starting blobcast indexer",
		"node_grpc", flagNodeGRPC,
		"sync_interval", syncInterval,
		"start_height", flagStartHeight)

	return indexerService.Start(ctx, syncInterval)
}

package node

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbSyncapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/syncapis/v1"
)

var (
	flagRemoteGRPC string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Rapid sync a blobcast node with a trusted remote",
	RunE:  runSync,
}

func init() {
	nodeCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringVar(&flagRemoteGRPC, "remote-grpc-address", "127.0.0.1:50051", "gRPC address for a trusted blobcast full node to sync from")
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := initNode(); err != nil {
		return fmt.Errorf("error initializing node: %v", err)
	}

	chainState, err := state.GetChainState()
	if err != nil {
		return fmt.Errorf("error getting chain state: %v", err)
	}
	defer chainState.Close()

	// Create gRPC connection to the node
	keepaliveParams := keepalive.ClientParameters{
		Time:                15 * time.Minute,
		Timeout:             60 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(
		flagRemoteGRPC,
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)),
	)
	if err != nil {
		return fmt.Errorf("error creating connection to node: %v", err)
	}
	defer conn.Close()

	// Create gRPC clients
	rollupClient := pbRollupapisV1.NewRollupServiceClient(conn)
	syncClient := pbSyncapisV1.NewSyncServiceClient(conn)

	// Initialize Noop DA client (not used for remote sync)
	noopDA, _ := celestia.NewNoopDA()

	blobcastChain, err := sync.NewBlobcastChain(context.Background(), noopDA)
	if err != nil {
		return fmt.Errorf("error creating blobcast chain: %v", err)
	}

	slog.Info("Starting blobcast sync from remote node", "remote_grpc", flagRemoteGRPC)

	if err := blobcastChain.SyncChainFromGrpc(context.Background(), rollupClient, syncClient); err != nil {
		return fmt.Errorf("error syncing chain: %v", err)
	}

	return nil
}

package node

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/grpc/rollup"
	"github.com/forma-dev/blobcast/pkg/grpc/storage"
	grpcSync "github.com/forma-dev/blobcast/pkg/grpc/sync"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
	pbSyncapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/syncapis/v1"
)

var (
	flagGRPCPort    string
	flagGRPCAddress string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a blobcast node",
	RunE:  runStart,
}

func init() {
	nodeCmd.AddCommand(startCmd)
	startCmd.Flags().StringVar(&flagGRPCAddress, "grpc-address", "127.0.0.1", "gRPC server address")
	startCmd.Flags().StringVar(&flagGRPCPort, "grpc-port", "50051", "gRPC server port")
}

func runStart(cmd *cobra.Command, args []string) error {
	if err := initNode(); err != nil {
		return fmt.Errorf("error initializing node: %v", err)
	}

	// Initialize Celestia DA client
	daConfig := celestia.DAConfig{
		Rpc:         flagCelestiaRPC,
		NamespaceId: state.CelestiaNamespace,
		AuthToken:   flagCelestiaAuth,
		MaxTxSize:   0,
	}

	celestiaDA, err := celestia.NewCelestiaDA(daConfig)
	if err != nil {
		return fmt.Errorf("error creating Celestia client: %v", err)
	}

	// Get chain state for the rollup service
	chainState, err := state.GetChainState()
	if err != nil {
		return fmt.Errorf("error getting chain state: %v", err)
	}
	defer chainState.Close()

	blobcastChain, err := sync.NewBlobcastChain(context.Background(), celestiaDA)
	if err != nil {
		return fmt.Errorf("error creating blobcast chain: %v", err)
	}

	slog.Info("Starting gRPC server", "address", flagGRPCAddress, "port", flagGRPCPort)

	grpcAddr := fmt.Sprintf("%s:%s", flagGRPCAddress, flagGRPCPort)
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("error listening on %s: %v", grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	pbStorageapisV1.RegisterStorageServiceServer(grpcServer, storage.NewStorageServiceServer())
	pbRollupapisV1.RegisterRollupServiceServer(grpcServer, rollup.NewRollupServiceServer(chainState))
	pbSyncapisV1.RegisterSyncServiceServer(grpcServer, grpcSync.NewSyncServiceServer(chainState))

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			slog.Error("error serving gRPC server", "error", err)
		}
	}()

	slog.Info("Starting blobcast chain")

	if err := blobcastChain.SyncChain(context.Background()); err != nil {
		return fmt.Errorf("error syncing chain: %v", err)
	}

	return nil
}

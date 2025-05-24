package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/grpc/rollup"
	"github.com/forma-dev/blobcast/pkg/grpc/storage"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/sync"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

var (
	flagGRPCPort    string
	flagGRPCAddress string
)

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start a blobcast node",
	RunE:    runStart,
	PreRunE: initStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVar(&flagGRPCAddress, "grpc-address", "127.0.0.1", "gRPC server address")
	startCmd.Flags().StringVar(&flagGRPCPort, "grpc-port", "50051", "gRPC server port")
}

func initStart(cmd *cobra.Command, args []string) error {
	var chainID string
	switch flagNetwork {
	case "mocha":
		chainID = state.ChainIDMocha
	case "mammoth":
		chainID = state.ChainIDMammoth
	case "celestia":
		chainID = state.ChainIDCelestia
	default:
		return fmt.Errorf("invalid network: %s", flagNetwork)
	}

	slog.Debug("Initializing blobcast chain", "chain_id", chainID)
	chainState, err := state.GetChainState()
	if err != nil {
		return fmt.Errorf("error getting chain state: %v", err)
	}

	chainState.SetChainID(chainID)

	switch flagNetwork {
	case "mocha":
		chainState.SetCelestiaHeightOffset(state.StartHeightMocha)
	case "mammoth":
		chainState.SetCelestiaHeightOffset(state.StartHeightMammoth)
	case "celestia":
		chainState.SetCelestiaHeightOffset(state.StartHeightCelestia)
	default:
		return fmt.Errorf("invalid network: %s", flagNetwork)
	}

	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
	// Initialize Celestia DA client
	daConfig := celestia.DAConfig{
		Rpc:         flagRPC,
		NamespaceId: flagNamespace,
		AuthToken:   flagAuth,
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

	blobcastChain, err := sync.NewBlobcastChain(context.Background(), celestiaDA)
	if err != nil {
		return fmt.Errorf("error creating blobcast chain: %v", err)
	}

	slog.Info("Starting gRPC server", "address", flagGRPCAddress, "port", flagGRPCPort)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", flagGRPCAddress, flagGRPCPort))
	if err != nil {
		return fmt.Errorf("error listening on %s:%s: %v", flagGRPCAddress, flagGRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	pbStorageapisV1.RegisterStorageServiceServer(grpcServer, storage.NewStorageServiceServer())
	pbRollupapisV1.RegisterRollupServiceServer(grpcServer, rollup.NewRollupServiceServer(chainState))

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

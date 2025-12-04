package rollup

import (
	"context"
	"log/slog"

	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/sync"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
)

type BlockHeaderSubscriber interface {
	SubscribeToNewHeaders() *sync.Subscription
	Unsubscribe(*sync.Subscription)
}

type RollupServiceServer struct {
	pbRollupapisV1.UnimplementedRollupServiceServer
	chainState       *state.ChainState
	headerSubscriber BlockHeaderSubscriber
}

func NewRollupServiceServer(chainState *state.ChainState, headerSubscriber BlockHeaderSubscriber) *RollupServiceServer {
	return &RollupServiceServer{
		chainState:       chainState,
		headerSubscriber: headerSubscriber,
	}
}

func (s *RollupServiceServer) GetBlockByHeight(
	ctx context.Context,
	req *pbRollupapisV1.GetBlockByHeightRequest,
) (*pbRollupapisV1.GetBlockByHeightResponse, error) {
	block, err := s.chainState.GetBlock(req.Height)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get block by height: %v", err)
	}

	if block == nil {
		return nil, status.Errorf(codes.NotFound, "block not found")
	}

	return &pbRollupapisV1.GetBlockByHeightResponse{
		Block: block.Proto(),
	}, nil
}

func (s *RollupServiceServer) GetBlockByHash(
	ctx context.Context,
	req *pbRollupapisV1.GetBlockByHashRequest,
) (*pbRollupapisV1.GetBlockByHashResponse, error) {
	if len(req.Hash) != 32 {
		return nil, status.Errorf(codes.InvalidArgument, "hash must be 32 bytes, got %d", len(req.Hash))
	}

	block, err := s.chainState.GetBlockByHash(state.HashKey(req.Hash))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get block by hash: %v", err)
	}

	if block == nil {
		return nil, status.Errorf(codes.NotFound, "block not found")
	}

	return &pbRollupapisV1.GetBlockByHashResponse{
		Block: block.Proto(),
	}, nil
}

func (s *RollupServiceServer) GetLatestBlock(
	ctx context.Context,
	req *pbRollupapisV1.GetLatestBlockRequest,
) (*pbRollupapisV1.GetLatestBlockResponse, error) {
	finalizedHeight, err := s.chainState.FinalizedHeight()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get finalized height: %v", err)
	}

	if finalizedHeight == 0 {
		return nil, status.Errorf(codes.NotFound, "no blocks available")
	}

	block, err := s.chainState.GetBlock(finalizedHeight)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get latest block: %v", err)
	}

	if block == nil {
		return nil, status.Errorf(codes.NotFound, "block not found")
	}

	return &pbRollupapisV1.GetLatestBlockResponse{
		Block: block.Proto(),
	}, nil
}

func (s *RollupServiceServer) GetChainInfo(
	ctx context.Context,
	req *pbRollupapisV1.GetChainInfoRequest,
) (*pbRollupapisV1.GetChainInfoResponse, error) {
	chainID, err := s.chainState.ChainID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chain ID: %v", err)
	}

	finalizedHeight, err := s.chainState.FinalizedHeight()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get finalized height: %v", err)
	}

	celestiaOffset, err := s.chainState.CelestiaHeightOffset()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get celestia height offset: %v", err)
	}

	return &pbRollupapisV1.GetChainInfoResponse{
		ChainId:              chainID,
		FinalizedHeight:      finalizedHeight,
		CelestiaHeightOffset: celestiaOffset,
	}, nil
}

func (s *RollupServiceServer) SubscribeToHeaders(
	req *pbRollupapisV1.SubscribeToHeadersRequest,
	stream pbRollupapisV1.RollupService_SubscribeToHeadersServer,
) error {
	ctx := stream.Context()

	slog.Info("New header subscription", "start_height", req.StartHeight)

	// Subscribe to new headers
	sub := s.headerSubscriber.SubscribeToNewHeaders()
	defer s.headerSubscriber.Unsubscribe(sub)
	headerChan := sub.Ch

	// If start height is specified, send historical headers first
	if req.StartHeight > 0 {
		finalizedHeight, err := s.chainState.FinalizedHeight()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get finalized height: %v", err)
		}

		// Stream historical headers from start height to current finalized height
		for height := req.StartHeight; height <= finalizedHeight; height++ {
			block, err := s.chainState.GetBlock(height)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to get block at height %d: %v", height, err)
			}

			if block == nil {
				return status.Errorf(codes.NotFound, "block at height %d not found", height)
			}

			if err := stream.Send(&pbRollupapisV1.SubscribeToHeadersResponse{
				Header: block.Header.Proto(),
				Hash:   block.Hash().Bytes(),
			}); err != nil {
				return status.Errorf(codes.Internal, "failed to send header: %v", err)
			}

			// Check if client disconnected
			select {
			case <-ctx.Done():
				slog.Info("Client disconnected during historical sync")
				return ctx.Err()
			default:
			}
		}

		slog.Info("Finished streaming historical headers", "from", req.StartHeight, "to", finalizedHeight)
	}

	// Stream new headers as they are finalized
	for {
		select {
		case <-ctx.Done():
			slog.Info("Client disconnected from header subscription")
			return ctx.Err()
		case header, ok := <-headerChan:
			if !ok {
				// Channel was closed, likely server shutting down
				return status.Errorf(codes.Unavailable, "subscription channel closed")
			}

			if err := stream.Send(&pbRollupapisV1.SubscribeToHeadersResponse{
				Header: header.Proto(),
				Hash:   header.Hash().Bytes(),
			}); err != nil {
				return status.Errorf(codes.Internal, "failed to send header: %v", err)
			}

			slog.Debug("Sent header to subscriber", "height", header.Height, "hash", header.Hash())
		}
	}
}

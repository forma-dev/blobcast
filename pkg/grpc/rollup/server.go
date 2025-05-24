package rollup

import (
	"context"

	"github.com/forma-dev/blobcast/pkg/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
)

type RollupServiceServer struct {
	pbRollupapisV1.UnimplementedRollupServiceServer
	chainState *state.ChainState
}

func NewRollupServiceServer(chainState *state.ChainState) *RollupServiceServer {
	return &RollupServiceServer{
		chainState: chainState,
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

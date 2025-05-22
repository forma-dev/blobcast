package state

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/forma-dev/blobcast/pkg/crypto/mmr"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
)

const (
	keyChainID              = "chain:id"
	keyFinalizedHeight      = "chain:height:finalized"
	keyCelestiaHeightOffset = "celestia:height_offset"
)

var (
	chainState      *ChainState
	chainStateMutex sync.Mutex
)

type ChainState struct {
	db             *pebble.DB
	blockPrefix    []byte
	stateMMRPrefix []byte
}

func GetChainState() (*ChainState, error) {
	chainStateMutex.Lock()
	defer chainStateMutex.Unlock()

	if chainState != nil {
		return chainState, nil
	}

	chainStateLocal, err := openChainState()
	if err != nil {
		return nil, err
	}

	chainState = chainStateLocal
	return chainStateLocal, nil
}

func openChainState() (*ChainState, error) {
	dbPath, err := getStateDbPath(ChainStateDb)
	if err != nil {
		return nil, err
	}

	db, err := pebble.Open(dbPath, nil)
	if err != nil {
		return nil, err
	}

	return &ChainState{
		db:             db,
		blockPrefix:    []byte("block:"),
		stateMMRPrefix: []byte("state_mmr:"),
	}, nil
}

func (s *ChainState) Close() error {
	return s.db.Close()
}

func (s *ChainState) ChainID() (string, error) {
	id, closer, err := s.db.Get([]byte(keyChainID))
	if err != nil {
		return "", err
	}
	defer closer.Close()
	return string(id), nil
}

func (s *ChainState) SetChainID(id string) error {
	return s.db.Set([]byte(keyChainID), []byte(id), nil)
}

func (s *ChainState) CelestiaHeightOffset() (uint64, error) {
	offset, closer, err := s.db.Get([]byte(keyCelestiaHeightOffset))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	defer closer.Close()

	return binary.BigEndian.Uint64(offset), nil
}

func (s *ChainState) SetCelestiaHeightOffset(offset uint64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, offset)
	return s.db.Set([]byte(keyCelestiaHeightOffset), buf, nil)
}

func (s *ChainState) FinalizedHeight() (uint64, error) {
	height, closer, err := s.db.Get([]byte(keyFinalizedHeight))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	defer closer.Close()

	return binary.BigEndian.Uint64(height), nil
}

func (s *ChainState) SetFinalizedHeight(height uint64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return s.db.Set([]byte(keyFinalizedHeight), buf, nil)
}

func (s *ChainState) PutBlock(height uint64, block *types.Block) error {
	key := s.blockKey(height)
	blockBytes, err := proto.Marshal(block.Proto())
	if err != nil {
		return err
	}
	return s.db.Set(key, blockBytes, nil)
}

func (s *ChainState) GetBlock(height uint64) (*types.Block, error) {
	key := s.blockKey(height)
	blockBytes, closer, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	pbBlock := &pbRollupV1.Block{}
	if err := proto.Unmarshal(blockBytes, pbBlock); err != nil {
		return nil, err
	}

	return types.BlockFromProto(pbBlock), nil
}

func (s *ChainState) blockKey(height uint64) []byte {
	k := make([]byte, len(s.blockPrefix)+8)
	copy(k, s.blockPrefix)
	binary.BigEndian.PutUint64(k[len(s.blockPrefix):], height)
	return k
}

func (s *ChainState) PutStateMMR(height uint64, mmr *mmr.MMR) error {
	if height == 0 {
		return nil
	}

	key := s.stateMMRKey(height)
	return s.db.Set(key, mmr.Snapshot(), nil)
}

func (s *ChainState) GetStateMMR(height uint64) (*mmr.MMR, error) {
	if height == 0 {
		return mmr.NewMMR(), nil
	}

	key := s.stateMMRKey(height)
	mmrBytes, closer, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	mmr, err := mmr.NewMMRFromSnapshot(mmrBytes)
	if err != nil {
		return nil, err
	}
	return mmr, nil
}

func (s *ChainState) stateMMRKey(height uint64) []byte {
	k := make([]byte, len(s.stateMMRPrefix)+8)
	copy(k, s.stateMMRPrefix)
	binary.BigEndian.PutUint64(k[len(s.stateMMRPrefix):], height)
	return k
}

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/forma-dev/blobcast/pkg/crypto/mmr"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

const (
	keyChainID              = "meta:chainid"
	keyFinalizedHeight      = "meta:finalized"
	keyCelestiaHeightOffset = "meta:celestia_offset"
)

var (
	chainState      *ChainState
	chainStateMutex sync.Mutex
)

type HashKey [32]byte

type ChainState struct {
	db                 *pebble.DB
	pendingTransaction *ChainStateTransaction
	blockPrefix        []byte
	stateMMRPrefix     []byte
	chunkPrefix        []byte
	fileManifestPrefix []byte
	dirManifestPrefix  []byte
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
		db:                 db,
		pendingTransaction: nil,
		blockPrefix:        []byte("blk:"),
		stateMMRPrefix:     []byte("mmr:"),
		chunkPrefix:        []byte("chk:"),
		fileManifestPrefix: []byte("man:f:"),
		dirManifestPrefix:  []byte("man:d:"),
	}, nil
}

func (s *ChainState) Close() error {
	return s.db.Close()
}

func (s *ChainState) BeginTransaction() (*ChainStateTransaction, error) {
	if s.pendingTransaction != nil {
		return nil, fmt.Errorf("transaction already in progress")
	}

	tx := NewChainStateTransaction(s)
	s.pendingTransaction = tx
	return tx, nil
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

func (s *ChainState) GetBlock(height uint64) (*types.Block, error) {
	key := prefixHeightKey(height, s.blockPrefix)
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

func (s *ChainState) GetStateMMR(height uint64) (*mmr.MMR, error) {
	if height == 0 {
		return mmr.NewMMR(), nil
	}

	key := prefixHeightKey(height, s.stateMMRPrefix)
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

func (s *ChainState) GetChunk(key HashKey) ([]byte, bool, error) {
	value, closer, err := s.db.Get(prefixKey(key[:], s.chunkPrefix))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer closer.Close()

	// safe copy
	safeValue := make([]byte, len(value))
	copy(safeValue, value)

	return safeValue, true, nil
}

func (s *ChainState) GetDirectoryManifest(id *types.BlobIdentifier) (*pbStorageV1.DirectoryManifest, bool, error) {
	key := prefixKey(id.Bytes(), s.dirManifestPrefix)

	data, closer, err := s.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer closer.Close()

	// Copy the data since the slice becomes invalid after closer.Close()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	manifest := &pbStorageV1.DirectoryManifest{}
	if err := proto.Unmarshal(dataCopy, manifest); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached directory manifest: %w", err)
	}

	return manifest, true, nil
}

func (s *ChainState) GetFileManifest(id *types.BlobIdentifier) (*pbStorageV1.FileManifest, bool, error) {
	key := prefixKey(id.Bytes(), s.fileManifestPrefix)

	data, closer, err := s.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer closer.Close()

	// Copy the data since the slice becomes invalid after closer.Close()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	manifest := &pbStorageV1.FileManifest{}
	if err := proto.Unmarshal(dataCopy, manifest); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached file manifest: %w", err)
	}

	return manifest, true, nil
}

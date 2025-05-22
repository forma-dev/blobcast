package state

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/cockroachdb/pebble"
)

var (
	chunkState      *ChunkState
	chunkStateMutex sync.Mutex
)

type ChunkKey [32]byte

type ChunkState struct {
	db         *pebble.DB
	dataPrefix []byte
	seenPrefix []byte
}

func GetChunkState() (*ChunkState, error) {
	chunkStateMutex.Lock()
	defer chunkStateMutex.Unlock()

	if chunkState != nil {
		return chunkState, nil
	}

	chunkStateLocal, err := openChunkState()
	if err != nil {
		return nil, err
	}

	chunkState = chunkStateLocal
	return chunkStateLocal, nil
}

func openChunkState() (*ChunkState, error) {
	dbPath, err := getStateDbPath(ChunkStateDb)
	if err != nil {
		return nil, err
	}

	db, err := pebble.Open(dbPath, nil)
	if err != nil {
		return nil, err
	}

	return &ChunkState{
		db:         db,
		dataPrefix: []byte("data:"),
		seenPrefix: []byte("seen:"),
	}, nil
}

func (s *ChunkState) Get(key ChunkKey) ([]byte, bool, error) {
	value, closer, err := s.db.Get(prefixKey(key[:], s.dataPrefix))
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

func (s *ChunkState) Set(key ChunkKey, value []byte) error {
	return s.db.Set(prefixKey(key[:], s.dataPrefix), value, nil)
}

func (s *ChunkState) seenKey(chunk ChunkKey, height uint64) []byte {
	k := make([]byte, len(s.seenPrefix)+len(chunk)+8)
	copy(k, s.seenPrefix)
	copy(k[len(s.seenPrefix):], chunk[:])
	binary.BigEndian.PutUint64(k[len(s.seenPrefix)+len(chunk):], height)
	return k
}

func (s *ChunkState) MarkSeen(key ChunkKey, height uint64) error {
	return s.db.Set(s.seenKey(key, height), nil, nil)
}

func (s *ChunkState) SeenAtHeight(key ChunkKey, height uint64) (bool, error) {
	// Create an iterator to find all entries for this chunk
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: append(s.seenPrefix, key[:]...),
		UpperBound: append(append(s.seenPrefix, key[:]...), 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF),
	})
	if err != nil {
		return false, err
	}
	defer iter.Close()

	// Check if there are any entries
	valid := iter.First()
	if !valid {
		return false, nil
	}

	// Iterate through all entries for this chunk
	for ; valid; valid = iter.Next() {
		k := iter.Key()
		if len(k) < len(s.seenPrefix)+len(key)+8 {
			continue
		}

		// Extract the height from the key
		seenHeight := binary.BigEndian.Uint64(k[len(s.seenPrefix)+len(key):])

		// If we find an entry with height <= requested height, return true
		if seenHeight <= height {
			return true, nil
		}
	}

	return false, nil
}

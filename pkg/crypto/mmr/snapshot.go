package mmr

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

type Snapshot []byte

func NewMMRFromSnapshot(snapshot Snapshot) (*MMR, error) {
	mmr := NewMMR()
	if err := mmr.Restore(snapshot); err != nil {
		return nil, err
	}
	return mmr, nil
}

func (mmr *MMR) Snapshot() Snapshot {
	numPeaks := len(mmr.peaks)
	numNodes := int(mmr.size)
	numLeaves := int(mmr.numLeaves)

	// Calculate size: numPeaks + peaks + numNodes + nodes + numLeaves + leafIndexes
	size := 8 + (numPeaks * 32) + 8 + (numNodes * 32) + 8 + (numLeaves * 32)
	buf := make([]byte, size)

	offset := 0

	// Write numPeaks
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(numPeaks))
	offset += 8

	// Write peaks
	for _, peak := range mmr.peaks {
		copy(buf[offset:offset+32], peak[:])
		offset += 32
	}

	// Write numNodes
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(numNodes))
	offset += 8

	// Write nodes in position order
	for pos := uint64(0); pos < mmr.size; pos++ {
		if hash, exists := mmr.nodes[pos]; exists {
			copy(buf[offset:offset+32], hash[:])
		} else {
			// Write zero hash for missing nodes
			var zeroHash crypto.Hash
			copy(buf[offset:offset+32], zeroHash[:])
		}
		offset += 32
	}

	// Write numLeaves
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(numLeaves))
	offset += 8

	// Write leafIndexes (sorted by index value)
	type hashIndex struct {
		hash  crypto.Hash
		index uint64
	}

	hashIndexes := make([]hashIndex, 0, len(mmr.leafIndexes))
	for hash, index := range mmr.leafIndexes {
		hashIndexes = append(hashIndexes, hashIndex{hash: hash, index: index})
	}

	sort.Slice(hashIndexes, func(i, j int) bool {
		return hashIndexes[i].index < hashIndexes[j].index
	})

	for _, hi := range hashIndexes {
		copy(buf[offset:offset+32], hi.hash[:])
		offset += 32
	}

	return buf
}

func (mmr *MMR) Restore(snapshot Snapshot) error {
	if len(snapshot) < 32 {
		return fmt.Errorf("invalid snapshot: expected at least 32 bytes, got %d", len(snapshot))
	}

	offset := 0

	// Read numPeaks
	numPeaks := binary.LittleEndian.Uint64(snapshot[offset : offset+8])
	offset += 8

	// Read peaks
	peaks := make([]crypto.Hash, numPeaks)
	for i := range peaks {
		if offset+32 > len(snapshot) {
			return fmt.Errorf("invalid snapshot: not enough data for peaks")
		}
		copy(peaks[i][:], snapshot[offset:offset+32])
		offset += 32
	}

	// Read numNodes
	if offset+8 > len(snapshot) {
		return fmt.Errorf("invalid snapshot: not enough data for numNodes")
	}
	numNodes := binary.LittleEndian.Uint64(snapshot[offset : offset+8])
	offset += 8

	// Read nodes
	nodes := make(map[uint64]crypto.Hash)
	for pos := uint64(0); pos < numNodes; pos++ {
		if offset+32 > len(snapshot) {
			return fmt.Errorf("invalid snapshot: not enough data for node %d", pos)
		}

		var hash crypto.Hash
		copy(hash[:], snapshot[offset:offset+32])
		offset += 32

		// Only store non-zero hashes
		if !hash.IsZero() {
			nodes[pos] = hash
		}
	}

	// Read numLeaves
	numLeaves := binary.LittleEndian.Uint64(snapshot[offset : offset+8])
	offset += 8

	// Read leafIndexes
	leafIndexes := make(map[crypto.Hash]uint64)
	for i := uint64(0); i < numLeaves; i++ {
		if offset+32 > len(snapshot) {
			return fmt.Errorf("invalid snapshot: not enough data for leaf %d", i)
		}
		var leafHash crypto.Hash
		copy(leafHash[:], snapshot[offset:offset+32])
		leafIndexes[leafHash] = i
		offset += 32
	}

	// Restore state
	mmr.numLeaves = numLeaves
	mmr.size = numNodes
	mmr.peaks = peaks
	mmr.nodes = nodes
	mmr.leafIndexes = leafIndexes

	return nil
}

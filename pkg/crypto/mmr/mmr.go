package mmr

import (
	"encoding/binary"
	"fmt"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

type MMR struct {
	peaks       []crypto.Hash
	nodes       map[uint64]crypto.Hash
	leafIndexes map[crypto.Hash]uint64
	numLeaves   uint64
	size        uint64
}

func NewMMR() *MMR {
	return &MMR{
		peaks:       []crypto.Hash{},
		nodes:       make(map[uint64]crypto.Hash),
		leafIndexes: make(map[crypto.Hash]uint64),
		numLeaves:   0,
		size:        0,
	}
}

func (mmr *MMR) NumLeaves() uint64 {
	return mmr.numLeaves
}

func (mmr *MMR) NumPeaks() uint64 {
	return uint64(len(mmr.peaks))
}

func (mmr *MMR) Size() uint64 {
	return mmr.size
}

func (mmr *MMR) LeafIndex(leaf []byte) (uint64, error) {
	leafHash := crypto.HashBytes(leaf)
	index, ok := mmr.leafIndexes[leafHash]
	if !ok {
		return 0, fmt.Errorf("leaf not found")
	}
	return index, nil
}

func (mmr *MMR) Root() crypto.Hash {
	if mmr.size == 0 {
		return crypto.Hash{}
	}
	return mmr.bagPeaksWithSize(mmr.peaksLR(), mmr.size)
}

func (mmr *MMR) AddLeaf(leaf []byte) crypto.Hash {
	leafHash := crypto.HashBytes(leaf)
	leafIndex := mmr.numLeaves
	leafPos := mmr.size
	mmr.size++

	leafPosHash := mmr.hashWithPosition(leafPos, leafHash.Bytes())

	// Store the leaf node
	mmr.nodes[leafPos] = leafPosHash
	mmr.leafIndexes[leafHash] = leafIndex

	// Use the original carry algorithm
	carry := leafPosHash
	height := uint64(0)

	for {
		if height >= uint64(len(mmr.peaks)) {
			mmr.peaks = append(mmr.peaks, carry)
			break
		}

		if mmr.peaks[height].IsZero() {
			mmr.peaks[height] = carry
			break
		}

		// Merge with existing peak
		leftHash := mmr.peaks[height]
		rightHash := carry

		// Create parent node
		parentPos := mmr.size
		mmr.size++
		parentHash := mmr.hashWithPosition(parentPos, leftHash.Bytes(), rightHash.Bytes())

		// Store the parent node
		mmr.nodes[parentPos] = parentHash

		// Clear the peak and continue with the parent
		mmr.peaks[height] = crypto.Hash{}
		carry = parentHash
		height++
	}

	mmr.numLeaves++
	return mmr.Root()
}

func (mmr *MMR) hashWithPosition(pos uint64, data ...[]byte) crypto.Hash {
	posBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(posBytes, pos)
	data = append([][]byte{posBytes}, data...)
	return crypto.HashBytes(data...)
}

func (mmr *MMR) bagPeaksWithSize(peaks []crypto.Hash, size uint64) crypto.Hash {
	peaksRoot := mmr.bagPeaks(peaks)
	sizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBytes, size)
	return crypto.HashBytes(sizeBytes, peaksRoot.Bytes())
}

// bagPeaks combines peaks to create the final root
// peaks are bagged (folded) right to left
func (mmr *MMR) bagPeaks(peaks []crypto.Hash) crypto.Hash {
	revPeaks := make([]crypto.Hash, len(peaks))
	for i, peak := range peaks {
		revPeaks[len(peaks)-1-i] = peak
	}
	var accum crypto.Hash
	for _, peak := range revPeaks {
		if peak.IsZero() {
			continue
		}
		if accum.IsZero() {
			accum = peak
		} else {
			accum = crypto.HashBytes(peak.Bytes(), accum.Bytes())
		}
	}
	return accum
}

func (mmr *MMR) peaksLR() []crypto.Hash {
	peaks := make([]crypto.Hash, 0)
	for _, peak := range mmr.peaks {
		if !peak.IsZero() {
			peaks = append([]crypto.Hash{peak}, peaks...)
		}
	}
	return peaks
}

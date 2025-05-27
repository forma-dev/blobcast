package mmr

import (
	"fmt"
	"log/slog"
	"math"
	"math/bits"
	"slices"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

type InclusionProof struct {
	LeafIndex uint64
	LeafHash  crypto.Hash
	Siblings  []crypto.Hash
	Peaks     []crypto.Hash
	MMRSize   uint64
}

func (mmr *MMR) GenerateInclusionProof(leafIndex uint64) (*InclusionProof, error) {
	if leafIndex >= mmr.numLeaves {
		return nil, fmt.Errorf("leaf index %d out of range (max: %d)", leafIndex, mmr.numLeaves-1)
	}

	// Convert leaf index to MMR position
	leafPos := leafIndextoMMRIndex(leafIndex)

	// Verify this is actually a leaf position
	if !isLeaf(leafPos) {
		return nil, fmt.Errorf("position %d is not a leaf", leafPos)
	}

	// Get the leaf hash
	leafHash, exists := mmr.nodes[leafPos]
	if !exists {
		return nil, fmt.Errorf("no element at position %d", leafPos)
	}

	// Build the sibling path
	var siblings []crypto.Hash
	currentPos := leafPos

	// Walk up the tree collecting siblings
	for {
		parentPos, siblingPos := family(currentPos)

		// If parent is beyond MMR size, we've reached a peak
		if parentPos >= mmr.size {
			break
		}

		// Get the sibling hash
		if siblingHash, exists := mmr.nodes[siblingPos]; exists {
			siblings = append(siblings, siblingHash)
		} else {
			return nil, fmt.Errorf("missing sibling at position %d", siblingPos)
		}

		currentPos = parentPos
	}

	return &InclusionProof{
		LeafIndex: leafIndex,
		LeafHash:  leafHash,
		Siblings:  siblings,
		Peaks:     mmr.peaksLR(),
		MMRSize:   mmr.size,
	}, nil
}

func (mmr *MMR) VerifyInclusionProof(proof *InclusionProof) bool {
	nodePos := leafIndextoMMRIndex(proof.LeafIndex)

	// Start with the leaf hash and walk up using siblings
	currentHash := proof.LeafHash
	currentPos := nodePos

	// Walk up the tree using the sibling path to reconstruct the peak
	for i, sibling := range proof.Siblings {
		parentPos, siblingPos := family(currentPos)

		// Determine if we're left or right child based on sibling position
		if isLeftSibling(siblingPos) {
			// Sibling is left, so we are right
			currentHash = mmr.hashWithPosition(parentPos, sibling.Bytes(), currentHash.Bytes())
		} else {
			// Sibling is right, so we are left
			currentHash = mmr.hashWithPosition(parentPos, currentHash.Bytes(), sibling.Bytes())
		}

		currentPos = parentPos

		// If this is the last sibling, we should have reached a peak
		if i == len(proof.Siblings)-1 {
			break
		}
	}

	// At this point, currentHash should be one of the peaks in the proof
	peakFound := false
	for _, peak := range proof.Peaks {
		if peak.Equal(currentHash) {
			peakFound = true
			break
		}
	}

	if !peakFound {
		return false
	}

	// get the expected peaks at the point in time the proof was generated
	expectedPeakPositions := peakPositionsAtSize(proof.MMRSize)
	var expectedPeaks []crypto.Hash

	for _, peakPos := range expectedPeakPositions {
		if peakHash, exists := mmr.nodes[peakPos]; exists {
			expectedPeaks = append(expectedPeaks, peakHash)
		} else {
			// If we don't have a peak that should exist, the proof is invalid
			return false
		}
	}

	// proof peaks should match the expected peaks
	if !slices.Equal(proof.Peaks, expectedPeaks) {
		slog.Error("proof peaks != expected peaks", "proofPeaks", proof.Peaks, "expectedPeaks", expectedPeaks)
		return false
	}

	// verify the proof root matches the expected root (somewhat redundant)
	expectedHistoricalRoot := mmr.bagPeaksWithSize(expectedPeaks, proof.MMRSize)
	proofHistoricalRoot := mmr.bagPeaksWithSize(proof.Peaks, proof.MMRSize)

	return expectedHistoricalRoot.Equal(proofHistoricalRoot)
}

func peakPositionsAtSize(size uint64) []uint64 {
	var peaks []uint64
	pos := uint64(1)
	remaining := size

	for remaining > 0 {
		// Find max height h such that 2^(h+1) - 1 ≤ remaining
		h := int(math.Floor(math.Log2(float64(remaining+1)))) - 1
		treeSize := uint64(1)<<(h+1) - 1
		peakPos := pos + treeSize - 1
		peaks = append(peaks, peakPos-1)

		// Move to the next unprocessed node
		pos += treeSize
		remaining -= treeSize
	}

	return peaks
}

func leafIndextoMMRIndex(leafIndex uint64) uint64 {
	return 2*leafIndex - countOnes(leafIndex)
}

func countOnes(n uint64) uint64 {
	count := uint64(0)
	for n != 0 {
		count++
		n &= n - 1
	}
	return count
}

func peakMapHeight(size uint64) (uint64, uint64) {
	if size == 0 {
		return 0, 0
	}

	peakSize := ^uint64(0) >> bits.LeadingZeros64(size)

	peakMap := uint64(0)
	for peakSize != 0 {
		peakMap <<= 1
		if size >= peakSize {
			size -= peakSize
			peakMap |= 1
		}
		peakSize >>= 1
	}
	return peakMap, size
}

func isLeaf(pos uint64) bool {
	_, height := peakMapHeight(pos)
	return height == 0
}

func family(pos uint64) (uint64, uint64) {
	peakMap, height := peakMapHeight(pos)
	peak := uint64(1) << height

	if (peakMap & peak) != 0 {
		// Right sibling case
		parent := pos + 1
		sibling := pos + 1 - 2*peak
		return parent, sibling
	} else {
		// Left sibling case
		parent := pos + 2*peak
		sibling := pos + 2*peak - 1
		return parent, sibling
	}
}

func isLeftSibling(pos uint64) bool {
	peakMap, height := peakMapHeight(pos)
	peak := uint64(1) << height
	return (peakMap & peak) == 0
}

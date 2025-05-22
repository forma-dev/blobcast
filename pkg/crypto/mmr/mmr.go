package mmr

import (
	"encoding/binary"
	"fmt"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

type Snapshot []byte

type MMR struct {
	peaks     []crypto.Hash
	numLeaves uint64
}

func NewMMR() *MMR {
	return &MMR{
		peaks:     []crypto.Hash{},
		numLeaves: 0,
	}
}

func NewMMRFromSnapshot(snapshot Snapshot) (*MMR, error) {
	mmr := NewMMR()
	if err := mmr.Restore(snapshot); err != nil {
		return nil, err
	}
	return mmr, nil
}

func (mmr *MMR) NumLeaves() uint64 {
	return mmr.numLeaves
}

func (mmr *MMR) NumPeaks() uint64 {
	return uint64(len(mmr.peaks))
}

func (mmr *MMR) AddLeaf(leaf []byte) crypto.Hash {
	carry := crypto.HashBytes(leaf)
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

		carry = crypto.HashBytes(mmr.peaks[height].Bytes(), carry.Bytes())
		mmr.peaks[height] = crypto.Hash{}
		height++
	}

	mmr.numLeaves++
	return mmr.Root()
}

func (mmr *MMR) Root() crypto.Hash {
	var accum crypto.Hash
	for _, peak := range mmr.peaks {
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

func (mmr *MMR) Snapshot() Snapshot {
	numPeaks := len(mmr.peaks)
	size := 8 + 8 + (numPeaks * 32) // numLeaves + numPeaks + peaks
	buf := make([]byte, size)

	binary.BigEndian.PutUint64(buf[0:8], mmr.numLeaves)
	binary.BigEndian.PutUint64(buf[8:16], uint64(numPeaks))

	offset := 16
	for _, peak := range mmr.peaks {
		copy(buf[offset:offset+32], peak[:])
		offset += 32
	}

	return buf
}

func (mmr *MMR) Restore(snapshot Snapshot) error {
	if len(snapshot) < 16 {
		return fmt.Errorf("invalid snapshot: expected at least 16 bytes, got %d", len(snapshot))
	}

	numLeaves := binary.BigEndian.Uint64(snapshot[0:8])
	numPeaks := binary.BigEndian.Uint64(snapshot[8:16])

	expectedSize := 16 + (int(numPeaks) * 32)
	if len(snapshot) != expectedSize {
		return fmt.Errorf("invalid snapshot: expected %d bytes, got %d", expectedSize, len(snapshot))
	}

	mmr.numLeaves = numLeaves
	mmr.peaks = make([]crypto.Hash, numPeaks)

	offset := 16
	for i := range mmr.peaks {
		copy(mmr.peaks[i][:], snapshot[offset:offset+32])
		offset += 32
	}

	return nil
}

package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

type Hash [32]byte

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) Bytes() []byte {
	return h[:]
}

func (h Hash) Equal(other Hash) bool {
	return bytes.Equal(h[:], other[:])
}

func (h Hash) EqualBytes(other []byte) bool {
	return bytes.Equal(h[:], other)
}

func (h Hash) IsZero() bool {
	return h == Hash{}
}

func HashBytes(data ...[]byte) Hash {
	if len(data) == 1 {
		return sha256.Sum256(data[0])
	}

	var combined []byte
	for _, d := range data {
		combined = append(combined, d...)
	}
	return sha256.Sum256(combined)
}

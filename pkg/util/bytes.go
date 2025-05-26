package util

import (
	"encoding/binary"
	"time"
)

func BytesFromUint32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func BytesFromUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

func BytesFromTime(t time.Time) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(t.Unix()))
	return b
}

func BytesFromShortString(s string) []byte {
	if len(s) > 255 {
		s = s[:255]
	}

	b := make([]byte, 1+len(s))
	b[0] = uint8(len(s))
	copy(b[1:], []byte(s))
	return b
}

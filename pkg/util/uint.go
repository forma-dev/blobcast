package util

import "encoding/binary"

func Uint32FromBytes(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func Uint64FromBytes(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

func Uint64FromBytesKey(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

package fileutil

import (
	"github.com/gabriel-vasile/mimetype"
)

func DetectMimeType(buf []byte) (string, error) {
	return mimetype.Detect(buf).String(), nil
}

// Split into chunks of a specified size
func Split(buf []byte, lim int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf)
	}
	return chunks
}

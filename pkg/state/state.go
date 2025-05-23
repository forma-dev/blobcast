package state

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

var activeNetwork = "mocha"
var dataDir = ""

func getStateDbPath(stateDb string) (string, error) {
	stateDir := filepath.Join(dataDir, activeNetwork)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return "", fmt.Errorf("error creating state directory: %w", err)
	}

	return filepath.Join(stateDir, stateDb), nil
}

func prefixKey(key []byte, prefix []byte) []byte {
	prefixedKey := make([]byte, len(prefix)+len(key))
	copy(prefixedKey, prefix)
	copy(prefixedKey[len(prefix):], key)
	return prefixedKey
}

func prefixHeightKey(height uint64, prefix []byte) []byte {
	k := make([]byte, len(prefix)+8)
	copy(k, prefix)
	binary.BigEndian.PutUint64(k[len(prefix):], height)
	return k
}

func SetNetwork(network string) {
	activeNetwork = network
}

func GetNetwork() string {
	return activeNetwork
}

func SetDataDir(dir string) {
	dataDir = dir
}

func GetDataDir() string {
	return dataDir
}

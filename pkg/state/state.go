package state

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/forma-dev/blobcast/pkg/util"
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
	k := make([]byte, len(prefix)+len(key))
	copy(k, prefix)
	copy(k[len(prefix):], key)
	return k
}

func prefixHeightKey(height uint64, prefix []byte) []byte {
	return prefixKey(util.BytesFromUint64Key(height), prefix)
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

package state

import (
	"fmt"
	"os"
	"path/filepath"
)

var activeNetwork = "mocha"

func getStateDbPath(stateDb string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting user home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, DefaultHomeDir, activeNetwork)
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

func SetNetwork(network string) {
	activeNetwork = network
}

func GetNetwork() string {
	return activeNetwork
}

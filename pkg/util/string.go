package util

import (
	"fmt"
	"strconv"
	"strings"
)

func ShortStringFromBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return string(b[1:])
}

// parseSizeString converts a size string (e.g., "1.5MB", "1024KB") to bytes.
func ParseSizeString(sizeStr string) (int, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.ToUpper(sizeStr)

	var multiplier int
	var suffix string

	if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		suffix = "GB"
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		suffix = "MB"
	} else if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		suffix = "KB"
	} else if strings.HasSuffix(sizeStr, "G") {
		multiplier = 1024 * 1024 * 1024
		suffix = "G"
	} else if strings.HasSuffix(sizeStr, "M") {
		multiplier = 1024 * 1024
		suffix = "M"
	} else if strings.HasSuffix(sizeStr, "K") {
		multiplier = 1024
		suffix = "K"
	} else {
		// Assume bytes if no suffix
		val, err := strconv.Atoi(sizeStr)
		if err != nil {
			return 0, fmt.Errorf("invalid size string format: %s. Expected number or number with K/KB/M/MB/G/GB suffix", sizeStr)
		}
		return val, nil
	}

	// Remove suffix
	numStr := strings.TrimSuffix(sizeStr, suffix)

	// Parse the number part
	valFloat, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in size string: %s", numStr)
	}

	if valFloat < 0 {
		return 0, fmt.Errorf("size cannot be negative: %s", numStr)
	}

	return int(valFloat * float64(multiplier)), nil
}

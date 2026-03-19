package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration parses a string like "1h", "7d", "30m" into a time.Duration.
func ParseDuration(durationStr string) (time.Duration, error) {
	if len(durationStr) == 0 {
		return 0, fmt.Errorf("empty duration string")
	}

	unit := durationStr[len(durationStr)-1:]
	valueStr := durationStr[:len(durationStr)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value")
	}

	switch strings.ToLower(unit) {
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %s. Use m, h, d, w.", unit)
	}
}

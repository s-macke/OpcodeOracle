package tools

import (
	"fmt"
	"strconv"
	"strings"
)

// parseAddress parses an address string in various formats:
// $C000, 0xC000, C000, 49152
func parseAddress(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty address")
	}

	// Handle $ prefix (6502 convention)
	if strings.HasPrefix(s, "$") {
		s = s[1:]
		val, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid hex address: %s", s)
		}
		return uint16(val), nil
	}

	// Handle 0x prefix
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err := strconv.ParseUint(s[2:], 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid hex address: %s", s)
		}
		return uint16(val), nil
	}

	// Try hex first (if contains a-f/A-F)
	if containsHexLetter(s) {
		val, err := strconv.ParseUint(s, 16, 16)
		if err == nil {
			return uint16(val), nil
		}
	}

	// Try decimal
	val, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		// Last attempt: try as hex without prefix
		val, err = strconv.ParseUint(s, 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid address: %s", s)
		}
	}
	return uint16(val), nil
}

// containsHexLetter returns true if s contains any hex letter (a-f or A-F).
func containsHexLetter(s string) bool {
	for _, c := range s {
		if (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			return true
		}
	}
	return false
}

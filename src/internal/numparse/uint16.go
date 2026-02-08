package numparse

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseUint16 parses decimal values and explicit hex ($NNNN or 0xNNNN).
// Unprefixed values are treated as decimal only.
func ParseUint16(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty value")
	}

	raw := s
	base := 10
	if strings.HasPrefix(s, "$") {
		s = s[1:]
		base = 16
	} else if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
		base = 16
	}

	val, err := strconv.ParseUint(s, base, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", raw)
	}
	return uint16(val), nil
}

// ParseHexUint16 parses a hex uint16 from either 0xNNNN or bare hex NNNN.
func ParseHexUint16(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("invalid hex address %q", s)
	}

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}

	val, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid hex address %q: %w", s, err)
	}
	return uint16(val), nil
}

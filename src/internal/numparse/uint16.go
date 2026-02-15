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

// ParseUint16List parses a comma-separated list of uint16 values.
// Each value may be decimal or explicit hex ($NNNN or 0xNNNN).
// Duplicate values are removed while preserving first-seen order.
func ParseUint16List(s string) ([]uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty value")
	}

	parts := strings.Split(s, ",")
	values := make([]uint16, 0, len(parts))
	seen := make(map[uint16]struct{}, len(parts))

	for i, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return nil, fmt.Errorf("empty value at position %d", i+1)
		}

		val, err := ParseUint16(token)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q: %w", token, err)
		}

		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		values = append(values, val)
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("empty value")
	}
	return values, nil
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

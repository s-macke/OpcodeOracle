package main

import (
	"errors"
	"strconv"
	"strings"
)

// parseNumber parses a number in decimal, $hex, or 0xhex format
func parseNumber(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty value")
	}

	var val uint64
	var err error

	if strings.HasPrefix(s, "$") {
		// $hex format
		val, err = strconv.ParseUint(s[1:], 16, 16)
	} else if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// 0xhex format
		val, err = strconv.ParseUint(s[2:], 16, 16)
	} else {
		// Decimal format
		val, err = strconv.ParseUint(s, 10, 16)
	}

	if err != nil {
		return 0, errors.New("invalid number: " + s)
	}

	return uint16(val), nil
}

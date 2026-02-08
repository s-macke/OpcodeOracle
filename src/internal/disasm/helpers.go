package disasm

import (
	"fmt"
	"strings"
)

// formatCodeLeftColumn returns the standardized left column for code output.
// Unlabeled code lines omit addresses to match existing assembly style.
func formatCodeLeftColumn(addr uint16, label string) string {
	if label != "" {
		return fmt.Sprintf("$%04X %-*s", addr, labelFieldWidth, label+":")
	}
	return strings.Repeat(" ", leftColumnWidth)
}

// formatAddressColumn returns the left column with an address but no label.
func formatAddressColumn(addr uint16) string {
	return fmt.Sprintf("$%04X%s", addr, strings.Repeat(" ", leftColumnWidth-addressWidth))
}

// formatAddressOrLabelColumn returns address+label when present, otherwise address only.
func formatAddressOrLabelColumn(addr uint16, label string) string {
	if label != "" {
		return fmt.Sprintf("$%04X %-*s", addr, labelFieldWidth, label+":")
	}
	return formatAddressColumn(addr)
}

// padToColumn pads a string with spaces to reach the specified column.
func padToColumn(s string, col int) string {
	if len(s) >= col {
		return s + "  " // Always at least 2 spaces
	}
	return s + strings.Repeat(" ", col-len(s))
}

// toASCII converts bytes to printable ASCII, using '.' for non-printable.
func toASCII(bytes []byte) string {
	result := make([]byte, len(bytes))
	for i, b := range bytes {
		if b >= 0x20 && b <= 0x7E {
			result[i] = b
		} else {
			result[i] = '.'
		}
	}
	return string(result)
}

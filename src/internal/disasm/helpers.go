package disasm

import "strings"

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

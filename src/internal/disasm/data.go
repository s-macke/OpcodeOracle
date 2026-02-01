package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"
)

// formatDataAt formats data bytes at the given address.
// Returns the formatted output and number of bytes consumed.
func (d *disassembler) formatDataAt(addr, end uint16, needsBlankLine *bool) (string, int) {
	var sb strings.Builder

	// Output headline annotations
	headlines := d.getHeadlineAnnotations(addr, addr+1)
	if len(headlines) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatHeadlines(headlines))
		*needsBlankLine = false
	}

	// Check for symbol at this address
	syms := d.state.Symbols.At(addr)
	label := d.getLabelFromSymbols(syms)
	symType := d.getDataTypeFromSymbols(syms)

	// Get inline annotations
	inlines := d.getInlineAnnotations(addr, addr+1)

	// Check if this is a labeled data item
	if label != "" && (symType == symbols.SymbolWord || symType == symbols.SymbolByte) {
		if *needsBlankLine {
			sb.WriteString("\n")
		}

		if symType == symbols.SymbolWord {
			// Format as .WORD
			word, err := d.state.Binary.ReadWord(addr)
			if err != nil {
				// Fall back to single byte if can't read word
				b, _ := d.state.Binary.ReadByte(addr)
				line := fmt.Sprintf("$%04X %-18s.BYTE $%02X", addr, label+":", b)
				if len(inlines) > 0 {
					line = padToColumn(line, 38) + "; " + inlines[0].Comment
				}
				sb.WriteString(line + "\n")
				*needsBlankLine = true
				return sb.String(), 1
			}
			line := fmt.Sprintf("$%04X %-18s.WORD $%04X", addr, label+":", word)
			if len(inlines) > 0 {
				line = padToColumn(line, 38) + "; " + inlines[0].Comment
			}
			sb.WriteString(line + "\n")
			*needsBlankLine = true
			return sb.String(), 2
		}

		// Format as labeled .BYTE
		b, _ := d.state.Binary.ReadByte(addr)
		line := fmt.Sprintf("$%04X %-18s.BYTE $%02X", addr, label+":", b)
		if len(inlines) > 0 {
			line = padToColumn(line, 38) + "; " + inlines[0].Comment
		}
		sb.WriteString(line + "\n")
		*needsBlankLine = true
		return sb.String(), 1
	}

	// Determine chunk size (up to 16 bytes, break at boundaries)
	chunkSize := d.calculateDataChunkSize(addr, end)

	// For unlabeled data, treat inline annotations as headlines
	if len(inlines) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatHeadlines(inlines))
		*needsBlankLine = false
	}

	// Read bytes
	bytes := make([]byte, chunkSize)
	for i := 0; i < chunkSize; i++ {
		b, _ := d.state.Binary.ReadByte(addr + uint16(i))
		bytes[i] = b
	}

	// Format as standard .BYTE row
	hexParts := make([]string, chunkSize)
	for i, b := range bytes {
		hexParts[i] = fmt.Sprintf("$%02X", b)
	}
	hexStr := strings.Join(hexParts, ",")

	// Build ASCII representation
	ascii := toASCII(bytes)

	// Format: $XXXX padded to 24 chars, then .BYTE hex  ; "ASCII"
	// Column 95 = 24 (label) + 6 (.BYTE ) + 63 (max 16 bytes hex) + 2 (spacing)
	line := fmt.Sprintf("$%04X                   .BYTE %s", addr, hexStr)
	line = padToColumn(line, 95)
	line += fmt.Sprintf("; \"%s\"", ascii)

	sb.WriteString(line + "\n")
	*needsBlankLine = true
	return sb.String(), chunkSize
}

// calculateDataChunkSize determines how many bytes to include in a data row.
func (d *disassembler) calculateDataChunkSize(addr, end uint16) int {
	// Calculate bytes to next 16-byte boundary
	// When already aligned (addr & 0x0F == 0), this gives 16
	bytesToBoundary := 16 - int(addr&0x0F)

	maxBytes := bytesToBoundary

	// Limit to remaining range
	remaining := int(end - addr)
	if remaining < maxBytes {
		maxBytes = remaining
	}

	// Check for region boundaries, symbols, or annotations that would break the row
	for i := 1; i < maxBytes; i++ {
		nextAddr := addr + uint16(i)

		// Check for region boundary (code starts)
		if d.state.Regions.At(nextAddr) == regions.RegionCode {
			return i
		}

		// Check for symbol
		if len(d.state.Symbols.At(nextAddr)) > 0 {
			return i
		}

		// Check for any annotation (breaks chunk)
		if len(d.state.Annotations.At(nextAddr)) > 0 {
			return i
		}
	}

	return maxBytes
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

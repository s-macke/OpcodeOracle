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
	hdls := d.getHeadlines(addr, addr+1)
	if len(hdls) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatHeadlines(hdls))
		*needsBlankLine = false
	}

	// Check for symbol at this address
	label := d.getLabel(addr)
	symType := d.getDataType(addr)

	// Get inline annotations
	inlines := d.getInlineAnnotations(addr, addr+1)

	// Check if this is a labeled byte data item (SymbolWord is expanded to _LO/_HI bytes at creation time)
	if label != "" && symType == symbols.SymbolByte {
		// No blank line before labeled bytes - they flow naturally from preceding data
		// Output xref comments before labeled data
		for _, xref := range d.formatXRefs(addr) {
			sb.WriteString(xref + "\n")
		}
		// Format as labeled .BYTE
		b, _ := d.state.Binary.ReadByte(addr)
		line := fmt.Sprintf("$%04X %-18s.BYTE $%02X", addr, label+":", b)
		if len(inlines) > 0 {
			line = padToColumn(line, 38) + "; " + inlines[0].Comment
		}
		sb.WriteString(line + "\n")
		*needsBlankLine = false
		return sb.String(), 1
	}

	// Determine chunk size (up to 16 bytes, break at boundaries)
	chunkSize := d.calculateDataChunkSize(addr, end)

	// For unlabeled data, treat inline annotations as headlines
	if len(inlines) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatInlinesAsHeadlines(inlines))
		*needsBlankLine = false
	}

	// Output xref comments before data line
	for _, xref := range d.formatXRefs(addr) {
		sb.WriteString(xref + "\n")
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

// formatInlinesAsHeadlines formats inline annotations as a headline block (for data).
func (d *disassembler) formatInlinesAsHeadlines(inlines []inlineAnnotation) string {
	var sb strings.Builder
	sb.WriteString("; --------------------------------------------------------\n")
	for _, i := range inlines {
		sb.WriteString("; " + i.Comment + "\n")
	}
	sb.WriteString("; --------------------------------------------------------\n")
	return sb.String()
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

	// Check for region boundaries, symbols, headlines, or annotations that would break the row
	for i := 1; i < maxBytes; i++ {
		nextAddr := addr + uint16(i)

		// Check for region boundary (code starts)
		if d.state.Regions.At(nextAddr) == regions.RegionCode {
			return i
		}

		// Check for symbol
		if _, ok := d.state.Symbols.At(nextAddr); ok {
			return i
		}

		// Check for headline
		if d.state.Headlines != nil && len(d.state.Headlines.At(nextAddr)) > 0 {
			return i
		}

		// Check for inline annotation (breaks chunk)
		if len(d.state.Annotations.At(nextAddr)) > 0 {
			return i
		}

		// Check for xref (break so xref appears on its own line)
		if d.state.XRefs != nil && len(d.state.XRefs.To(nextAddr)) > 0 {
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

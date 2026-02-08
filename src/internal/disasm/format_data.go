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
	label := ""
	symType := symbols.SymbolType("")
	if sym, ok := d.getSymbol(addr); ok {
		label = sym.Name
		symType = sym.Type
	}

	// Get inline annotations
	inlines := d.getInlineAnnotations(addr, addr+1)
	inlineLines := splitInlineComments(inlines)

	// Check if this is a labeled byte data item (SymbolWord is expanded to _LO/_HI bytes at creation time)
	if label != "" && symType == symbols.SymbolByte {
		sb.WriteString(d.formatLabeledByte(addr, label, inlineLines))
		*needsBlankLine = false
		return sb.String(), 1
	}

	// Determine chunk size (up to one data row, break at boundaries)
	chunkSize := d.calculateDataChunkSize(addr, end)

	// For unlabeled data, treat inline annotations as headlines
	if len(inlines) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatInlinesAsHeadlines(inlines))
		*needsBlankLine = false
	}

	sb.WriteString(d.formatDataRow(addr, chunkSize))
	*needsBlankLine = true
	return sb.String(), chunkSize
}

func (d *disassembler) formatLabeledByte(addr uint16, label string, inlineLines []string) string {
	var sb strings.Builder

	// No blank line before labeled bytes - they flow naturally from preceding data
	// Output xref comments before labeled data
	for _, xref := range d.formatXRefs(addr) {
		sb.WriteString("; " + xref + "\n")
	}
	// Format as labeled .BYTE
	b, _ := d.state.Binary.ReadByte(addr)
	line := formatAddressOrLabelColumn(addr, label) + fmt.Sprintf(".BYTE $%02X", b)
	if len(inlineLines) > 0 {
		line = padToColumn(line, instructionCommentCol) + "; " + inlineLines[0]
		sb.WriteString(line + "\n")
		for _, comment := range inlineLines[1:] {
			sb.WriteString(padToColumn("", instructionCommentCol) + "; " + comment + "\n")
		}
		return sb.String()
	}
	sb.WriteString(line + "\n")
	return sb.String()
}

func (d *disassembler) formatDataRow(addr uint16, chunkSize int) string {
	var sb strings.Builder

	// Output xref comments before data line
	for _, xref := range d.formatXRefs(addr) {
		sb.WriteString("; " + xref + "\n")
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

	// Format: left column, then .BYTE hex, then ASCII comment.
	line := formatAddressColumn(addr) + fmt.Sprintf(".BYTE %s", hexStr)
	line = padToColumn(line, dataAsciiCol)
	line += fmt.Sprintf("; \"%s\"", ascii)

	sb.WriteString(line + "\n")
	return sb.String()
}

// calculateDataChunkSize determines how many bytes to include in a data row.
func (d *disassembler) calculateDataChunkSize(addr, end uint16) int {
	// Calculate bytes to next data-row boundary.
	// When already aligned, this yields dataRowMaxBytes.
	bytesToBoundary := dataRowMaxBytes - int(addr%dataRowMaxBytes)

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

// Package disasm provides a MOS6502 disassembler.
package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/asm"
	"opcodeoracle/internal/binary"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

// Disassembler formats MOS6502 machine code as assembly text.
type Disassembler interface {
	// Disassemble formats the address range [start, end) as assembly text.
	Disassemble(start, end uint16) (string, error)
}

// disassembler implements the Disassembler interface.
type disassembler struct {
	state *state.State
}

// NewDisassembler creates a disassembler that reads from the given state.
func NewDisassembler(s *state.State) Disassembler {
	return &disassembler{state: s}
}

// Disassemble formats the address range [start, end) as assembly text.
func (d *disassembler) Disassemble(start, end uint16) (string, error) {
	// Validate addresses against binary bounds
	if _, err := d.state.Binary.ReadByte(start); err != nil {
		return "", binary.ErrAddressOutOfRange
	}
	if end > start {
		if _, err := d.state.Binary.ReadByte(end - 1); err != nil {
			return "", binary.ErrAddressOutOfRange
		}
	}

	var sb strings.Builder
	addr := start
	needsBlankLine := false

	for addr < end {
		regionType := d.state.Regions.At(addr)

		if regionType == regions.RegionCode {
			output, size, err := d.formatCodeAt(addr, &needsBlankLine)
			if err != nil {
				return sb.String(), err
			}
			sb.WriteString(output)
			addr += uint16(size)
		} else {
			output, size := d.formatDataAt(addr, end, &needsBlankLine)
			sb.WriteString(output)
			addr += uint16(size)
		}
	}

	return sb.String(), nil
}

// formatCodeAt formats a single instruction at the given address.
// Returns the formatted output and instruction size.
func (d *disassembler) formatCodeAt(addr uint16, needsBlankLine *bool) (string, int, error) {
	var sb strings.Builder

	// Output headline annotations
	headlines := d.getHeadlineAnnotations(addr)
	if len(headlines) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatHeadlines(headlines))
		*needsBlankLine = false
	}

	// Read opcode
	opcode, err := d.state.Binary.ReadByte(addr)
	if err != nil {
		return "", 0, err
	}

	def := asm.Opcodes[opcode]
	if def.IsIllegal() {
		return "", 0, &IllegalOpcodeError{Address: addr, Opcode: opcode}
	}

	// Read operand bytes
	operand := make([]byte, def.OperandSize())
	for i := 0; i < def.OperandSize(); i++ {
		b, err := d.state.Binary.ReadByte(addr + uint16(i+1))
		if err != nil {
			return "", 0, err
		}
		operand[i] = b
	}

	// Format label column (12 chars)
	label := d.getLabel(addr)
	labelStr := ""
	if label != "" {
		labelStr = label + ":"
	}

	// Format mnemonic
	mnemonic := def.Op.String()

	// Format operand with label resolution for branches
	operandStr := d.formatOperandWithLabel(def, operand, addr)

	// Get inline annotations
	inlines := d.getInlineAnnotations(addr)

	// Build the instruction line
	// Label column: 12 chars, mnemonic: 3 chars + space, operand: variable
	line := fmt.Sprintf("%-12s%s%s", labelStr, mnemonic, operandStr)

	if len(inlines) > 0 {
		// Pad to column for comment (minimum 2 spaces before semicolon)
		line = padToColumn(line, 26)
		line += "; " + inlines[0].Comment
		sb.WriteString(line + "\n")

		// Additional inline comments on continuation lines
		for i := 1; i < len(inlines); i++ {
			contLine := padToColumn("", 26) + "; " + inlines[i].Comment
			sb.WriteString(contLine + "\n")
		}
	} else {
		sb.WriteString(line + "\n")
	}

	*needsBlankLine = true
	return sb.String(), def.Size, nil
}

// formatDataAt formats data bytes at the given address.
// Returns the formatted output and number of bytes consumed.
func (d *disassembler) formatDataAt(addr, end uint16, needsBlankLine *bool) (string, int) {
	var sb strings.Builder

	// Output headline annotations
	headlines := d.getHeadlineAnnotations(addr)
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
	inlines := d.getInlineAnnotations(addr)

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
				line := fmt.Sprintf("%s: .BYTE $%02X", label, b)
				if len(inlines) > 0 {
					line = padToColumn(line, 78) + "; " + inlines[0].Comment
				}
				sb.WriteString(line + "\n")
				*needsBlankLine = true
				return sb.String(), 1
			}
			line := fmt.Sprintf("%s: .WORD $%04X", label, word)
			if len(inlines) > 0 {
				line = padToColumn(line, 78) + "; " + inlines[0].Comment
			}
			sb.WriteString(line + "\n")
			*needsBlankLine = true
			return sb.String(), 2
		}

		// Format as labeled .BYTE
		b, _ := d.state.Binary.ReadByte(addr)
		line := fmt.Sprintf("%s: .BYTE $%02X", label, b)
		if len(inlines) > 0 {
			line = padToColumn(line, 78) + "; " + inlines[0].Comment
		}
		sb.WriteString(line + "\n")
		*needsBlankLine = true
		return sb.String(), 1
	}

	// Determine chunk size (up to 16 bytes, break at boundaries)
	chunkSize := d.calculateDataChunkSize(addr, end)

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

	// Format: $XXXX    .BYTE hex  ; "ASCII"
	line := fmt.Sprintf("$%04X    .BYTE %s", addr, hexStr)
	line = padToColumn(line, 78)
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

		// Check for headline annotation
		for _, ann := range d.state.Annotations.At(nextAddr) {
			if ann.Type == annotations.AnnotationHeadline {
				return i
			}
		}
	}

	return maxBytes
}

// getLabel returns the label name for an address, or empty string if none.
func (d *disassembler) getLabel(addr uint16) string {
	syms := d.state.Symbols.At(addr)
	return d.getLabelFromSymbols(syms)
}

// getLabelFromSymbols extracts a label name from a symbol slice.
func (d *disassembler) getLabelFromSymbols(syms []symbols.Symbol) string {
	for _, sym := range syms {
		// Prefer user-defined symbols, but accept any
		if sym.Source == symbols.SourceUser {
			return sym.Name
		}
	}
	// Return first available
	if len(syms) > 0 {
		return syms[0].Name
	}
	return ""
}

// getDataTypeFromSymbols extracts the data type from a symbol slice.
func (d *disassembler) getDataTypeFromSymbols(syms []symbols.Symbol) symbols.SymbolType {
	for _, sym := range syms {
		if sym.Type == symbols.SymbolWord || sym.Type == symbols.SymbolByte {
			return sym.Type
		}
	}
	return ""
}

// formatOperandWithLabel formats the operand, resolving branch targets to labels.
func (d *disassembler) formatOperandWithLabel(def asm.OpcodeDef, operand []byte, pc uint16) string {
	if def.Mode == asm.AddrRelative && len(operand) >= 1 {
		// Calculate branch target
		target := calculateBranchTarget(pc, operand[0])

		// Look up symbol at target
		label := d.getLabel(target)
		if label != "" {
			return " " + label
		}
		// Generate auto-label format
		return fmt.Sprintf(" L_%04X", target)
	}

	// JSR and JMP with absolute addressing - resolve to label if available
	if def.Mode == asm.AddrAbsolute && (def.Op == asm.JSR || def.Op == asm.JMP) && len(operand) >= 2 {
		target := uint16(operand[0]) | uint16(operand[1])<<8
		label := d.getLabel(target)
		if label != "" {
			return " " + label
		}
	}

	return def.FormatOperand(operand)
}

// calculateBranchTarget computes the target address for a relative branch.
func calculateBranchTarget(pc uint16, operand byte) uint16 {
	// Branch is relative to the address AFTER the branch instruction (PC + 2)
	nextPC := pc + 2
	if operand > 0x7F {
		// Negative offset (two's complement)
		return nextPC - uint16(256-int(operand))
	}
	return nextPC + uint16(operand)
}

// getHeadlineAnnotations returns headline annotations at the given address.
func (d *disassembler) getHeadlineAnnotations(addr uint16) []annotations.Annotation {
	var headlines []annotations.Annotation
	for _, ann := range d.state.Annotations.At(addr) {
		if ann.Type == annotations.AnnotationHeadline {
			headlines = append(headlines, ann)
		}
	}
	return headlines
}

// getInlineAnnotations returns inline annotations at the given address.
func (d *disassembler) getInlineAnnotations(addr uint16) []annotations.Annotation {
	var inlines []annotations.Annotation
	for _, ann := range d.state.Annotations.At(addr) {
		if ann.Type == annotations.AnnotationInline {
			inlines = append(inlines, ann)
		}
	}
	return inlines
}

// formatHeadlines formats headline annotations as a block comment.
func (d *disassembler) formatHeadlines(headlines []annotations.Annotation) string {
	var sb strings.Builder
	sb.WriteString("; --------------------------------------------------------\n")
	for _, h := range headlines {
		sb.WriteString("; " + h.Comment + "\n")
	}
	sb.WriteString("; --------------------------------------------------------\n")
	return sb.String()
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

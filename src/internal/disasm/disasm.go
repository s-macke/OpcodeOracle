// Package disasm provides a MOS6502 disassembler.
package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/asm"
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
	state      *state.State
	boundaries analysis.InstructionBoundaries // optional, may be nil
}

// NewDisassembler creates a disassembler that reads from the given state.
// The boundaries parameter is optional and provides instruction boundary information
// from flow analysis. When provided, the disassembler will error if asked to
// disassemble at an operand byte address.
func NewDisassembler(s *state.State, boundaries analysis.InstructionBoundaries) Disassembler {
	return &disassembler{state: s, boundaries: boundaries}
}

// Disassemble formats the address range [start, end) as assembly text.
func (d *disassembler) Disassemble(start, end uint16) (string, error) {
	// Calculate binary bounds
	origin := d.state.Binary.Start()
	binaryEnd := d.state.Binary.End()

	// Validate start address
	if start < origin || start > binaryEnd {
		return "", &AddressOutOfRangeError{
			Address:    start,
			RangeStart: origin,
			RangeEnd:   binaryEnd,
		}
	}

	// Validate end address (end is exclusive, so check end-1)
	if end > start && (end-1 < origin || end-1 > binaryEnd) {
		return "", &AddressOutOfRangeError{
			Address:    end - 1,
			RangeStart: origin,
			RangeEnd:   binaryEnd,
		}
	}

	// Validate range order
	if start > end {
		return "", &InvalidRangeError{Start: start, End: end}
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
	// Check if address is mid-instruction (operand byte)
	if d.boundaries != nil && d.boundaries.IsInstructionDataAt(addr) {
		return "", 0, &MidInstructionError{Address: addr}
	}

	var sb strings.Builder

	// Read opcode first to determine instruction size
	opcode, err := d.state.Binary.ReadByte(addr)
	if err != nil {
		return "", 0, err
	}

	def := asm.Opcodes[opcode]
	if def.IsIllegal() {
		return "", 0, &IllegalOpcodeError{Address: addr, Opcode: opcode}
	}

	// Now we know the size - collect all annotations within this instruction
	instrEnd := addr + uint16(def.Size)

	// Output headline annotations (from all bytes of instruction)
	headlines := d.getHeadlineAnnotations(addr, instrEnd)
	if len(headlines) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatHeadlines(headlines))
		*needsBlankLine = false
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

	// Format label column (24 chars total)
	label := d.getLabel(addr)
	var labelCol string
	if label != "" {
		// With label: $XXXX label: padded to 24 chars (5 + 1 + 18 = 24)
		labelCol = fmt.Sprintf("$%04X %-18s", addr, label+":")
	} else {
		// No label: 24 spaces
		labelCol = "                        "
	}

	// Format mnemonic
	mnemonic := def.Op.String()

	// Format operand with label resolution for branches
	operandStr := d.formatOperandWithLabel(def, operand, addr)

	// Get inline annotations (from all bytes of instruction)
	inlines := d.getInlineAnnotations(addr, instrEnd)

	// Build the instruction line
	line := labelCol + mnemonic + operandStr

	if len(inlines) > 0 {
		// Pad to column for comment (minimum 2 spaces before semicolon)
		line = padToColumn(line, 38)
		line += "; " + inlines[0].Comment
		sb.WriteString(line + "\n")

		// Additional inline comments on continuation lines
		for i := 1; i < len(inlines); i++ {
			contLine := padToColumn("", 38) + "; " + inlines[i].Comment
			sb.WriteString(contLine + "\n")
		}
	} else {
		sb.WriteString(line + "\n")
	}

	*needsBlankLine = true
	return sb.String(), def.Size, nil
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

// getHeadlineAnnotations returns headline annotations in the address range [start, end).
func (d *disassembler) getHeadlineAnnotations(start, end uint16) []annotations.Annotation {
	var headlines []annotations.Annotation
	for addr := start; addr < end; addr++ {
		for _, ann := range d.state.Annotations.At(addr) {
			if ann.Type == annotations.AnnotationHeadline {
				headlines = append(headlines, ann)
			}
		}
	}
	return headlines
}

// getInlineAnnotations returns inline annotations in the address range [start, end).
func (d *disassembler) getInlineAnnotations(start, end uint16) []annotations.Annotation {
	var inlines []annotations.Annotation
	for addr := start; addr < end; addr++ {
		for _, ann := range d.state.Annotations.At(addr) {
			if ann.Type == annotations.AnnotationInline {
				inlines = append(inlines, ann)
			}
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

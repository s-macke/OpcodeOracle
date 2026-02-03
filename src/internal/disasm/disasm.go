// Package disasm provides a MOS6502 disassembler.
package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/asm"
	"opcodeoracle/internal/headlines"
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
		// Output warning comment and byte as data, then continue
		return d.formatMidInstructionAt(addr, needsBlankLine), 1, nil
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

	// Now we know the size - collect all headlines within this instruction
	instrEnd := addr + uint16(def.Size)

	// Output headline annotations (from all bytes of instruction)
	hdls := d.getHeadlines(addr, instrEnd)
	if len(hdls) > 0 {
		if *needsBlankLine {
			sb.WriteString("\n")
		}
		sb.WriteString(d.formatHeadlines(hdls))
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
	xrefComments := d.formatXRefs(addr)
	operandXRefs := d.formatOperandXRefs(addr, def.OperandSize())

	// Get symbol for operand address (if any)
	operandSymbol := d.getOperandSymbol(def, operand, addr)

	if len(inlines) > 0 {
		// Collect all annotation lines (handling multi-line comments)
		var commentLines []string
		for _, ann := range inlines {
			for _, l := range strings.Split(ann.Comment, "\n") {
				commentLines = append(commentLines, l)
			}
		}

		// First line: operand symbol if present, otherwise first annotation
		if operandSymbol != "" {
			line = padToColumn(line, 38) + "; " + operandSymbol
			sb.WriteString(line + "\n")
			// All annotations on continuation lines
			for _, comment := range commentLines {
				sb.WriteString(padToColumn("", 38) + "; " + comment + "\n")
			}
		} else {
			// No symbol - first annotation on instruction line
			line = padToColumn(line, 38) + "; " + commentLines[0]
			sb.WriteString(line + "\n")
			// Remaining annotations on continuation lines
			for i := 1; i < len(commentLines); i++ {
				sb.WriteString(padToColumn("", 38) + "; " + commentLines[i] + "\n")
			}
		}

		// Xrefs on continuation lines after annotations
		for _, xref := range xrefComments {
			sb.WriteString(padToColumn("", 38) + xref + "\n")
		}
	} else if operandSymbol != "" {
		// No annotations - show operand symbol as comment
		line = padToColumn(line, 38) + "; " + operandSymbol
		sb.WriteString(line + "\n")

		// Xrefs on continuation lines
		for _, xref := range xrefComments {
			sb.WriteString(padToColumn("", 38) + xref + "\n")
		}
	} else if len(xrefComments) > 0 {
		// No annotations or symbol - first xref on same line as instruction
		line = padToColumn(line, 38) + xrefComments[0]
		sb.WriteString(line + "\n")

		// Additional xrefs on continuation lines
		for i := 1; i < len(xrefComments); i++ {
			sb.WriteString(padToColumn("", 38) + xrefComments[i] + "\n")
		}
	} else {
		sb.WriteString(line + "\n")
	}

	// Output operand xrefs (self-modifying code references)
	for _, oxref := range operandXRefs {
		sb.WriteString(padToColumn("", 38) + oxref + "\n")
	}

	*needsBlankLine = true
	return sb.String(), def.Size, nil
}

// getLabel returns the label name for an address, or empty string if none.
func (d *disassembler) getLabel(addr uint16) string {
	if sym, ok := d.state.Symbols.At(addr); ok {
		return sym.Name
	}
	return ""
}

// getDataType returns the data type from the symbol at addr, or empty if none.
func (d *disassembler) getDataType(addr uint16) symbols.SymbolType {
	if sym, ok := d.state.Symbols.At(addr); ok {
		if sym.Type == symbols.SymbolByte {
			return sym.Type
		}
	}
	return ""
}

// getOperandAddress extracts the target address from an instruction operand.
// Returns the address and true if the addressing mode uses a 16-bit address, false otherwise.
func getOperandAddress(mode asm.AddrMode, operand []byte) (uint16, bool) {
	if len(operand) < 2 {
		return 0, false
	}
	switch mode {
	case asm.AddrAbsolute, asm.AddrAbsoluteX, asm.AddrAbsoluteY, asm.AddrIndirect:
		return uint16(operand[0]) | uint16(operand[1])<<8, true
	}
	return 0, false
}

// formatOperandWithLabel formats the operand, resolving branch targets to labels.
func (d *disassembler) formatOperandWithLabel(def asm.OpcodeDef, operand []byte, pc uint16) string {
	if def.Mode == asm.AddrRelative && len(operand) >= 1 {
		// Always show numeric target address (symbol will be in comment)
		target := calculateBranchTarget(pc, operand[0])
		return fmt.Sprintf(" $%04X", target)
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

// getOperandSymbol returns the symbol name for the instruction's operand target, if any.
func (d *disassembler) getOperandSymbol(def asm.OpcodeDef, operand []byte, pc uint16) string {
	// Relative branches
	if def.Mode == asm.AddrRelative && len(operand) >= 1 {
		target := calculateBranchTarget(pc, operand[0])
		return d.getLabel(target)
	}
	// Absolute modes (skip JSR/JMP which substitute label in operand)
	if def.Mode == asm.AddrAbsolute && (def.Op == asm.JSR || def.Op == asm.JMP) {
		return ""
	}
	if opAddr, ok := getOperandAddress(def.Mode, operand); ok {
		return d.getLabel(opAddr)
	}
	return ""
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

// getHeadlines returns headlines in the address range [start, end).
func (d *disassembler) getHeadlines(start, end uint16) []headlines.Headline {
	var result []headlines.Headline
	if d.state.Headlines == nil {
		return result
	}
	for addr := start; addr < end; addr++ {
		result = append(result, d.state.Headlines.At(addr)...)
	}
	return result
}

// inlineAnnotation is a simple struct to hold inline annotation data for disasm output.
type inlineAnnotation struct {
	Comment string
}

// getInlineAnnotations returns inline annotations in the address range [start, end).
func (d *disassembler) getInlineAnnotations(start, end uint16) []inlineAnnotation {
	var inlines []inlineAnnotation
	for addr := start; addr < end; addr++ {
		for _, ann := range d.state.Annotations.At(addr) {
			inlines = append(inlines, inlineAnnotation{Comment: ann.Comment})
		}
	}
	return inlines
}

// formatXRefs formats cross-references pointing to the given address.
// Returns a slice of formatted xref comment lines.
func (d *disassembler) formatXRefs(addr uint16) []string {
	if d.state.XRefs == nil {
		return nil
	}

	refs := d.state.XRefs.To(addr)
	if len(refs) == 0 {
		return nil
	}

	var result []string
	for _, ref := range refs {
		line := fmt.Sprintf("; xref: %s from $%04X", ref.Type, ref.From)
		if label := d.getLabel(ref.From); label != "" {
			line += " (" + label + ")"
		}
		result = append(result, line)
	}

	return result
}

// formatOperandXRefs formats cross-references pointing to operand bytes within an instruction.
// Returns formatted strings for each operand byte offset (1-indexed) that has xrefs.
func (d *disassembler) formatOperandXRefs(addr uint16, operandSize int) []string {
	if d.state.XRefs == nil || operandSize == 0 {
		return nil
	}

	var result []string
	for offset := 1; offset <= operandSize; offset++ {
		refs := d.state.XRefs.To(addr + uint16(offset))
		if len(refs) == 0 {
			continue
		}

		var parts []string
		for _, ref := range refs {
			part := fmt.Sprintf("%s from $%04X", ref.Type, ref.From)
			if label := d.getLabel(ref.From); label != "" {
				part += " (" + label + ")"
			}
			parts = append(parts, part)
		}

		result = append(result, fmt.Sprintf("; xref+%d: %s  [instruction data modified]", offset, strings.Join(parts, ", ")))
	}

	return result
}

// formatHeadlines formats headline annotations as a block comment.
func (d *disassembler) formatHeadlines(hdls []headlines.Headline) string {
	var sb strings.Builder
	sb.WriteString("; --------------------------------------------------------\n")
	for _, h := range hdls {
		for _, line := range strings.Split(h.Comment, "\n") {
			sb.WriteString("; " + line + "\n")
		}
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

// formatMidInstructionAt handles a mid-instruction address by outputting a warning
// comment and the byte as data. Returns formatted output.
func (d *disassembler) formatMidInstructionAt(addr uint16, needsBlankLine *bool) string {
	var sb strings.Builder

	if *needsBlankLine {
		sb.WriteString("\n")
	}

	// Output warning comment
	sb.WriteString(fmt.Sprintf("; WARNING: mid-instruction byte at $%04X\n", addr))

	// Read and output the byte as data
	b, _ := d.state.Binary.ReadByte(addr)
	sb.WriteString(fmt.Sprintf("$%04X                   .BYTE $%02X\n", addr, b))

	*needsBlankLine = true
	return sb.String()
}

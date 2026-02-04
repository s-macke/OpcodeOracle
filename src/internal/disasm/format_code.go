package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/asm"
)

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
	label := ""
	if sym, ok := d.getSymbol(addr); ok {
		label = sym.Name
	}
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

	// Format operand and resolve symbols for comments when needed.
	operandStr, operandSymbol := d.formatOperandWithSymbol(def, operand, addr)

	// Get inline annotations (from all bytes of instruction)
	inlines := d.getInlineAnnotations(addr, instrEnd)

	// Build the instruction line
	line := labelCol + mnemonic + operandStr
	xrefComments := d.formatXRefs(addr)
	operandXRefs := d.formatOperandXRefs(addr, def.OperandSize())

	// Collect all annotation lines (handling multi-line comments)
	annotationLines := splitInlineComments(inlines)

	// Decide which comment goes on the instruction line.
	var firstComment string
	var continuation []string
	switch {
	case operandSymbol != "":
		firstComment = operandSymbol
		continuation = append(continuation, annotationLines...)
		continuation = append(continuation, xrefComments...)
	case len(annotationLines) > 0:
		firstComment = annotationLines[0]
		continuation = append(continuation, annotationLines[1:]...)
		continuation = append(continuation, xrefComments...)
	case len(xrefComments) > 0:
		firstComment = xrefComments[0]
		continuation = append(continuation, xrefComments[1:]...)
	}

	writeInstructionWithComments(&sb, line, firstComment, continuation)

	// Output operand xrefs (self-modifying code references)
	for _, oxref := range operandXRefs {
		sb.WriteString(padToColumn("", instructionCommentCol) + oxref + "\n")
	}

	*needsBlankLine = true
	return sb.String(), def.Size, nil
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

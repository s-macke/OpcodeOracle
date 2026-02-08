package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/asm"
)

// formatCodeAt formats a single instruction at the given address.
// Returns the formatted output and instruction size.
func (d *disassembler) formatCodeAt(addr uint16) (string, int, error) {
	// Check if address is mid-instruction (operand byte)
	if d.boundaries != nil && d.boundaries.IsInstructionDataAt(addr) {
		// Output warning comment and byte as data, then continue
		return d.formatMidInstructionAt(addr), 1, nil
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

	// Output headline annotations (from all bytes of instruction).
	d.writeHeadlines(&sb, addr, instrEnd)

	operand, err := d.readOperandBytes(addr, def.OperandSize())
	if err != nil {
		return "", 0, err
	}

	labelCol := strings.Repeat(" ", codeInstrIndent)

	xrefComments := d.formatXRefs(addr)
	if sym, ok := d.getSymbol(addr); ok {
		writeLabeledCodeLine(&sb, addr, sym.Name, xrefComments)
		// Label-related xrefs are attached to the label line.
		xrefComments = nil
	}

	// Format mnemonic
	mnemonic := def.Op.String()

	// Format operand and resolve symbols for comments when needed.
	operandStr, operandSymbol := d.formatOperandWithSymbol(def, operand, addr)

	// Build the instruction line
	line := labelCol + mnemonic + operandStr

	operandXRefs := d.formatOperandXRefs(addr, def.OperandSize())
	// Collect all annotation lines (handling multi-line comments)
	annotationLines := d.getInlineCommentLines(addr, instrEnd)

	comments := chooseInstructionComments(
		operandSymbol,
		annotationLines,
		xrefComments,
		operandXRefs,
	)

	writeInstructionWithComments(&sb, line, comments)

	return sb.String(), def.Size, nil
}

func (d *disassembler) readOperandBytes(addr uint16, operandSize int) ([]byte, error) {
	operand := make([]byte, operandSize)
	for i := 0; i < operandSize; i++ {
		b, err := d.state.Binary.ReadByte(addr + uint16(i+1))
		if err != nil {
			return nil, err
		}
		operand[i] = b
	}
	return operand, nil
}

func writeLabeledCodeLine(sb *strings.Builder, addr uint16, label string, comments []string) {
	labelLine := formatAddressOrLabelColumn(addr, label)
	writeInstructionWithComments(sb, labelLine, comments)
}

func chooseInstructionComments(
	operandSymbol string,
	annotationLines []string,
	xrefComments []string,
	operandXRefs []string,
) []string {
	var comments []string

	if operandSymbol != "" {
		comments = append(comments, operandSymbol)
	}
	comments = append(comments, annotationLines...)
	comments = append(comments, xrefComments...)
	comments = append(comments, operandXRefs...)

	return comments
}

// formatMidInstructionAt handles a mid-instruction address by outputting a warning
// comment and the byte as data. Returns formatted output.
func (d *disassembler) formatMidInstructionAt(addr uint16) string {
	var sb strings.Builder

	sb.WriteString("\n")

	// Output warning comment
	sb.WriteString(fmt.Sprintf("; WARNING: mid-instruction byte at $%04X\n", addr))

	// Read and output the byte as data
	b, _ := d.state.Binary.ReadByte(addr)
	sb.WriteString(formatAddressColumn(addr) + fmt.Sprintf(".BYTE $%02X\n", b))

	return sb.String()
}

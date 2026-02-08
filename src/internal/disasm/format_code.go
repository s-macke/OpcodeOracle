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

	// Resolve optional label at the instruction address.
	label := ""
	if sym, ok := d.getSymbol(addr); ok {
		label = sym.Name
	}
	labelCol := strings.Repeat(" ", codeInstrIndent)
	xrefComments := d.formatXRefs(addr)
	xrefComments = d.writeCodeLabelLine(&sb, addr, label, xrefComments)

	// Format mnemonic
	mnemonic := def.Op.String()

	// Format operand and resolve symbols for comments when needed.
	operandStr, operandSymbol := d.formatOperandWithSymbol(def, operand, addr)

	// Build the instruction line
	line := labelCol + mnemonic + operandStr
	operandXRefs := d.formatOperandXRefs(addr, def.OperandSize())

	// Collect all annotation lines (handling multi-line comments)
	annotationLines := d.getInlineCommentLines(addr, instrEnd)

	firstComment, continuation, operandXRefs := chooseInstructionComments(
		operandSymbol,
		annotationLines,
		xrefComments,
		operandXRefs,
	)

	writeInstructionWithComments(&sb, line, firstComment, continuation)

	// Output operand xrefs (self-modifying code references)
	for _, oxref := range operandXRefs {
		sb.WriteString(padToColumn("", instructionCommentCol) + "; " + oxref + "\n")
	}

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

// writeCodeLabelLine prints the label line for code, including label-related xrefs.
// It returns xrefs that should still be attached to instruction-level comments.
func (d *disassembler) writeCodeLabelLine(sb *strings.Builder, addr uint16, label string, xrefComments []string) []string {
	if label == "" {
		return xrefComments
	}

	labelLine := formatAddressOrLabelColumn(addr, label)
	labelXRef := ""
	labelXRefContinuation := []string(nil)
	if len(xrefComments) > 0 {
		labelXRef = xrefComments[0]
		labelXRefContinuation = xrefComments[1:]
		// Label-related xrefs are attached to the label line.
		xrefComments = nil
	}
	writeInstructionWithComments(sb, labelLine, labelXRef, labelXRefContinuation)
	return xrefComments
}

func chooseInstructionComments(
	operandSymbol string,
	annotationLines []string,
	xrefComments []string,
	operandXRefs []string,
) (string, []string, []string) {
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

	// If there are only operand xrefs, use the first as the inline comment.
	if firstComment == "" && len(operandXRefs) > 0 {
		firstComment = operandXRefs[0]
		continuation = append(continuation, operandXRefs[1:]...)
		operandXRefs = nil
	}

	return firstComment, continuation, operandXRefs
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

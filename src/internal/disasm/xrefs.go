package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/xrefs"
)

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
		line := "xref: " + d.formatXRefPart(ref)
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
			part := d.formatXRefPart(ref)
			parts = append(parts, part)
		}

		result = append(result, fmt.Sprintf("xref+%d: %s  [instruction data modified]", offset, strings.Join(parts, ", ")))
	}

	return result
}

func (d *disassembler) formatXRefPart(ref xrefs.XRef) string {
	part := fmt.Sprintf("%s from $%04X", ref.Type, ref.From)
	if label := d.labelAt(ref.From); label != "" {
		part += " (" + label + ")"
	}
	return part
}

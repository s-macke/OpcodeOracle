package disasm

import (
	"fmt"
	"strings"
)

const unknownRegionComment = "UNKNOWN REGION: no backing binary data"

// formatUnknownSpan formats a contiguous range of addresses that do not have
// backing bytes in the loaded binary.
func (d *disassembler) formatUnknownSpan(addr, end uint16) (string, int) {
	spanEnd := d.calculateUnknownSpanEnd(addr, end)
	size := int(spanEnd - addr)
	if size <= 0 {
		size = 1
	}

	return d.formatUnknownAt(addr, spanEnd), size
}

func (d *disassembler) formatUnknownAt(addr, spanEnd uint16) string {
	var sb strings.Builder

	// Keep headline behavior consistent with code/data formatting.
	d.writeHeadlines(&sb, addr, addr+1)

	for _, xref := range d.formatXRefs(addr) {
		sb.WriteString("; " + xref + "\n")
	}

	label := ""
	if sym, ok := d.getSymbol(addr); ok {
		label = sym.Name
	}
	labelCol := formatAddressOrLabelColumn(addr, label)

	comment := unknownRegionComment
	if spanEnd > addr+1 {
		comment = fmt.Sprintf("%s ($%04X-$%04X)", unknownRegionComment, addr, spanEnd-1)
	}

	inlineLines := d.getInlineCommentLines(addr, addr+1)
	writeInstructionWithComments(&sb, labelCol, comment, inlineLines)

	return sb.String()
}

func (d *disassembler) calculateUnknownSpanEnd(addr, end uint16) uint16 {
	limit := end
	for next := addr + 1; next < end; next++ {
		if d.hasBinaryByte(next) {
			limit = next
			break
		}
		if _, ok := d.state.Symbols.At(next); ok {
			limit = next
			break
		}
		if d.state.Headlines != nil && len(d.state.Headlines.At(next)) > 0 {
			limit = next
			break
		}
		if len(d.state.Annotations.At(next)) > 0 {
			limit = next
			break
		}
		if d.state.XRefs != nil && len(d.state.XRefs.To(next)) > 0 {
			limit = next
			break
		}
	}
	return limit
}

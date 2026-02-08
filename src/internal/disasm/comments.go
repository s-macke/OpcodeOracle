package disasm

import (
	"strings"

	"opcodeoracle/internal/headlines"
)

// inlineAnnotation is a simple struct to hold inline annotation data for disasm output.
type inlineAnnotation struct {
	Comment string
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

// getInlineCommentLines returns split inline annotation lines in [start, end).
func (d *disassembler) getInlineCommentLines(start, end uint16) []string {
	return splitInlineComments(d.getInlineAnnotations(start, end))
}

// formatHeadlines formats headline annotations as a block comment.
func (d *disassembler) formatHeadlines(hdls []headlines.Headline) string {
	var lines []string
	for _, h := range hdls {
		for _, line := range strings.Split(h.Comment, "\n") {
			lines = append(lines, line)
		}
	}
	return formatBlockComment(lines)
}

// formatInlinesAsHeadlines formats inline annotations as a headline block (for data).
func (d *disassembler) formatInlinesAsHeadlines(inlines []inlineAnnotation) string {
	return formatBlockComment(splitInlineComments(inlines))
}

func (d *disassembler) writeHeadlines(sb *strings.Builder, start, end uint16) {
	hdls := d.getHeadlines(start, end)
	if len(hdls) == 0 {
		return
	}
	sb.WriteString(d.formatHeadlines(hdls))
}

func splitInlineComments(inlines []inlineAnnotation) []string {
	var commentLines []string
	for _, ann := range inlines {
		for _, line := range strings.Split(ann.Comment, "\n") {
			commentLines = append(commentLines, line)
		}
	}
	return commentLines
}

func writeInstructionWithComments(sb *strings.Builder, line, firstComment string, continuation []string) {
	if firstComment == "" {
		sb.WriteString(line + "\n")
		return
	}

	sb.WriteString(padToColumn(line, instructionCommentCol) + "; " + firstComment + "\n")
	for _, comment := range continuation {
		sb.WriteString(padToColumn("", instructionCommentCol) + "; " + comment + "\n")
	}
}

func formatBlockComment(lines []string) string {
	var sb strings.Builder
	sb.WriteString("; --------------------------------------------------------\n")
	for _, line := range lines {
		sb.WriteString("; " + line + "\n")
	}
	sb.WriteString("; --------------------------------------------------------\n")
	return sb.String()
}

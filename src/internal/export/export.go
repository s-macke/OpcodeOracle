package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"opcodeoracle/internal/disasm"
	"opcodeoracle/internal/state"
)

// Exporter generates assembly output files from disassembled state.
type Exporter struct {
	state  *state.State
	disasm disasm.Disassembler
}

// NewExporter creates an exporter for the given state.
func NewExporter(s *state.State) *Exporter {
	return &Exporter{
		state:  s,
		disasm: disasm.NewDisassembler(s),
	}
}

// Export writes complete disassembly to a single file.
func (e *Exporter) Export(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	segments := e.identifySegments()

	var sb strings.Builder

	// Write header
	sb.WriteString(e.generateHeader(""))
	sb.WriteString("\n")

	// Write origin directive
	sb.WriteString(fmt.Sprintf("         .ORG $%04X\n\n", e.state.Binary.Origin))

	// Write each segment
	for _, seg := range segments {
		// Section title
		sb.WriteString(sectionTitle(seg) + "\n\n")

		// Disassembled content
		content, err := e.disasm.Disassemble(seg.Start, seg.End+1)
		if err != nil {
			return fmt.Errorf("disassembling segment at $%04X: %w", seg.Start, err)
		}
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	sb.WriteString("; End of disassembly\n")

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

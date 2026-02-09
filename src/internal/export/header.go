package export

import (
	"fmt"
	"time"

	"opcodeoracle/internal/segments"
)

const headerSeparator = "; ============================================================"

// generateHeader creates the file header with optional segment info.
func (e *Exporter) generateHeader(segmentInfo string) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	sourceFile := e.state.Metadata.SourceFile
	if sourceFile == "" {
		sourceFile = "unknown"
	}

	header := headerSeparator + "\n"
	header += "; AUTO-GENERATED FILE - DO NOT EDIT\n"
	header += headerSeparator + "\n"
	header += fmt.Sprintf("; Generated: %s\n", timestamp)
	header += fmt.Sprintf("; Source:    %s\n", sourceFile)
	if segmentInfo != "" {
		header += fmt.Sprintf("; Segment:   %s\n", segmentInfo)
	}
	header += headerSeparator + "\n"

	return header
}

// sectionTitle returns the section comment line for a segment.
func sectionTitle(seg segments.Segment) string {
	switch seg.Type {
	case segments.Sub:
		if seg.Name != "" {
			return fmt.Sprintf("; === SUBROUTINE: %s ===", seg.Name)
		}
		return fmt.Sprintf("; === SUBROUTINE @ $%04X ===", seg.Start)
	case segments.Code:
		return "; === CODE SECTION ==="
	case segments.Data:
		return "; === DATA SECTION ==="
	default:
		return "; === SECTION ==="
	}
}

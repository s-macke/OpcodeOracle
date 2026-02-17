package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/segments"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestIdentifySegments_DataOnly(t *testing.T) {
	s := state.NewState(make([]byte, 256), 0x0800, nil, "test.prg")
	// Default state has everything as data
	gotSegments := segments.Plan(s)

	if len(gotSegments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(gotSegments))
	}
	if gotSegments[0].Type != segments.Data {
		t.Errorf("expected data segment, got %s", gotSegments[0].Type)
	}
}

func TestIdentifySegments_CodeWithoutSubroutines(t *testing.T) {
	s := state.NewState(make([]byte, 256), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x08FF, regions.RegionCode)

	gotSegments := segments.Plan(s)

	// Should have: data before code, code segment, data after code
	var codeSegs []segments.Segment
	for _, seg := range gotSegments {
		if seg.Type == segments.Code {
			codeSegs = append(codeSegs, seg)
		}
	}

	if len(codeSegs) != 1 {
		t.Fatalf("expected 1 code segment, got %d", len(codeSegs))
	}
	if codeSegs[0].Start != 0x0800 || codeSegs[0].End != 0x08FF {
		t.Errorf("code segment has wrong bounds: %04X-%04X", codeSegs[0].Start, codeSegs[0].End)
	}
}

func TestIdentifySegments_CodeWithSubroutines(t *testing.T) {
	s := state.NewState(make([]byte, 0x1000), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0FFF, regions.RegionCode)

	// Add subroutine symbols
	s.Symbols.Add(0x0800, symbols.Symbol{Name: "main", Type: symbols.SymbolEntry, Source: symbols.SourceUser})
	s.Symbols.Add(0x0900, symbols.Symbol{Name: "sub1", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})
	s.Symbols.Add(0x0A00, symbols.Symbol{Name: "sub2", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})

	gotSegments := segments.Plan(s)

	// Find subroutine segments
	var subSegs []segments.Segment
	for _, seg := range gotSegments {
		if seg.Type == segments.Sub {
			subSegs = append(subSegs, seg)
		}
	}

	if len(subSegs) != 3 {
		t.Fatalf("expected 3 subroutine segments, got %d", len(subSegs))
	}

	// Check first subroutine (entry at 0x0800)
	if subSegs[0].Start != 0x0800 || subSegs[0].End != 0x08FF {
		t.Errorf("segment 0: expected 0800-08FF, got %04X-%04X", subSegs[0].Start, subSegs[0].End)
	}
	if subSegs[0].Name != "main" {
		t.Errorf("segment 0: expected name 'main', got '%s'", subSegs[0].Name)
	}

	// Check second subroutine
	if subSegs[1].Start != 0x0900 || subSegs[1].End != 0x09FF {
		t.Errorf("segment 1: expected 0900-09FF, got %04X-%04X", subSegs[1].Start, subSegs[1].End)
	}

	// Check third subroutine (extends to end of region)
	if subSegs[2].Start != 0x0A00 || subSegs[2].End != 0x0FFF {
		t.Errorf("segment 2: expected 0A00-0FFF, got %04X-%04X", subSegs[2].Start, subSegs[2].End)
	}
}

func TestIdentifySegments_CodeBeforeFirstSubroutine(t *testing.T) {
	s := state.NewState(make([]byte, 0x200), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x09FF, regions.RegionCode)

	// Subroutine starts at 0x0900, not at region start
	s.Symbols.Add(0x0900, symbols.Symbol{Name: "sub1", Type: symbols.SymbolSubroutine, Source: symbols.SourceAuto})

	gotSegments := segments.Plan(s)

	// Should have code segment before subroutine
	var codeSegs, subSegs []segments.Segment
	for _, seg := range gotSegments {
		switch seg.Type {
		case segments.Code:
			codeSegs = append(codeSegs, seg)
		case segments.Sub:
			subSegs = append(subSegs, seg)
		}
	}

	if len(codeSegs) != 1 {
		t.Fatalf("expected 1 code segment before subroutine, got %d", len(codeSegs))
	}
	if codeSegs[0].Start != 0x0800 || codeSegs[0].End != 0x08FF {
		t.Errorf("code segment: expected 0800-08FF, got %04X-%04X", codeSegs[0].Start, codeSegs[0].End)
	}

	if len(subSegs) != 1 {
		t.Fatalf("expected 1 subroutine segment, got %d", len(subSegs))
	}
	if subSegs[0].Start != 0x0900 || subSegs[0].End != 0x09FF {
		t.Errorf("sub segment: expected 0900-09FF, got %04X-%04X", subSegs[0].Start, subSegs[0].End)
	}
}

func TestSectionTitle(t *testing.T) {
	tests := []struct {
		seg      segments.Segment
		expected string
	}{
		{segments.Segment{Type: segments.Code}, "; === CODE SECTION ==="},
		{segments.Segment{Type: segments.Data}, "; === DATA SECTION ==="},
		{segments.Segment{Type: segments.Sub, Name: "foo"}, "; === SUBROUTINE: foo ==="},
		{segments.Segment{Type: segments.Sub, Start: 0x1000}, "; === SUBROUTINE @ $1000 ==="},
	}

	for _, tt := range tests {
		got := sectionTitle(tt.seg)
		if got != tt.expected {
			t.Errorf("sectionTitle(%v) = %q, want %q", tt.seg, got, tt.expected)
		}
	}
}

func TestGenerateHeader(t *testing.T) {
	s := state.NewState(make([]byte, 256), 0x0800, nil, "test.prg")
	exp := NewExporter(s, nil)

	header := exp.generateHeader("")

	if !strings.Contains(header, "AUTO-GENERATED FILE") {
		t.Error("header missing AUTO-GENERATED notice")
	}
	if !strings.Contains(header, "Source:    test.prg") {
		t.Error("header missing source file")
	}
	if !strings.Contains(header, "Generated:") {
		t.Error("header missing timestamp")
	}
}

func TestGenerateHeader_WithSegmentInfo(t *testing.T) {
	s := state.NewState(make([]byte, 256), 0x0800, nil, "test.prg")
	exp := NewExporter(s, nil)

	header := exp.generateHeader("subroutine @ $1000")

	if !strings.Contains(header, "Segment:   subroutine @ $1000") {
		t.Error("header missing segment info")
	}
}

func TestExport(t *testing.T) {
	// Create state with some code
	data := []byte{0xA9, 0x00, 0x8D, 0x20, 0xD0, 0x60} // LDA #$00, STA $D020, RTS
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0805, regions.RegionCode)

	exp := NewExporter(s, nil)

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.asm")

	err := exp.Export(outPath)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	// Check for expected content
	output := string(content)
	if !strings.Contains(output, ".ORG $0800") {
		t.Error("output missing .ORG directive")
	}
	if !strings.Contains(output, "LDA") {
		t.Error("output missing LDA instruction")
	}
	if !strings.Contains(output, "; End of disassembly") {
		t.Error("output missing end marker")
	}
}

func TestExportDeterministicAcrossRuns(t *testing.T) {
	buildState := func() *state.State {
		data := []byte{
			0x60,             // RTS at 0x0800 (xref target)
			0xEA,             // NOP padding
			0x4C, 0x00, 0x08, // JMP $0800 at 0x0802
			0x20, 0x00, 0x08, // JSR $0800 at 0x0805
			0xD0, 0xF6, // BNE $0800 at 0x0808
		}
		s := state.NewState(data, 0x0800, nil, "test.prg")
		s.Regions.Set(0x0800, 0x0809, regions.RegionCode)

		if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "main", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
			t.Fatalf("add symbol at $0800: %v", err)
		}
		if err := s.Symbols.Add(0x0805, symbols.Symbol{Name: "seed_sub", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
			t.Fatalf("add symbol at $0805: %v", err)
		}
		if err := s.Symbols.Add(0x0802, symbols.Symbol{Name: "seed_label", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
			t.Fatalf("add symbol at $0802: %v", err)
		}
		if err := s.Symbols.Add(0x0808, symbols.Symbol{Name: "seed_entry", Type: symbols.SymbolEntry, Source: symbols.SourceUser}); err != nil {
			t.Fatalf("add symbol at $0808: %v", err)
		}

		return s
	}

	normalizeHeader := func(s string) string {
		lines := strings.Split(s, "\n")
		var kept []string
		for _, line := range lines {
			if strings.HasPrefix(line, "; Generated: ") {
				continue
			}
			kept = append(kept, line)
		}
		return strings.Join(kept, "\n")
	}

	exportOnce := func(run int) string {
		s := buildState()
		analyzer := analysis.NewAnalyzer(s, analysis.UpdateXRefsOnly)
		if err := analyzer.Analyze(); err != nil {
			t.Fatalf("analysis run %d failed: %v", run, err)
		}

		exp := NewExporter(s, analyzer)
		outPath := filepath.Join(t.TempDir(), "output.asm")
		if err := exp.Export(outPath); err != nil {
			t.Fatalf("export run %d failed: %v", run, err)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read output run %d: %v", run, err)
		}
		return normalizeHeader(string(content))
	}

	const runs = 10
	var baseline string
	for i := 0; i < runs; i++ {
		current := exportOnce(i)
		if i == 0 {
			baseline = current
			continue
		}
		if current != baseline {
			t.Fatalf("export output differed on run %d\n--- baseline ---\n%s\n--- current ---\n%s", i, baseline, current)
		}
	}
}

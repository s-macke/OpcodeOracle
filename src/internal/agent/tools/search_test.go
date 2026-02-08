package tools

import (
	"context"
	"strings"
	"testing"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestSearchDisassemblyFindsMatchAndInfersAddress(t *testing.T) {
	data := []byte{
		0xA9, 0x00, // LDA #$00 at $0800
		0x60, // RTS
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Symbols.Add(0x0800, symbols.Symbol{
		Name:   "MAIN",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})

	tool := NewSearchDisassemblyTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{"query":"LDA"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "Found 1 matches (showing 1):") {
		t.Fatalf("expected one match header, got:\n%s", out)
	}
	if !strings.Contains(out, "[1] $0800") {
		t.Fatalf("expected inferred address $0800 for LDA line, got:\n%s", out)
	}
	if !strings.Contains(out, "LDA #$00") {
		t.Fatalf("expected matching LDA line in output, got:\n%s", out)
	}
}

func TestSearchDisassemblyCaseSensitivity(t *testing.T) {
	data := []byte{
		0xA9, 0x00, // LDA #$00
		0x60, // RTS
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)

	tool := NewSearchDisassemblyTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"query":"lda","case_sensitive":true}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "No disassembly matches found.") {
		t.Fatalf("expected no matches for case-sensitive lda, got:\n%s", out)
	}

	out, err = tool.InvokableRun(context.Background(), `{"query":"lda","case_sensitive":false}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Found 1 matches") {
		t.Fatalf("expected case-insensitive match, got:\n%s", out)
	}
}

func TestSearchDisassemblyMaxResults(t *testing.T) {
	data := []byte{
		0x60, // RTS at $0800
		0xEA, // NOP
		0x60, // RTS at $0802
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)

	tool := NewSearchDisassemblyTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{"query":"RTS","max_results":1}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Found 2 matches (showing 1):") {
		t.Fatalf("expected max_results truncation header, got:\n%s", out)
	}
}

func TestSearchDisassemblyEmptyQuery(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0800, regions.RegionCode)

	tool := NewSearchDisassemblyTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{"query":"   "}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Error: query cannot be empty") {
		t.Fatalf("expected empty-query error, got:\n%s", out)
	}
}

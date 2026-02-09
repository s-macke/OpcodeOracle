package tools

import (
	"context"
	"strings"
	"testing"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestListSubroutinesAndDataSegmentsDefaultsToBinaryBounds(t *testing.T) {
	s := state.NewState(make([]byte, 0x20), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x081F, regions.RegionCode)
	if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "entry", Type: symbols.SymbolEntry, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}
	if err := s.Symbols.Add(0x0810, symbols.Symbol{Name: "sub_0810", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "Found 2 segments in range $0800-$081F:") {
		t.Fatalf("expected default binary bounds in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Subroutine/Entry segments (2):") {
		t.Fatalf("expected subroutine section in output, got:\n%s", out)
	}
	if !strings.Contains(out, "$0800: entry") {
		t.Fatalf("expected entry symbol in output, got:\n%s", out)
	}
	if !strings.Contains(out, "$0810: sub_0810") {
		t.Fatalf("expected subroutine symbol in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsSupportsSingleBound(t *testing.T) {
	s := state.NewState(make([]byte, 0x30), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x082F, regions.RegionCode)
	if err := s.Symbols.Add(0x0801, symbols.Symbol{Name: "sub_0801", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}
	if err := s.Symbols.Add(0x0820, symbols.Symbol{Name: "sub_0820", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{"start_addr":"$0820"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "$0820: sub_0820") {
		t.Fatalf("expected in-range symbol in output, got:\n%s", out)
	}
	if strings.Contains(out, "$0801: sub_0801") {
		t.Fatalf("did not expect out-of-range symbol in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsIncludesOverlappingSegments(t *testing.T) {
	s := state.NewState(make([]byte, 0x40), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x083F, regions.RegionCode)
	if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "sub_0800", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}
	if err := s.Symbols.Add(0x0810, symbols.Symbol{Name: "sub_0810", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}
	if err := s.Symbols.Add(0x0820, symbols.Symbol{Name: "sub_0820", Type: symbols.SymbolSubroutine, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{"start_addr":"$0808","end_addr":"$0818"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "Found 2 segments in range $0808-$0818:") {
		t.Fatalf("expected overlap-filtered count in output, got:\n%s", out)
	}
	if !strings.Contains(out, "$0800: sub_0800") || !strings.Contains(out, "$0810: sub_0810") {
		t.Fatalf("expected overlapping segment symbols in output, got:\n%s", out)
	}
	if strings.Contains(out, "$0820: sub_0820") {
		t.Fatalf("did not expect non-overlapping symbol in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsMarksCodeAsDocumented(t *testing.T) {
	s := state.NewState(make([]byte, 0x40), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x083F, regions.RegionCode)
	s.Headlines.Set(0x0818, "middle of code", author.Assistant)

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "$0800-$083F: code [documented]") {
		t.Fatalf("expected documented code segment in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsMarksDataAsDocumented(t *testing.T) {
	s := state.NewState(make([]byte, 0x40), 0x0800, nil, "test.prg")
	// Entire range remains data by default
	s.Headlines.Set(0x0810, "data note", author.Assistant)

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "$0800-$083F: data [documented]") {
		t.Fatalf("expected documented data segment in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsShowsDataSymbol(t *testing.T) {
	s := state.NewState(make([]byte, 0x40), 0x0800, nil, "test.prg")
	if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "table_ptr", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "$0800-$083F: data (table_ptr, label)") {
		t.Fatalf("expected data symbol in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsDoesNotShowNonStartDataSymbol(t *testing.T) {
	s := state.NewState(make([]byte, 0x40), 0x0800, nil, "test.prg")
	if err := s.Symbols.Add(0x0810, symbols.Symbol{Name: "table_ptr", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if strings.Contains(out, "data (table_ptr, label)") {
		t.Fatalf("did not expect non-start data symbol in output, got:\n%s", out)
	}
}

func TestListSubroutinesAndDataSegmentsShowsCodeSymbol(t *testing.T) {
	s := state.NewState(make([]byte, 0x40), 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x083F, regions.RegionCode)
	if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "main_code", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("Add symbol error = %v, want nil", err)
	}

	tool := NewListSubroutinesAndDataSegmentsTool(&Context{State: s})
	out, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}

	if !strings.Contains(out, "$0800-$083F: code (main_code, label)") {
		t.Fatalf("expected code symbol in output, got:\n%s", out)
	}
}

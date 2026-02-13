package tools

import (
	"context"
	"strings"
	"testing"

	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestRemoveSymbolToolByAddress(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "entry", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("setup Add error = %v", err)
	}
	tool := NewRemoveSymbolTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Removed symbol at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	if _, ok := s.Symbols.At(0x0800); ok {
		t.Fatal("symbol should have been removed")
	}
}

func TestRemoveSymbolToolNameGuard(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	if err := s.Symbols.Add(0x0800, symbols.Symbol{Name: "entry", Type: symbols.SymbolLabel, Source: symbols.SourceUser}); err != nil {
		t.Fatalf("setup Add error = %v", err)
	}
	tool := NewRemoveSymbolTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800","name":"other"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Error: symbol at $0800 is 'entry', not 'other'") {
		t.Fatalf("unexpected output: %s", out)
	}
	if _, ok := s.Symbols.At(0x0800); !ok {
		t.Fatal("symbol should still exist after name mismatch")
	}
}

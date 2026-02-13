package tools

import (
	"context"
	"strings"
	"testing"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/state"
)

func TestReinterpretAsCodeTool(t *testing.T) {
	s := state.NewState([]byte{0x20, 0x04, 0x08, 0x60, 0xEA, 0x60}, 0x0800, []uint16{0x0800}, "test.prg")
	analyzer, err := analysis.ReanalyzeFromScratch(s)
	if err != nil {
		t.Fatalf("setup reanalyze error = %v", err)
	}
	tool := NewReinterpretAsCodeTool(&Context{State: s, Analyzer: analyzer})

	out, err := tool.InvokableRun(context.Background(), `{"code_address":"$0804"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Reinterpreted $0804 as code") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReinterpretAsDataToolHardLock(t *testing.T) {
	s := state.NewState([]byte{0x4C, 0x03, 0x08, 0x60}, 0x0800, []uint16{0x0800}, "test.prg")
	analyzer, err := analysis.ReanalyzeFromScratch(s)
	if err != nil {
		t.Fatalf("setup reanalyze error = %v", err)
	}
	ctx := &Context{State: s, Analyzer: analyzer}
	tool := NewReinterpretAsDataTool(ctx)

	out, err := tool.InvokableRun(context.Background(), `{"start_addr":"$0803","end_addr":"$0803"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "hard-locked data") {
		t.Fatalf("unexpected output: %s", out)
	}
	if ctx.Analyzer.IsInstructionAt(0x0803) {
		t.Fatal("expected 0x0803 to remain data after reinterpretation")
	}
}

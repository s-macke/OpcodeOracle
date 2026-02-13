package tools

import (
	"context"
	"strings"
	"testing"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/state"
)

func TestAddAnnotationToolExtendAppends(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Annotations.Set(0x0800, "first", author.Assistant)
	tool := NewAddAnnotationTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800","comment":"second","extend":true}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Extended annotation at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	got := s.Annotations.Get(0x0800, author.Assistant)
	if got == nil || got.Comment != "first\nsecond" {
		t.Fatalf("annotation comment = %#v, want first\\nsecond", got)
	}
}

func TestAddAnnotationToolWithoutExtendReplaces(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Annotations.Set(0x0800, "first", author.Assistant)
	tool := NewAddAnnotationTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800","comment":"second"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Added annotation at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	got := s.Annotations.Get(0x0800, author.Assistant)
	if got == nil || got.Comment != "second" {
		t.Fatalf("annotation comment = %#v, want second", got)
	}
}

func TestAddHeadlineToolExtendAppends(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Headlines.Set(0x0800, "first", author.Assistant)
	tool := NewAddHeadlineTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800","comment":"second","extend":true}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Extended headline at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	got := s.Headlines.Get(0x0800, author.Assistant)
	if got == nil || got.Comment != "first\nsecond" {
		t.Fatalf("headline comment = %#v, want first\\nsecond", got)
	}
}

func TestAddHeadlineToolExtendDryRunMessage(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	tool := NewAddHeadlineTool(&Context{State: s, DryRun: true})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800","comment":"second","extend":true}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Would extend headline at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
}

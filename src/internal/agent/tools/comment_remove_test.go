package tools

import (
	"context"
	"strings"
	"testing"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/state"
)

func TestRemoveAnnotationToolDefaultAuthor(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Annotations.Set(0x0800, "note", author.Assistant)
	tool := NewRemoveAnnotationTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Removed annotation at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	if got := s.Annotations.Get(0x0800, author.Assistant); got != nil {
		t.Fatal("assistant annotation should have been removed")
	}
}

func TestRemoveAnnotationToolUserAuthor(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Annotations.Set(0x0800, "user note", author.User)
	tool := NewRemoveAnnotationTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800","author":"user"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Removed annotation at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	if got := s.Annotations.Get(0x0800, author.User); got != nil {
		t.Fatal("user annotation should have been removed")
	}
}

func TestRemoveHeadlineToolDefaultAuthor(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	s.Headlines.Set(0x0800, "headline", author.Assistant)
	tool := NewRemoveHeadlineTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "Removed headline at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
	if got := s.Headlines.Get(0x0800, author.Assistant); got != nil {
		t.Fatal("assistant headline should have been removed")
	}
}

func TestRemoveHeadlineToolMissing(t *testing.T) {
	s := state.NewState([]byte{0xEA}, 0x0800, nil, "test.prg")
	tool := NewRemoveHeadlineTool(&Context{State: s})

	out, err := tool.InvokableRun(context.Background(), `{"address":"$0800"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v, want nil", err)
	}
	if !strings.Contains(out, "No headline found at $0800") {
		t.Fatalf("unexpected output: %s", out)
	}
}

package annotations

import (
	"testing"
)

func TestAnnotationTable(t *testing.T) {
	table := NewTable()

	// Test empty table
	anns := table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) on empty table returned %d annotations, want 0", len(anns))
	}

	// Add annotation
	table.Add(0x0800, AnnotationInline, "Initialize", "user")

	anns = table.At(0x0800)
	if len(anns) != 1 {
		t.Fatalf("At(0x0800) returned %d annotations, want 1", len(anns))
	}
	if anns[0].Comment != "Initialize" {
		t.Errorf("At(0x0800)[0].Comment = %q, want %q", anns[0].Comment, "Initialize")
	}
	if anns[0].Type != AnnotationInline {
		t.Errorf("At(0x0800)[0].Type = %v, want %v", anns[0].Type, AnnotationInline)
	}

	// Add headline annotation
	table.Add(0x0800, AnnotationHeadline, "Main entry point", "auto")

	anns = table.At(0x0800)
	if len(anns) != 2 {
		t.Fatalf("At(0x0800) returned %d annotations, want 2", len(anns))
	}
}

func TestAnnotationTableRemove(t *testing.T) {
	table := NewTable()

	table.Add(0x0800, AnnotationInline, "First", "user")
	table.Add(0x0800, AnnotationInline, "Second", "user")
	table.Add(0x0800, AnnotationInline, "Third", "user")

	// Remove middle
	err := table.Remove(0x0800, 1)
	if err != nil {
		t.Errorf("Remove(0x0800, 1) returned error: %v", err)
	}

	anns := table.At(0x0800)
	if len(anns) != 2 {
		t.Fatalf("At(0x0800) returned %d annotations, want 2", len(anns))
	}
	if anns[0].Comment != "First" || anns[1].Comment != "Third" {
		t.Errorf("Wrong annotations after remove: %v", anns)
	}

	// Remove out of range
	err = table.Remove(0x0800, 5)
	if err != ErrIndexOutOfRange {
		t.Errorf("Remove(0x0800, 5) error = %v, want ErrIndexOutOfRange", err)
	}

	// Remove negative index
	err = table.Remove(0x0800, -1)
	if err != ErrIndexOutOfRange {
		t.Errorf("Remove(0x0800, -1) error = %v, want ErrIndexOutOfRange", err)
	}
}

func TestAnnotationTableClear(t *testing.T) {
	table := NewTable()

	table.Add(0x0800, AnnotationInline, "First", "user")
	table.Add(0x0800, AnnotationInline, "Second", "user")

	table.Clear(0x0800)

	anns := table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) after Clear returned %d annotations, want 0", len(anns))
	}

	// Clear nonexistent should not panic
	table.Clear(0x0900)
}

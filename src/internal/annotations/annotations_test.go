package annotations

import (
	"testing"

	"opcodeoracle/internal/author"
)

func TestAnnotationTable(t *testing.T) {
	table := NewTable()

	// Test empty table
	anns := table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) on empty table returned %d annotations, want 0", len(anns))
	}

	// Set user annotation
	table.Set(0x0800, "Initialize", author.User)

	anns = table.At(0x0800)
	if len(anns) != 1 {
		t.Fatalf("At(0x0800) returned %d annotations, want 1", len(anns))
	}
	if anns[0].Comment != "Initialize" {
		t.Errorf("At(0x0800)[0].Comment = %q, want %q", anns[0].Comment, "Initialize")
	}
	if anns[0].Author != author.User {
		t.Errorf("At(0x0800)[0].Author = %v, want %v", anns[0].Author, author.User)
	}

	// Set assistant annotation (different author, so both should exist)
	table.Set(0x0800, "Main entry point", author.Assistant)

	anns = table.At(0x0800)
	if len(anns) != 2 {
		t.Fatalf("At(0x0800) returned %d annotations, want 2", len(anns))
	}
}

func TestAnnotationTableOverwrite(t *testing.T) {
	table := NewTable()

	// Set user annotation
	table.Set(0x0800, "First", author.User)

	// Overwrite with same author - should replace
	table.Set(0x0800, "Second", author.User)

	anns := table.At(0x0800)
	if len(anns) != 1 {
		t.Fatalf("At(0x0800) returned %d annotations, want 1 (overwrite)", len(anns))
	}
	if anns[0].Comment != "Second" {
		t.Errorf("At(0x0800)[0].Comment = %q, want %q", anns[0].Comment, "Second")
	}
}

func TestAnnotationTableGet(t *testing.T) {
	table := NewTable()

	// Get from empty table
	if ann := table.Get(0x0800, author.User); ann != nil {
		t.Errorf("Get(0x0800, User) on empty table = %v, want nil", ann)
	}

	// Set and get
	table.Set(0x0800, "User comment", author.User)
	table.Set(0x0800, "Assistant comment", author.Assistant)

	userAnn := table.Get(0x0800, author.User)
	if userAnn == nil {
		t.Fatal("Get(0x0800, User) = nil, want annotation")
	}
	if userAnn.Comment != "User comment" {
		t.Errorf("userAnn.Comment = %q, want %q", userAnn.Comment, "User comment")
	}

	assistantAnn := table.Get(0x0800, author.Assistant)
	if assistantAnn == nil {
		t.Fatal("Get(0x0800, Assistant) = nil, want annotation")
	}
	if assistantAnn.Comment != "Assistant comment" {
		t.Errorf("assistantAnn.Comment = %q, want %q", assistantAnn.Comment, "Assistant comment")
	}
}

func TestAnnotationTableRemove(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, "User", author.User)
	table.Set(0x0800, "Assistant", author.Assistant)

	// Remove user annotation
	table.Remove(0x0800, author.User)

	if ann := table.Get(0x0800, author.User); ann != nil {
		t.Errorf("Get after Remove(User) = %v, want nil", ann)
	}

	// Assistant should still exist
	if ann := table.Get(0x0800, author.Assistant); ann == nil {
		t.Error("Get(Assistant) after Remove(User) = nil, want annotation")
	}

	anns := table.At(0x0800)
	if len(anns) != 1 {
		t.Errorf("At(0x0800) returned %d annotations, want 1", len(anns))
	}

	// Remove assistant (should clean up map entry)
	table.Remove(0x0800, author.Assistant)

	anns = table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) after removing all returned %d annotations, want 0", len(anns))
	}

	// Remove from nonexistent address should not panic
	table.Remove(0x0900, author.User)
}

func TestAnnotationTableClear(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, "User", author.User)
	table.Set(0x0800, "Assistant", author.Assistant)

	table.Clear(0x0800)

	anns := table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) after Clear returned %d annotations, want 0", len(anns))
	}

	// Clear nonexistent should not panic
	table.Clear(0x0900)
}

func TestAnnotationTableAll(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, "User 0800", author.User)
	table.Set(0x0801, "Assistant 0801", author.Assistant)
	table.Set(0x0801, "User 0801", author.User)

	all := table.All()

	if len(all) != 2 {
		t.Errorf("All() returned %d addresses, want 2", len(all))
	}

	addr0800 := all[0x0800]
	if addr0800 == nil {
		t.Fatal("All()[0x0800] = nil, want AddressAnnotations")
	}
	if addr0800.User == nil || addr0800.User.Comment != "User 0800" {
		t.Errorf("All()[0x0800].User = %v, want User 0800", addr0800.User)
	}
	if addr0800.Assistant != nil {
		t.Errorf("All()[0x0800].Assistant = %v, want nil", addr0800.Assistant)
	}

	addr0801 := all[0x0801]
	if addr0801 == nil {
		t.Fatal("All()[0x0801] = nil, want AddressAnnotations")
	}
	if addr0801.User == nil || addr0801.User.Comment != "User 0801" {
		t.Errorf("All()[0x0801].User = %v, want User 0801", addr0801.User)
	}
	if addr0801.Assistant == nil || addr0801.Assistant.Comment != "Assistant 0801" {
		t.Errorf("All()[0x0801].Assistant = %v, want Assistant 0801", addr0801.Assistant)
	}
}

func TestAnnotationTableExtend(t *testing.T) {
	table := NewTable()

	// Extend on empty address should create new annotation
	table.Extend(0x0800, "First line", author.User)

	ann := table.Get(0x0800, author.User)
	if ann == nil {
		t.Fatal("Get(0x0800, User) after Extend = nil, want annotation")
	}
	if ann.Comment != "First line" {
		t.Errorf("ann.Comment = %q, want %q", ann.Comment, "First line")
	}

	// Extend existing annotation should append with newline
	table.Extend(0x0800, "Second line", author.User)

	ann = table.Get(0x0800, author.User)
	if ann == nil {
		t.Fatal("Get(0x0800, User) after second Extend = nil, want annotation")
	}
	want := "First line\nSecond line"
	if ann.Comment != want {
		t.Errorf("ann.Comment = %q, want %q", ann.Comment, want)
	}

	// Extend should not affect other authors
	table.Extend(0x0800, "Assistant note", author.Assistant)

	userAnn := table.Get(0x0800, author.User)
	if userAnn.Comment != want {
		t.Errorf("User annotation changed after extending assistant: %q", userAnn.Comment)
	}

	assistantAnn := table.Get(0x0800, author.Assistant)
	if assistantAnn == nil {
		t.Fatal("Get(0x0800, Assistant) = nil, want annotation")
	}
	if assistantAnn.Comment != "Assistant note" {
		t.Errorf("assistantAnn.Comment = %q, want %q", assistantAnn.Comment, "Assistant note")
	}
}

func TestAnnotationTableExtendMultiple(t *testing.T) {
	table := NewTable()

	// Extend multiple times
	table.Extend(0x0800, "Line 1", author.User)
	table.Extend(0x0800, "Line 2", author.User)
	table.Extend(0x0800, "Line 3", author.User)

	ann := table.Get(0x0800, author.User)
	if ann == nil {
		t.Fatal("Get(0x0800, User) = nil, want annotation")
	}

	want := "Line 1\nLine 2\nLine 3"
	if ann.Comment != want {
		t.Errorf("ann.Comment = %q, want %q", ann.Comment, want)
	}
}

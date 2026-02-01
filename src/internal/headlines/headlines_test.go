package headlines

import (
	"testing"

	"opcodeoracle/internal/author"
)

func TestHeadlineTable(t *testing.T) {
	table := NewTable()

	// Test empty table
	hdls := table.At(0x0800)
	if len(hdls) != 0 {
		t.Errorf("At(0x0800) on empty table returned %d headlines, want 0", len(hdls))
	}

	// Set user headline
	table.Set(0x0800, "Section Start", author.User)

	hdls = table.At(0x0800)
	if len(hdls) != 1 {
		t.Fatalf("At(0x0800) returned %d headlines, want 1", len(hdls))
	}
	if hdls[0].Comment != "Section Start" {
		t.Errorf("At(0x0800)[0].Comment = %q, want %q", hdls[0].Comment, "Section Start")
	}
	if hdls[0].Author != author.User {
		t.Errorf("At(0x0800)[0].Author = %v, want %v", hdls[0].Author, author.User)
	}

	// Set assistant headline (different author, so both should exist)
	table.Set(0x0800, "Main entry point", author.Assistant)

	hdls = table.At(0x0800)
	if len(hdls) != 2 {
		t.Fatalf("At(0x0800) returned %d headlines, want 2", len(hdls))
	}
}

func TestHeadlineTableOverwrite(t *testing.T) {
	table := NewTable()

	// Set user headline
	table.Set(0x0800, "First", author.User)

	// Overwrite with same author - should replace
	table.Set(0x0800, "Second", author.User)

	hdls := table.At(0x0800)
	if len(hdls) != 1 {
		t.Fatalf("At(0x0800) returned %d headlines, want 1 (overwrite)", len(hdls))
	}
	if hdls[0].Comment != "Second" {
		t.Errorf("At(0x0800)[0].Comment = %q, want %q", hdls[0].Comment, "Second")
	}
}

func TestHeadlineTableGet(t *testing.T) {
	table := NewTable()

	// Get from empty table
	if hdl := table.Get(0x0800, author.User); hdl != nil {
		t.Errorf("Get(0x0800, User) on empty table = %v, want nil", hdl)
	}

	// Set and get
	table.Set(0x0800, "User headline", author.User)
	table.Set(0x0800, "Assistant headline", author.Assistant)

	userHdl := table.Get(0x0800, author.User)
	if userHdl == nil {
		t.Fatal("Get(0x0800, User) = nil, want headline")
	}
	if userHdl.Comment != "User headline" {
		t.Errorf("userHdl.Comment = %q, want %q", userHdl.Comment, "User headline")
	}

	assistantHdl := table.Get(0x0800, author.Assistant)
	if assistantHdl == nil {
		t.Fatal("Get(0x0800, Assistant) = nil, want headline")
	}
	if assistantHdl.Comment != "Assistant headline" {
		t.Errorf("assistantHdl.Comment = %q, want %q", assistantHdl.Comment, "Assistant headline")
	}
}

func TestHeadlineTableRemove(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, "User", author.User)
	table.Set(0x0800, "Assistant", author.Assistant)

	// Remove user headline
	table.Remove(0x0800, author.User)

	if hdl := table.Get(0x0800, author.User); hdl != nil {
		t.Errorf("Get after Remove(User) = %v, want nil", hdl)
	}

	// Assistant should still exist
	if hdl := table.Get(0x0800, author.Assistant); hdl == nil {
		t.Error("Get(Assistant) after Remove(User) = nil, want headline")
	}

	hdls := table.At(0x0800)
	if len(hdls) != 1 {
		t.Errorf("At(0x0800) returned %d headlines, want 1", len(hdls))
	}

	// Remove assistant (should clean up map entry)
	table.Remove(0x0800, author.Assistant)

	hdls = table.At(0x0800)
	if len(hdls) != 0 {
		t.Errorf("At(0x0800) after removing all returned %d headlines, want 0", len(hdls))
	}

	// Remove from nonexistent address should not panic
	table.Remove(0x0900, author.User)
}

func TestHeadlineTableClear(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, "User", author.User)
	table.Set(0x0800, "Assistant", author.Assistant)

	table.Clear(0x0800)

	hdls := table.At(0x0800)
	if len(hdls) != 0 {
		t.Errorf("At(0x0800) after Clear returned %d headlines, want 0", len(hdls))
	}

	// Clear nonexistent should not panic
	table.Clear(0x0900)
}

func TestHeadlineTableAll(t *testing.T) {
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
		t.Fatal("All()[0x0800] = nil, want AddressHeadlines")
	}
	if addr0800.User == nil || addr0800.User.Comment != "User 0800" {
		t.Errorf("All()[0x0800].User = %v, want User 0800", addr0800.User)
	}
	if addr0800.Assistant != nil {
		t.Errorf("All()[0x0800].Assistant = %v, want nil", addr0800.Assistant)
	}

	addr0801 := all[0x0801]
	if addr0801 == nil {
		t.Fatal("All()[0x0801] = nil, want AddressHeadlines")
	}
	if addr0801.User == nil || addr0801.User.Comment != "User 0801" {
		t.Errorf("All()[0x0801].User = %v, want User 0801", addr0801.User)
	}
	if addr0801.Assistant == nil || addr0801.Assistant.Comment != "Assistant 0801" {
		t.Errorf("All()[0x0801].Assistant = %v, want Assistant 0801", addr0801.Assistant)
	}
}

func TestHeadlineTableExtend(t *testing.T) {
	table := NewTable()

	// Extend on empty address should create new headline
	table.Extend(0x0800, "First line", author.User)

	hdl := table.Get(0x0800, author.User)
	if hdl == nil {
		t.Fatal("Get(0x0800, User) after Extend = nil, want headline")
	}
	if hdl.Comment != "First line" {
		t.Errorf("hdl.Comment = %q, want %q", hdl.Comment, "First line")
	}

	// Extend existing headline should append with newline
	table.Extend(0x0800, "Second line", author.User)

	hdl = table.Get(0x0800, author.User)
	if hdl == nil {
		t.Fatal("Get(0x0800, User) after second Extend = nil, want headline")
	}
	want := "First line\nSecond line"
	if hdl.Comment != want {
		t.Errorf("hdl.Comment = %q, want %q", hdl.Comment, want)
	}

	// Extend should not affect other authors
	table.Extend(0x0800, "Assistant note", author.Assistant)

	userHdl := table.Get(0x0800, author.User)
	if userHdl.Comment != want {
		t.Errorf("User headline changed after extending assistant: %q", userHdl.Comment)
	}

	assistantHdl := table.Get(0x0800, author.Assistant)
	if assistantHdl == nil {
		t.Fatal("Get(0x0800, Assistant) = nil, want headline")
	}
	if assistantHdl.Comment != "Assistant note" {
		t.Errorf("assistantHdl.Comment = %q, want %q", assistantHdl.Comment, "Assistant note")
	}
}

func TestHeadlineTableExtendMultiple(t *testing.T) {
	table := NewTable()

	// Extend multiple times
	table.Extend(0x0800, "Line 1", author.User)
	table.Extend(0x0800, "Line 2", author.User)
	table.Extend(0x0800, "Line 3", author.User)

	hdl := table.Get(0x0800, author.User)
	if hdl == nil {
		t.Fatal("Get(0x0800, User) = nil, want headline")
	}

	want := "Line 1\nLine 2\nLine 3"
	if hdl.Comment != want {
		t.Errorf("hdl.Comment = %q, want %q", hdl.Comment, want)
	}
}

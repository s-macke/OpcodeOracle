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

	// Set user annotation
	table.Set(0x0800, AnnotationInline, "Initialize", AuthorUser)

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
	if anns[0].Author != AuthorUser {
		t.Errorf("At(0x0800)[0].Author = %v, want %v", anns[0].Author, AuthorUser)
	}

	// Set assistant annotation (different author, so both should exist)
	table.Set(0x0800, AnnotationHeadline, "Main entry point", AuthorAssistant)

	anns = table.At(0x0800)
	if len(anns) != 2 {
		t.Fatalf("At(0x0800) returned %d annotations, want 2", len(anns))
	}
}

func TestAnnotationTableOverwrite(t *testing.T) {
	table := NewTable()

	// Set user annotation
	table.Set(0x0800, AnnotationInline, "First", AuthorUser)

	// Overwrite with same author - should replace
	table.Set(0x0800, AnnotationHeadline, "Second", AuthorUser)

	anns := table.At(0x0800)
	if len(anns) != 1 {
		t.Fatalf("At(0x0800) returned %d annotations, want 1 (overwrite)", len(anns))
	}
	if anns[0].Comment != "Second" {
		t.Errorf("At(0x0800)[0].Comment = %q, want %q", anns[0].Comment, "Second")
	}
	if anns[0].Type != AnnotationHeadline {
		t.Errorf("At(0x0800)[0].Type = %v, want %v", anns[0].Type, AnnotationHeadline)
	}
}

func TestAnnotationTableGet(t *testing.T) {
	table := NewTable()

	// Get from empty table
	if ann := table.Get(0x0800, AuthorUser); ann != nil {
		t.Errorf("Get(0x0800, AuthorUser) on empty table = %v, want nil", ann)
	}

	// Set and get
	table.Set(0x0800, AnnotationInline, "User comment", AuthorUser)
	table.Set(0x0800, AnnotationHeadline, "Assistant comment", AuthorAssistant)

	userAnn := table.Get(0x0800, AuthorUser)
	if userAnn == nil {
		t.Fatal("Get(0x0800, AuthorUser) = nil, want annotation")
	}
	if userAnn.Comment != "User comment" {
		t.Errorf("userAnn.Comment = %q, want %q", userAnn.Comment, "User comment")
	}

	assistantAnn := table.Get(0x0800, AuthorAssistant)
	if assistantAnn == nil {
		t.Fatal("Get(0x0800, AuthorAssistant) = nil, want annotation")
	}
	if assistantAnn.Comment != "Assistant comment" {
		t.Errorf("assistantAnn.Comment = %q, want %q", assistantAnn.Comment, "Assistant comment")
	}
}

func TestAnnotationTableRemove(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, AnnotationInline, "User", AuthorUser)
	table.Set(0x0800, AnnotationInline, "Assistant", AuthorAssistant)

	// Remove user annotation
	table.Remove(0x0800, AuthorUser)

	if ann := table.Get(0x0800, AuthorUser); ann != nil {
		t.Errorf("Get after Remove(AuthorUser) = %v, want nil", ann)
	}

	// Assistant should still exist
	if ann := table.Get(0x0800, AuthorAssistant); ann == nil {
		t.Error("Get(AuthorAssistant) after Remove(AuthorUser) = nil, want annotation")
	}

	anns := table.At(0x0800)
	if len(anns) != 1 {
		t.Errorf("At(0x0800) returned %d annotations, want 1", len(anns))
	}

	// Remove assistant (should clean up map entry)
	table.Remove(0x0800, AuthorAssistant)

	anns = table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) after removing all returned %d annotations, want 0", len(anns))
	}

	// Remove from nonexistent address should not panic
	table.Remove(0x0900, AuthorUser)
}

func TestAnnotationTableClear(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, AnnotationInline, "User", AuthorUser)
	table.Set(0x0800, AnnotationInline, "Assistant", AuthorAssistant)

	table.Clear(0x0800)

	anns := table.At(0x0800)
	if len(anns) != 0 {
		t.Errorf("At(0x0800) after Clear returned %d annotations, want 0", len(anns))
	}

	// Clear nonexistent should not panic
	table.Clear(0x0900)
}

func TestAuthorString(t *testing.T) {
	tests := []struct {
		author Author
		want   string
	}{
		{AuthorUser, "user"},
		{AuthorAssistant, "assistant"},
		{Author(99), "unknown"},
	}

	for _, tc := range tests {
		got := tc.author.String()
		if got != tc.want {
			t.Errorf("Author(%d).String() = %q, want %q", tc.author, got, tc.want)
		}
	}
}

func TestParseAuthor(t *testing.T) {
	tests := []struct {
		input   string
		want    Author
		wantErr bool
	}{
		{"user", AuthorUser, false},
		{"assistant", AuthorAssistant, false},
		{"User", 0, true},
		{"ASSISTANT", 0, true},
		{"auto", 0, true},
		{"", 0, true},
	}

	for _, tc := range tests {
		got, err := ParseAuthor(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseAuthor(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("ParseAuthor(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestAnnotationTableAll(t *testing.T) {
	table := NewTable()

	table.Set(0x0800, AnnotationInline, "User 0800", AuthorUser)
	table.Set(0x0801, AnnotationHeadline, "Assistant 0801", AuthorAssistant)
	table.Set(0x0801, AnnotationInline, "User 0801", AuthorUser)

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

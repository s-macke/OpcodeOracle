package xrefs

import (
	"testing"
)

func TestXRefTable(t *testing.T) {
	table := NewTable()

	// Test empty table
	refs := table.To(0x0800)
	if len(refs) != 0 {
		t.Errorf("To(0x0800) on empty table returned %d refs, want 0", len(refs))
	}

	refs = table.From(0x0800)
	if len(refs) != 0 {
		t.Errorf("From(0x0800) on empty table returned %d refs, want 0", len(refs))
	}

	// Add xref
	table.Add(0x0810, 0x0900, XRefCall)

	refs = table.To(0x0900)
	if len(refs) != 1 {
		t.Fatalf("To(0x0900) returned %d refs, want 1", len(refs))
	}
	if refs[0].From != 0x0810 || refs[0].Type != XRefCall {
		t.Errorf("To(0x0900)[0] = {%04X, %04X, %v}, want {0810, 0900, call}",
			refs[0].From, refs[0].To, refs[0].Type)
	}

	refs = table.From(0x0810)
	if len(refs) != 1 {
		t.Fatalf("From(0x0810) returned %d refs, want 1", len(refs))
	}
}

func TestXRefTableMultiple(t *testing.T) {
	table := NewTable()

	// Multiple refs to same target
	table.Add(0x0810, 0x0900, XRefCall)
	table.Add(0x0820, 0x0900, XRefJump)
	table.Add(0x0830, 0x0900, XRefBranch)

	refs := table.To(0x0900)
	if len(refs) != 3 {
		t.Fatalf("To(0x0900) returned %d refs, want 3", len(refs))
	}

	// Multiple refs from same source
	table.Add(0x0840, 0x0950, XRefRead)
	table.Add(0x0840, 0x0960, XRefWrite)

	refs = table.From(0x0840)
	if len(refs) != 2 {
		t.Fatalf("From(0x0840) returned %d refs, want 2", len(refs))
	}
}

func TestXRefTableRemove(t *testing.T) {
	table := NewTable()

	table.Add(0x0810, 0x0900, XRefCall)
	table.Add(0x0820, 0x0900, XRefJump)
	table.Add(0x0810, 0x0900, XRefRead) // Same from/to, different type

	// Remove all refs from 0x0810 to 0x0900
	table.Remove(0x0810, 0x0900)

	refs := table.To(0x0900)
	if len(refs) != 1 {
		t.Fatalf("To(0x0900) after remove returned %d refs, want 1", len(refs))
	}
	if refs[0].From != 0x0820 {
		t.Errorf("Remaining ref from = %04X, want 0820", refs[0].From)
	}
}

func TestXRefTableRemoveNonexistent(t *testing.T) {
	table := NewTable()

	// Removing from empty table should not panic
	table.Remove(0x0800, 0x0900)

	table.Add(0x0810, 0x0900, XRefCall)
	table.Remove(0x0800, 0x0900) // Different from

	refs := table.To(0x0900)
	if len(refs) != 1 {
		t.Errorf("To(0x0900) returned %d refs, want 1", len(refs))
	}
}

func TestXRefTableDuplicate(t *testing.T) {
	table := NewTable()

	// Add same xref twice
	table.Add(0x0810, 0x0900, XRefCall)
	table.Add(0x0810, 0x0900, XRefCall)

	refs := table.To(0x0900)
	if len(refs) != 1 {
		t.Errorf("To(0x0900) returned %d refs, want 1 (duplicate should be ignored)", len(refs))
	}

	// Same from/to but different type is not a duplicate
	table.Add(0x0810, 0x0900, XRefJump)

	refs = table.To(0x0900)
	if len(refs) != 2 {
		t.Errorf("To(0x0900) returned %d refs, want 2", len(refs))
	}
}

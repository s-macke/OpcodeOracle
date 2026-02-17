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

func TestXRefTableReturnsCanonicalOrder(t *testing.T) {
	table := NewTable()

	// Intentionally shuffled insertion order.
	table.Add(0x0830, 0x0900, XRefJump)
	table.Add(0x0810, 0x0900, XRefCall)
	table.Add(0x0810, 0x0900, XRefBranch)
	table.Add(0x0810, 0x0950, XRefRead)
	table.Add(0x0810, 0x0940, XRefRead)

	toRefs := table.To(0x0900)
	if len(toRefs) != 3 {
		t.Fatalf("To(0x0900) returned %d refs, want 3", len(toRefs))
	}
	if toRefs[0].From != 0x0810 || toRefs[0].Type != XRefBranch {
		t.Fatalf("To order[0] = {%04X,%04X,%s}, want {0810,0900,branch}", toRefs[0].From, toRefs[0].To, toRefs[0].Type)
	}
	if toRefs[1].From != 0x0810 || toRefs[1].Type != XRefCall {
		t.Fatalf("To order[1] = {%04X,%04X,%s}, want {0810,0900,call}", toRefs[1].From, toRefs[1].To, toRefs[1].Type)
	}
	if toRefs[2].From != 0x0830 || toRefs[2].Type != XRefJump {
		t.Fatalf("To order[2] = {%04X,%04X,%s}, want {0830,0900,jump}", toRefs[2].From, toRefs[2].To, toRefs[2].Type)
	}

	fromRefs := table.From(0x0810)
	if len(fromRefs) != 4 {
		t.Fatalf("From(0x0810) returned %d refs, want 4", len(fromRefs))
	}
	if fromRefs[0].Type != XRefBranch || fromRefs[0].To != 0x0900 {
		t.Fatalf("From order[0] = {%04X,%04X,%s}, want {0810,0900,branch}", fromRefs[0].From, fromRefs[0].To, fromRefs[0].Type)
	}
	if fromRefs[1].Type != XRefCall || fromRefs[1].To != 0x0900 {
		t.Fatalf("From order[1] = {%04X,%04X,%s}, want {0810,0900,call}", fromRefs[1].From, fromRefs[1].To, fromRefs[1].Type)
	}
	if fromRefs[2].Type != XRefRead || fromRefs[2].To != 0x0940 {
		t.Fatalf("From order[2] = {%04X,%04X,%s}, want {0810,0940,read}", fromRefs[2].From, fromRefs[2].To, fromRefs[2].Type)
	}
	if fromRefs[3].Type != XRefRead || fromRefs[3].To != 0x0950 {
		t.Fatalf("From order[3] = {%04X,%04X,%s}, want {0810,0950,read}", fromRefs[3].From, fromRefs[3].To, fromRefs[3].Type)
	}
}

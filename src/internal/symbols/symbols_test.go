package symbols

import (
	"testing"
)

func TestSymbolTable(t *testing.T) {
	table := NewTable()

	// Test empty table
	syms := table.At(0x0800)
	if len(syms) != 0 {
		t.Errorf("At(0x0800) on empty table returned %d symbols, want 0", len(syms))
	}

	// Add symbol
	sym1 := Symbol{Name: "start", Type: SymbolEntry, Source: SourceUser}
	table.Add(0x0800, sym1)

	syms = table.At(0x0800)
	if len(syms) != 1 {
		t.Fatalf("At(0x0800) returned %d symbols, want 1", len(syms))
	}
	if syms[0].Name != "start" {
		t.Errorf("At(0x0800)[0].Name = %q, want %q", syms[0].Name, "start")
	}

	// Add another symbol at same address
	sym2 := Symbol{Name: "L_0800", Type: SymbolLabel, Source: SourceAuto}
	table.Add(0x0800, sym2)

	syms = table.At(0x0800)
	if len(syms) != 2 {
		t.Fatalf("At(0x0800) returned %d symbols, want 2", len(syms))
	}

	// Remove first symbol
	table.Remove(0x0800, "start")
	syms = table.At(0x0800)
	if len(syms) != 1 {
		t.Fatalf("At(0x0800) after remove returned %d symbols, want 1", len(syms))
	}
	if syms[0].Name != "L_0800" {
		t.Errorf("At(0x0800)[0].Name = %q, want %q", syms[0].Name, "L_0800")
	}

	// Remove last symbol
	table.Remove(0x0800, "L_0800")
	syms = table.At(0x0800)
	if len(syms) != 0 {
		t.Errorf("At(0x0800) after removing all returned %d symbols, want 0", len(syms))
	}
}

func TestSymbolTableRemoveNonexistent(t *testing.T) {
	table := NewTable()

	// Removing from empty table should not panic
	table.Remove(0x0800, "nonexistent")

	// Add and remove different name should not affect existing
	table.Add(0x0800, Symbol{Name: "test", Type: SymbolLabel, Source: SourceUser})
	table.Remove(0x0800, "other")

	syms := table.At(0x0800)
	if len(syms) != 1 {
		t.Errorf("At(0x0800) returned %d symbols, want 1", len(syms))
	}
}

func TestSymbolTableDuplicate(t *testing.T) {
	table := NewTable()

	sym := Symbol{Name: "start", Type: SymbolEntry, Source: SourceUser}

	// Add same symbol twice
	table.Add(0x0800, sym)
	table.Add(0x0800, sym)

	syms := table.At(0x0800)
	if len(syms) != 1 {
		t.Errorf("At(0x0800) returned %d symbols, want 1 (duplicate should be ignored)", len(syms))
	}

	// Same name but different type is not a duplicate
	sym2 := Symbol{Name: "start", Type: SymbolLabel, Source: SourceUser}
	table.Add(0x0800, sym2)

	syms = table.At(0x0800)
	if len(syms) != 2 {
		t.Errorf("At(0x0800) returned %d symbols, want 2", len(syms))
	}
}

func TestSubroutinesInRange(t *testing.T) {
	t.Run("empty table returns empty slice", func(t *testing.T) {
		table := NewTable()
		result := table.SubroutinesInRange(0x0800, 0x0900)
		if len(result) != 0 {
			t.Errorf("SubroutinesInRange on empty table returned %d symbols, want 0", len(result))
		}
	})

	t.Run("symbols outside range are excluded", func(t *testing.T) {
		table := NewTable()
		table.Add(0x0700, Symbol{Name: "before", Type: SymbolSubroutine, Source: SourceUser})
		table.Add(0x0800, Symbol{Name: "inside", Type: SymbolSubroutine, Source: SourceUser})
		table.Add(0x0900, Symbol{Name: "atEnd", Type: SymbolSubroutine, Source: SourceUser})
		table.Add(0x0901, Symbol{Name: "after", Type: SymbolSubroutine, Source: SourceUser})

		result := table.SubroutinesInRange(0x0800, 0x0900)
		if len(result) != 2 {
			t.Fatalf("SubroutinesInRange returned %d symbols, want 2", len(result))
		}
		if result[0].Symbol.Name != "inside" || result[1].Symbol.Name != "atEnd" {
			t.Errorf("SubroutinesInRange returned wrong symbols: %v", result)
		}
	})

	t.Run("only subroutine/entry types are included", func(t *testing.T) {
		table := NewTable()
		table.Add(0x0800, Symbol{Name: "sub", Type: SymbolSubroutine, Source: SourceUser})
		table.Add(0x0810, Symbol{Name: "entry", Type: SymbolEntry, Source: SourceUser})
		table.Add(0x0820, Symbol{Name: "label", Type: SymbolLabel, Source: SourceUser})
		table.Add(0x0830, Symbol{Name: "byte", Type: SymbolByte, Source: SourceUser})
		table.Add(0x0840, Symbol{Name: "word", Type: SymbolWord, Source: SourceUser})

		result := table.SubroutinesInRange(0x0800, 0x0900)
		if len(result) != 2 {
			t.Fatalf("SubroutinesInRange returned %d symbols, want 2", len(result))
		}
		if result[0].Symbol.Name != "sub" || result[1].Symbol.Name != "entry" {
			t.Errorf("SubroutinesInRange returned wrong symbols: %v", result)
		}
	})

	t.Run("results are sorted by address", func(t *testing.T) {
		table := NewTable()
		// Add in non-sorted order
		table.Add(0x0850, Symbol{Name: "third", Type: SymbolSubroutine, Source: SourceUser})
		table.Add(0x0810, Symbol{Name: "first", Type: SymbolEntry, Source: SourceUser})
		table.Add(0x0830, Symbol{Name: "second", Type: SymbolSubroutine, Source: SourceUser})

		result := table.SubroutinesInRange(0x0800, 0x0900)
		if len(result) != 3 {
			t.Fatalf("SubroutinesInRange returned %d symbols, want 3", len(result))
		}
		if result[0].Addr != 0x0810 || result[1].Addr != 0x0830 || result[2].Addr != 0x0850 {
			t.Errorf("SubroutinesInRange not sorted by address: %v", result)
		}
	})
}

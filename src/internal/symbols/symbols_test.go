package symbols

import (
	"testing"
)

func TestSymbolTable(t *testing.T) {
	table := NewTable()

	// Test empty table
	_, ok := table.At(0x0800)
	if ok {
		t.Error("At(0x0800) on empty table should return false")
	}

	// Add symbol
	sym1 := Symbol{Name: "start", Type: SymbolEntry, Source: SourceUser}
	table.Add(0x0800, sym1)

	sym, ok := table.At(0x0800)
	if !ok {
		t.Fatal("At(0x0800) should return true after adding symbol")
	}
	if sym.Name != "start" {
		t.Errorf("At(0x0800).Name = %q, want %q", sym.Name, "start")
	}

	// Adding lower-priority symbol at same address should NOT replace
	sym2 := Symbol{Name: "L_0800", Type: SymbolLabel, Source: SourceAuto}
	table.Add(0x0800, sym2)

	sym, ok = table.At(0x0800)
	if !ok {
		t.Fatal("At(0x0800) should still return true")
	}
	if sym.Name != "start" {
		t.Errorf("At(0x0800).Name = %q, want %q (user symbol should not be replaced by auto)", sym.Name, "start")
	}

	// Remove symbol
	table.Remove(0x0800, "start")
	_, ok = table.At(0x0800)
	if ok {
		t.Error("At(0x0800) after remove should return false")
	}
}

func TestSymbolTableRemoveNonexistent(t *testing.T) {
	table := NewTable()

	// Removing from empty table should not panic
	table.Remove(0x0800, "nonexistent")

	// Add and remove different name should not affect existing
	table.Add(0x0800, Symbol{Name: "test", Type: SymbolLabel, Source: SourceUser})
	table.Remove(0x0800, "other")

	sym, ok := table.At(0x0800)
	if !ok {
		t.Error("At(0x0800) should return true")
	}
	if sym.Name != "test" {
		t.Errorf("At(0x0800).Name = %q, want %q", sym.Name, "test")
	}
}

func TestSymbolTablePriority(t *testing.T) {
	t.Run("user replaces auto", func(t *testing.T) {
		table := NewTable()

		// Add auto symbol first
		table.Add(0x0800, Symbol{Name: "L_0800", Type: SymbolLabel, Source: SourceAuto})

		// User symbol should replace it
		table.Add(0x0800, Symbol{Name: "main", Type: SymbolEntry, Source: SourceUser})

		sym, _ := table.At(0x0800)
		if sym.Name != "main" {
			t.Errorf("At(0x0800).Name = %q, want %q (user should replace auto)", sym.Name, "main")
		}
	})

	t.Run("auto does not replace user", func(t *testing.T) {
		table := NewTable()

		// Add user symbol first
		table.Add(0x0800, Symbol{Name: "main", Type: SymbolEntry, Source: SourceUser})

		// Auto symbol should NOT replace it
		table.Add(0x0800, Symbol{Name: "L_0800", Type: SymbolLabel, Source: SourceAuto})

		sym, _ := table.At(0x0800)
		if sym.Name != "main" {
			t.Errorf("At(0x0800).Name = %q, want %q (auto should not replace user)", sym.Name, "main")
		}
	})

	t.Run("assistant replaces auto", func(t *testing.T) {
		table := NewTable()

		table.Add(0x0800, Symbol{Name: "L_0800", Type: SymbolLabel, Source: SourceAuto})
		table.Add(0x0800, Symbol{Name: "loop", Type: SymbolLabel, Source: SourceAssistant})

		sym, _ := table.At(0x0800)
		if sym.Name != "loop" {
			t.Errorf("At(0x0800).Name = %q, want %q", sym.Name, "loop")
		}
	})

	t.Run("user replaces assistant", func(t *testing.T) {
		table := NewTable()

		table.Add(0x0800, Symbol{Name: "loop", Type: SymbolLabel, Source: SourceAssistant})
		table.Add(0x0800, Symbol{Name: "main_loop", Type: SymbolLabel, Source: SourceUser})

		sym, _ := table.At(0x0800)
		if sym.Name != "main_loop" {
			t.Errorf("At(0x0800).Name = %q, want %q", sym.Name, "main_loop")
		}
	})

	t.Run("same priority replaces", func(t *testing.T) {
		table := NewTable()

		table.Add(0x0800, Symbol{Name: "first", Type: SymbolLabel, Source: SourceAuto})
		table.Add(0x0800, Symbol{Name: "second", Type: SymbolLabel, Source: SourceAuto})

		sym, _ := table.At(0x0800)
		if sym.Name != "second" {
			t.Errorf("At(0x0800).Name = %q, want %q (same priority should replace)", sym.Name, "second")
		}
	})
}

func TestSymbolWordExpansion(t *testing.T) {
	table := NewTable()

	// Add a word symbol
	table.Add(0x0900, Symbol{Name: "PTR", Type: SymbolWord, Source: SourceUser})

	// Should create PTR_LO at 0x0900
	symLo, ok := table.At(0x0900)
	if !ok {
		t.Fatal("At(0x0900) should return true after adding word symbol")
	}
	if symLo.Name != "PTR_LO" {
		t.Errorf("At(0x0900).Name = %q, want %q", symLo.Name, "PTR_LO")
	}
	if symLo.Type != SymbolByte {
		t.Errorf("At(0x0900).Type = %q, want %q", symLo.Type, SymbolByte)
	}

	// Should create PTR_HI at 0x0901
	symHi, ok := table.At(0x0901)
	if !ok {
		t.Fatal("At(0x0901) should return true after adding word symbol")
	}
	if symHi.Name != "PTR_HI" {
		t.Errorf("At(0x0901).Name = %q, want %q", symHi.Name, "PTR_HI")
	}
	if symHi.Type != SymbolByte {
		t.Errorf("At(0x0901).Type = %q, want %q", symHi.Type, SymbolByte)
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
		// Word expands to _LO and _HI bytes, so we test with a regular byte instead
		table.Add(0x0840, Symbol{Name: "data", Type: SymbolByte, Source: SourceUser})

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

func TestAllReturnsMap(t *testing.T) {
	table := NewTable()
	table.Add(0x0800, Symbol{Name: "a", Type: SymbolLabel, Source: SourceUser})
	table.Add(0x0900, Symbol{Name: "b", Type: SymbolSubroutine, Source: SourceUser})

	all := table.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d entries, want 2", len(all))
	}
	if all[0x0800].Name != "a" {
		t.Errorf("All()[0x0800].Name = %q, want %q", all[0x0800].Name, "a")
	}
	if all[0x0900].Name != "b" {
		t.Errorf("All()[0x0900].Name = %q, want %q", all[0x0900].Name, "b")
	}
}

func TestAtOfTypes(t *testing.T) {
	table := NewTable()
	if err := table.Add(0x0800, Symbol{Name: "main", Type: SymbolLabel, Source: SourceUser}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	t.Run("match by allowed type", func(t *testing.T) {
		sym, ok := table.AtOfTypes(0x0800, SymbolLabel, SymbolEntry)
		if !ok {
			t.Fatal("AtOfTypes should return true for allowed symbol type")
		}
		if sym.Name != "main" {
			t.Fatalf("AtOfTypes name = %q, want %q", sym.Name, "main")
		}
	})

	t.Run("reject disallowed type", func(t *testing.T) {
		_, ok := table.AtOfTypes(0x0800, SymbolSubroutine)
		if ok {
			t.Fatal("AtOfTypes should return false for disallowed symbol type")
		}
	})

	t.Run("empty allowed behaves like At", func(t *testing.T) {
		sym, ok := table.AtOfTypes(0x0800)
		if !ok {
			t.Fatal("AtOfTypes with empty allowed should return existing symbol")
		}
		if sym.Name != "main" {
			t.Fatalf("AtOfTypes name = %q, want %q", sym.Name, "main")
		}
	})

	t.Run("missing address returns false", func(t *testing.T) {
		_, ok := table.AtOfTypes(0x0810, SymbolLabel)
		if ok {
			t.Fatal("AtOfTypes should return false for missing address")
		}
	})
}

package symbols

import (
	"fmt"
	"sort"
)

type SymbolType string

const (
	SymbolSubroutine SymbolType = "subroutine"
	SymbolLabel      SymbolType = "label"
	SymbolByte       SymbolType = "byte"
	SymbolWord       SymbolType = "word"
	SymbolEntry      SymbolType = "entry"
)

type SymbolSource string

const (
	SourceUser      SymbolSource = "user"
	SourceAssistant SymbolSource = "assistant"
	SourceAuto      SymbolSource = "auto"
)

type Symbol struct {
	Name   string
	Type   SymbolType
	Source SymbolSource
}

type Table struct {
	symbols map[uint16]Symbol
}

func NewTable() *Table {
	return &Table{
		symbols: make(map[uint16]Symbol),
	}
}

// At returns the symbol at the given address, or empty Symbol if none.
func (t *Table) At(addr uint16) (Symbol, bool) {
	sym, ok := t.symbols[addr]
	return sym, ok
}

// AtOfTypes returns the symbol at addr only if its type is in allowed.
// If allowed is empty, it behaves like At.
func (t *Table) AtOfTypes(addr uint16, allowed ...SymbolType) (Symbol, bool) {
	sym, ok := t.At(addr)
	if !ok {
		return Symbol{}, false
	}
	if len(allowed) == 0 {
		return sym, true
	}
	for _, typ := range allowed {
		if sym.Type == typ {
			return sym, true
		}
	}
	return Symbol{}, false
}

// sourcePriority returns the priority of a symbol source (higher is better).
func sourcePriority(source SymbolSource) int {
	switch source {
	case SourceUser:
		return 3
	case SourceAssistant:
		return 2
	default: // SourceAuto, SourceC64ROM, etc.
		return 1
	}
}

// shouldReplace returns true if the new symbol should replace the existing one.
func shouldReplace(existing, new Symbol) bool {
	return sourcePriority(new.Source) >= sourcePriority(existing.Source)
}

// Add adds a symbol at the given address.
// For SymbolWord, expands to _LO and _HI byte symbols.
// Uses priority: user > assistant > auto. Only replaces if new has equal or higher priority.
// Returns an error if a symbol with the same name already exists at a different address.
func (t *Table) Add(addr uint16, sym Symbol) error {
	// Handle SymbolWord: expand to _LO and _HI
	if sym.Type == SymbolWord {
		if err := t.Add(addr, Symbol{Name: sym.Name + "_LO", Type: SymbolByte, Source: sym.Source}); err != nil {
			return err
		}
		return t.Add(addr+1, Symbol{Name: sym.Name + "_HI", Type: SymbolByte, Source: sym.Source})
	}

	// Check for duplicate name at a different address
	for existingAddr, existingSym := range t.symbols {
		if existingSym.Name == sym.Name && existingAddr != addr {
			return fmt.Errorf("symbol name %q already exists at $%04X", sym.Name, existingAddr)
		}
	}

	// Only replace if new symbol has equal or higher priority
	if existing, ok := t.symbols[addr]; ok {
		if !shouldReplace(existing, sym) {
			return nil
		}
	}
	t.symbols[addr] = sym
	return nil
}

// Remove removes the symbol at the given address if it matches the name.
func (t *Table) Remove(addr uint16, name string) {
	if sym, ok := t.symbols[addr]; ok && sym.Name == name {
		delete(t.symbols, addr)
	}
}

// All returns all symbols as a map from address to symbol.
func (t *Table) All() map[uint16]Symbol {
	return t.symbols
}

// RemoveBySource removes all symbols generated from the given source.
func (t *Table) RemoveBySource(source SymbolSource) {
	for addr, sym := range t.symbols {
		if sym.Source == source {
			delete(t.symbols, addr)
		}
	}
}

// AddressedSymbol pairs a symbol with its address.
type AddressedSymbol struct {
	Addr   uint16
	Symbol Symbol
}

// SubroutinesInRange returns all subroutine/entry symbols within [start, end], sorted by address.
func (t *Table) SubroutinesInRange(start, end uint16) []AddressedSymbol {
	var result []AddressedSymbol

	for addr, sym := range t.symbols {
		if addr < start || addr > end {
			continue
		}
		if sym.Type == SymbolSubroutine || sym.Type == SymbolEntry {
			result = append(result, AddressedSymbol{Addr: addr, Symbol: sym})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Addr < result[j].Addr
	})

	return result
}

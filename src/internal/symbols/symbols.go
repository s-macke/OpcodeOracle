package symbols

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
	SourceUser   SymbolSource = "user"
	SourceAuto   SymbolSource = "auto"
	SourceC64ROM SymbolSource = "c64rom"
	SourceImport SymbolSource = "import"
)

type Symbol struct {
	Name   string
	Type   SymbolType
	Source SymbolSource
}

type Table struct {
	symbols map[uint16][]Symbol
}

func NewTable() *Table {
	return &Table{
		symbols: make(map[uint16][]Symbol),
	}
}

// At returns all symbols at the given address.
func (t *Table) At(addr uint16) []Symbol {
	if syms, ok := t.symbols[addr]; ok {
		return syms
	}
	return []Symbol{}
}

// Add adds a symbol at the given address if not already present.
func (t *Table) Add(addr uint16, sym Symbol) {
	for _, existing := range t.symbols[addr] {
		if existing == sym {
			return
		}
	}
	t.symbols[addr] = append(t.symbols[addr], sym)
}

// Remove removes a symbol by name at the given address.
func (t *Table) Remove(addr uint16, name string) {
	syms := t.symbols[addr]
	for i, sym := range syms {
		if sym.Name == name {
			t.symbols[addr] = append(syms[:i], syms[i+1:]...)
			if len(t.symbols[addr]) == 0 {
				delete(t.symbols, addr)
			}
			return
		}
	}
}

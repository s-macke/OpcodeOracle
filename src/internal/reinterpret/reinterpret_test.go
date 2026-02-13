package reinterpret

import (
	"testing"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

func TestAsCodeCreatesSeedAndReanalyzes(t *testing.T) {
	// 0800: JSR $0804 ; RTS ; NOP ; RTS
	s := state.NewState([]byte{0x20, 0x04, 0x08, 0x60, 0xEA, 0x60}, 0x0800, []uint16{0x0800}, "test.prg")

	analyzer, err := AsCode(s, 0x0804, regions.RegionSourceUser)
	if err != nil {
		t.Fatalf("AsCode error = %v, want nil", err)
	}
	if analyzer == nil {
		t.Fatal("analyzer = nil, want non-nil")
	}
	sym, ok := s.Symbols.At(0x0804)
	if !ok {
		t.Fatal("missing symbol at 0x0804")
	}
	if sym.Type != symbols.SymbolSubroutine && sym.Type != symbols.SymbolLabel && sym.Type != symbols.SymbolEntry {
		t.Fatalf("symbol type = %s, want code-seeding type", sym.Type)
	}
}

func TestAsDataAddsHardLock(t *testing.T) {
	// 0800: JMP $0803 ; 0803: RTS
	s := state.NewState([]byte{0x4C, 0x03, 0x08, 0x60}, 0x0800, []uint16{0x0800}, "test.prg")

	analyzer, err := AsData(s, 0x0803, 0x0803, regions.RegionSourceAssistant)
	if err != nil {
		t.Fatalf("AsData error = %v, want nil", err)
	}
	if analyzer.IsInstructionAt(0x0803) {
		t.Fatal("forced-data address 0x0803 should not decode as code")
	}
}

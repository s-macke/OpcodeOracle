package analysis

import (
	"testing"

	"opcodeoracle/internal/asm"
	"opcodeoracle/internal/binary"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
	"opcodeoracle/internal/xrefs"
)

// Helper to create a minimal state for testing
func newTestState(data []byte, origin uint16, entryPoints []uint16) *state.State {
	// Create a full 64K region table initialized as data
	rt := regions.NewTable()
	rt.SetRegions([]regions.Region{
		{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData},
	})

	return &state.State{
		Binary: binary.Binary{
			Data:   data,
			Origin: origin,
		},
		EntryPoints: entryPoints,
		Symbols:     symbols.NewTable(),
		Regions:     rt,
		XRefs:       xrefs.NewTable(),
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		opcode   byte
		expected InstructionClass
	}{
		// Sequential
		{0xA9, ClassSequential}, // LDA #imm
		{0xEA, ClassSequential}, // NOP
		{0x8D, ClassSequential}, // STA abs
		{0xE8, ClassSequential}, // INX
		{0x48, ClassSequential}, // PHA

		// Jump
		{0x4C, ClassJump}, // JMP abs
		{0x6C, ClassJump}, // JMP (ind)

		// Branch
		{0x90, ClassBranch}, // BCC
		{0xB0, ClassBranch}, // BCS
		{0xF0, ClassBranch}, // BEQ
		{0x30, ClassBranch}, // BMI
		{0xD0, ClassBranch}, // BNE
		{0x10, ClassBranch}, // BPL
		{0x50, ClassBranch}, // BVC
		{0x70, ClassBranch}, // BVS

		// Call
		{0x20, ClassCall}, // JSR

		// Return
		{0x60, ClassReturn}, // RTS
		{0x40, ClassReturn}, // RTI

		// Terminal
		{0x00, ClassTerminal}, // BRK

		// Illegal
		{0x02, ClassIllegal},
		{0x03, ClassIllegal},
		{0xFF, ClassIllegal},
	}

	for _, tc := range tests {
		def := asm.Opcodes[tc.opcode]
		got := classify(def)
		if got != tc.expected {
			t.Errorf("classify(0x%02X/%s): got %d, want %d",
				tc.opcode, def.Op.String(), got, tc.expected)
		}
	}
}

func TestCalculateBranchTarget(t *testing.T) {
	tests := []struct {
		pc       uint16
		operand  byte
		expected uint16
	}{
		// Forward branches (positive offset)
		{0x1000, 0x00, 0x1002}, // BEQ *+2 (no skip)
		{0x1000, 0x10, 0x1012}, // BEQ *+18 (skip 16 bytes after instruction)
		{0x1000, 0x7F, 0x1081}, // BEQ *+129 (max positive)

		// Backward branches (negative offset)
		{0x1000, 0xFF, 0x1001}, // BEQ *-1 (offset -1)
		{0x1000, 0xFE, 0x1000}, // BEQ *-2 (branch to self instruction)
		{0x1000, 0x80, 0x0F82}, // BEQ *-126 (max negative)

		// Edge cases
		{0x0002, 0xFC, 0x0000}, // Branch to zero page
		{0xFFFD, 0x01, 0x0000}, // Wrap around (overflow)
	}

	for _, tc := range tests {
		got := calculateBranchTarget(tc.pc, tc.operand)
		if got != tc.expected {
			t.Errorf("calculateBranchTarget(0x%04X, 0x%02X): got 0x%04X, want 0x%04X",
				tc.pc, tc.operand, got, tc.expected)
		}
	}
}

func TestAnalyzeLinearCode(t *testing.T) {
	// Simple linear program: LDA #$42, NOP, RTS
	// Origin at $1000
	data := []byte{
		0xA9, 0x42, // LDA #$42
		0xEA, // NOP
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify entry point symbol created
	if _, ok := s.Symbols.At(0x1000); !ok {
		t.Error("Expected symbol at entry point 0x1000")
	}

	// Verify all addresses were visited
	for addr := uint16(0x1000); addr <= 0x1003; addr++ {
		if !analyzer.visited[addr] {
			// Note: visited is per-instruction start, not per-byte
		}
	}

	// Check that regions are marked as code
	r := s.Regions.RegionAt(0x1000)
	if r == nil || r.Type != regions.RegionCode {
		t.Error("Expected 0x1000 to be marked as code")
	}
}

func TestAnalyzeConditionalBranch(t *testing.T) {
	// Program with branch: LDA #$00, BEQ skip, INX, skip: RTS
	// At origin $1000:
	// 1000: A9 00    LDA #$00
	// 1002: F0 01    BEQ $1005 (skip INX)
	// 1004: E8       INX
	// 1005: 60       RTS
	data := []byte{
		0xA9, 0x00, // LDA #$00
		0xF0, 0x01, // BEQ +1 (to $1005)
		0xE8, // INX
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify both paths discovered
	if !analyzer.visited[0x1004] {
		t.Error("Fall-through path (INX at 0x1004) not discovered")
	}
	if !analyzer.visited[0x1005] {
		t.Error("Branch target (RTS at 0x1005) not discovered")
	}

	// Verify branch xref created
	xrefs := s.XRefs.From(0x1002)
	if len(xrefs) == 0 {
		t.Error("Expected xref from branch at 0x1002")
	} else if xrefs[0].To != 0x1005 {
		t.Errorf("Expected xref to 0x1005, got 0x%04X", xrefs[0].To)
	}

	// Verify label created at branch target
	sym, ok := s.Symbols.At(0x1005)
	if !ok || sym.Type != symbols.SymbolLabel {
		t.Error("Expected label at branch target 0x1005")
	}
}

func TestAnalyzeJSR(t *testing.T) {
	// Program: JSR $1010, RTS, ... subroutine at $1010: INX, RTS
	// At origin $1000:
	// 1000: 20 10 10  JSR $1010
	// 1003: 60        RTS
	// ... padding ...
	// 1010: E8        INX
	// 1011: 60        RTS
	data := make([]byte, 0x12) // 18 bytes
	data[0x00] = 0x20          // JSR
	data[0x01] = 0x10          // low byte of $1010
	data[0x02] = 0x10          // high byte of $1010
	data[0x03] = 0x60          // RTS
	data[0x10] = 0xE8          // INX (at $1010)
	data[0x11] = 0x60          // RTS

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify subroutine discovered
	if !analyzer.visited[0x1010] {
		t.Error("Subroutine at 0x1010 not discovered")
	}

	// Verify return address path discovered
	if !analyzer.visited[0x1003] {
		t.Error("Return address at 0x1003 not discovered")
	}

	// Verify call xref created
	xrefs := s.XRefs.From(0x1000)
	if len(xrefs) == 0 {
		t.Error("Expected xref from JSR at 0x1000")
	} else if xrefs[0].Type != "call" {
		t.Errorf("Expected call xref, got %s", xrefs[0].Type)
	}

	// Verify subroutine symbol created
	sym, ok := s.Symbols.At(0x1010)
	if !ok || sym.Type != symbols.SymbolSubroutine {
		t.Error("Expected subroutine symbol at 0x1010")
	} else if sym.Name != "SUB_1010" {
		t.Errorf("Expected subroutine name SUB_1010, got %s", sym.Name)
	}
}

func TestAnalyzeJMPAbsolute(t *testing.T) {
	// Program: JMP $1010, ... target at $1010: NOP, RTS
	// 1000: 4C 10 10  JMP $1010
	// ... padding ...
	// 1010: EA        NOP
	// 1011: 60        RTS
	data := make([]byte, 0x12)
	data[0x00] = 0x4C // JMP abs
	data[0x01] = 0x10 // low byte of $1010
	data[0x02] = 0x10 // high byte of $1010
	data[0x10] = 0xEA // NOP
	data[0x11] = 0x60 // RTS

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify jump target discovered
	if !analyzer.visited[0x1010] {
		t.Error("Jump target at 0x1010 not discovered")
	}

	// Code after JMP should NOT be visited (unreachable)
	if analyzer.visited[0x1003] {
		t.Error("Code after JMP should not be visited")
	}

	// Verify jump xref and label
	xrefs := s.XRefs.From(0x1000)
	if len(xrefs) == 0 {
		t.Error("Expected xref from JMP")
	}

	sym, ok := s.Symbols.At(0x1010)
	if !ok || sym.Type != symbols.SymbolLabel || sym.Name != "L_1010" {
		t.Error("Expected label L_1010 at jump target")
	}
}

func TestAnalyzeJMPIndirect(t *testing.T) {
	// Indirect JMP cannot be followed statically
	// 1000: 6C 10 10  JMP ($1010)
	data := make([]byte, 0x12)
	data[0x00] = 0x6C // JMP (ind)
	data[0x01] = 0x10 // low byte
	data[0x02] = 0x10 // high byte

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Instruction should be marked as visited/code
	if !analyzer.visited[0x1000] {
		t.Error("JMP indirect should be visited")
	}

	// No xrefs should be created (can't know target statically)
	xrefs := s.XRefs.From(0x1000)
	if len(xrefs) != 0 {
		t.Error("Indirect JMP should not create xrefs")
	}
}

func TestAnalyzeIllegalOpcode(t *testing.T) {
	// Program starting with illegal opcode
	data := []byte{
		0x02, // Illegal opcode
		0x60, // RTS (should not be reached)
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	// Should not return error (graceful handling)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Illegal opcode should not be visited (not marked as code)
	if analyzer.visited[0x1000] {
		t.Error("Illegal opcode should not be marked as visited")
	}

	// Following code should not be discovered
	if analyzer.visited[0x1001] {
		t.Error("Code after illegal opcode should not be discovered")
	}
}

func TestAnalyzeOutOfBounds(t *testing.T) {
	// JMP to address outside binary
	// 1000: 4C 00 20  JMP $2000 (out of bounds)
	data := []byte{
		0x4C, 0x00, 0x20, // JMP $2000
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	// Should not panic or return error
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// JMP instruction should still be visited
	if !analyzer.visited[0x1000] {
		t.Error("JMP instruction should be visited")
	}

	// Out of bounds target should not be visited
	if analyzer.visited[0x2000] {
		t.Error("Out of bounds address should not be visited")
	}
}

func TestAnalyzeMultipleEntryPoints(t *testing.T) {
	// Two separate entry points
	// 1000: E8       INX
	// 1001: 60       RTS
	// 1002: CA       DEX
	// 1003: 60       RTS
	data := []byte{
		0xE8, // INX at $1000
		0x60, // RTS
		0xCA, // DEX at $1002
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000, 0x1002})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Both entry points should be discovered
	if !analyzer.visited[0x1000] {
		t.Error("First entry point not discovered")
	}
	if !analyzer.visited[0x1002] {
		t.Error("Second entry point not discovered")
	}

	// Both should have entry symbols
	sym1, ok1 := s.Symbols.At(0x1000)
	sym2, ok2 := s.Symbols.At(0x1002)

	if !ok1 || sym1.Type != symbols.SymbolEntry {
		t.Error("Expected entry symbol at 0x1000")
	}
	if !ok2 || sym2.Type != symbols.SymbolEntry {
		t.Error("Expected entry symbol at 0x1002")
	}
}

func TestAnalyzeBackwardBranch(t *testing.T) {
	// Loop: INX, BNE loop
	// 1000: E8       INX
	// 1001: D0 FD    BNE $1000 (branch back -3)
	// 1003: 60       RTS
	data := []byte{
		0xE8,       // INX
		0xD0, 0xFD, // BNE -3 (to $1000)
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Loop should be discovered without infinite loop
	if !analyzer.visited[0x1000] {
		t.Error("Loop start not discovered")
	}
	if !analyzer.visited[0x1001] {
		t.Error("Branch instruction not discovered")
	}
	if !analyzer.visited[0x1003] {
		t.Error("Fall-through after loop not discovered")
	}

	// Label at loop start (note: entry symbol may have replaced label due to priority)
	sym, ok := s.Symbols.At(0x1000)
	if !ok {
		t.Error("Expected symbol at loop target 0x1000")
	} else if sym.Type != symbols.SymbolLabel && sym.Type != symbols.SymbolEntry {
		t.Errorf("Expected label or entry at loop target 0x1000, got %v", sym.Type)
	}
}

func TestAnalyzeFromAddress(t *testing.T) {
	// Test AnalyzeFrom with a single address
	// 1000: E8       INX
	// 1001: E8       INX
	// 1002: 60       RTS
	// 1010: CA       DEX (separate, unreachable code)
	// 1011: 60       RTS
	data := make([]byte, 0x12)
	data[0x00] = 0xE8 // INX at $1000
	data[0x01] = 0xE8 // INX
	data[0x02] = 0x60 // RTS
	data[0x10] = 0xCA // DEX at $1010
	data[0x11] = 0x60 // RTS

	// No entry points, no symbols
	s := newTestState(data, 0x1000, []uint16{})
	analyzer := NewAnalyzer(s, UpdateAll)

	// Analyze from $1000 only
	if err := analyzer.AnalyzeFrom(0x1000); err != nil {
		t.Fatalf("AnalyzeFrom failed: %v", err)
	}

	// Code at $1000-$1002 should be discovered
	if !analyzer.visited[0x1000] {
		t.Error("Address 0x1000 not discovered")
	}
	if !analyzer.visited[0x1002] {
		t.Error("Address 0x1002 not discovered")
	}

	// Code at $1010 should NOT be discovered (not reachable)
	if analyzer.visited[0x1010] {
		t.Error("Address 0x1010 should not be discovered")
	}

	// No entry symbols should be created (AnalyzeFrom doesn't add symbols)
	if sym, ok := s.Symbols.At(0x1000); ok && sym.Type == symbols.SymbolEntry {
		t.Error("AnalyzeFrom should not create entry symbols")
	}
}

func TestAnalyzeFromMultipleCalls(t *testing.T) {
	// Test calling AnalyzeFrom multiple times
	data := make([]byte, 0x12)
	data[0x00] = 0xE8 // INX at $1000
	data[0x01] = 0x60 // RTS
	data[0x10] = 0xCA // DEX at $1010
	data[0x11] = 0x60 // RTS

	s := newTestState(data, 0x1000, []uint16{})
	analyzer := NewAnalyzer(s, UpdateAll)

	// First analysis
	if err := analyzer.AnalyzeFrom(0x1000); err != nil {
		t.Fatalf("First AnalyzeFrom failed: %v", err)
	}

	// Second analysis from different address
	if err := analyzer.AnalyzeFrom(0x1010); err != nil {
		t.Fatalf("Second AnalyzeFrom failed: %v", err)
	}

	// Both regions should be discovered
	if !analyzer.visited[0x1000] {
		t.Error("Address 0x1000 not discovered")
	}
	if !analyzer.visited[0x1010] {
		t.Error("Address 0x1010 not discovered")
	}
}

func TestAnalyzeFromExistingSymbols(t *testing.T) {
	// Code at $1000 is unreachable from entry point, but has a subroutine symbol
	// 1000: E8       INX (not reachable from entry)
	// 1001: 60       RTS
	// 1010: 60       RTS (entry point)
	data := make([]byte, 0x12)
	data[0x00] = 0xE8 // INX at $1000
	data[0x01] = 0x60 // RTS
	data[0x10] = 0x60 // RTS at $1010 (entry point)

	s := newTestState(data, 0x1000, []uint16{0x1010}) // Entry only at $1010

	// Pre-populate symbol table with a subroutine at $1000
	s.Symbols.Add(0x1000, symbols.Symbol{
		Name:   "user_sub",
		Type:   symbols.SymbolSubroutine,
		Source: symbols.SourceUser,
	})

	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Entry point should be discovered
	if !analyzer.visited[0x1010] {
		t.Error("Entry point at 0x1010 not discovered")
	}

	// Subroutine from symbol table should also be discovered
	if !analyzer.visited[0x1000] {
		t.Error("Subroutine at 0x1000 (from symbol table) not discovered")
	}

	// Code following subroutine should be discovered
	if !analyzer.visited[0x1001] {
		t.Error("Code at 0x1001 not discovered")
	}
}

func TestAnalyzeFromExistingLabels(t *testing.T) {
	// Label symbol should also trigger analysis
	// 1000: EA       NOP
	// 1001: 60       RTS
	data := []byte{
		0xEA, // NOP
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{}) // No entry points!

	// Pre-populate with a label
	s.Symbols.Add(0x1000, symbols.Symbol{
		Name:   "start_label",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})

	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Label should trigger analysis
	if !analyzer.visited[0x1000] {
		t.Error("Label at 0x1000 not discovered")
	}
	if !analyzer.visited[0x1001] {
		t.Error("Code at 0x1001 not discovered")
	}
}

func TestInBounds(t *testing.T) {
	// Binary at $1000-$10FF (256 bytes)
	data := make([]byte, 256)
	s := newTestState(data, 0x1000, nil)
	analyzer := NewAnalyzer(s, UpdateAll)

	tests := []struct {
		addr     uint16
		expected bool
	}{
		{0x0FFF, false}, // Before binary
		{0x1000, true},  // Start of binary
		{0x1080, true},  // Middle of binary
		{0x10FF, true},  // End of binary
		{0x1100, false}, // After binary
		{0x0000, false}, // Way before
		{0xFFFF, false}, // Way after
	}

	for _, tc := range tests {
		got := analyzer.inBounds(tc.addr)
		if got != tc.expected {
			t.Errorf("inBounds(0x%04X): got %v, want %v", tc.addr, got, tc.expected)
		}
	}
}

func TestUpdateFlagsXRefsOnly(t *testing.T) {
	// Test that UpdateXRefsOnly only updates XRefs, not regions or symbols
	// Program: JSR $1010, RTS, ... subroutine at $1010: RTS
	data := make([]byte, 0x12)
	data[0x00] = 0x20 // JSR
	data[0x01] = 0x10 // low byte of $1010
	data[0x02] = 0x10 // high byte of $1010
	data[0x03] = 0x60 // RTS
	data[0x10] = 0x60 // RTS (at $1010)

	s := newTestState(data, 0x1000, []uint16{0x1000})

	// Reset regions to data (newTestState sets up data regions)
	s.Regions.SetRegions([]regions.Region{
		{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData},
	})

	// Clear symbols
	s.Symbols = symbols.NewTable()

	analyzer := NewAnalyzer(s, UpdateXRefsOnly)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// XRefs should be created
	xrefs := s.XRefs.From(0x1000)
	if len(xrefs) == 0 {
		t.Error("Expected xref from JSR at 0x1000")
	}

	// Regions should NOT be updated (still data)
	r := s.Regions.RegionAt(0x1000)
	if r != nil && r.Type == regions.RegionCode {
		t.Error("Regions should not be updated with UpdateXRefsOnly flag")
	}

	// Symbols should NOT be created
	if _, ok := s.Symbols.At(0x1000); ok {
		t.Error("Symbols should not be created with UpdateXRefsOnly flag")
	}

	if _, ok := s.Symbols.At(0x1010); ok {
		t.Error("Subroutine symbol should not be created with UpdateXRefsOnly flag")
	}
}

func TestUpdateFlagsNoSymbols(t *testing.T) {
	// Test that UpdateRegions | UpdateXRefs skips symbol creation
	data := []byte{
		0x20, 0x05, 0x10, // JSR $1005
		0x60, // RTS
		0x60, // RTS (subroutine at $1005)
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	s.Symbols = symbols.NewTable()

	analyzer := NewAnalyzer(s, UpdateRegions|UpdateXRefs)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Regions should be marked as code
	r := s.Regions.RegionAt(0x1000)
	if r == nil || r.Type != regions.RegionCode {
		t.Error("Expected 0x1000 to be marked as code")
	}

	// XRefs should be created
	xrefs := s.XRefs.From(0x1000)
	if len(xrefs) == 0 {
		t.Error("Expected xref from JSR")
	}

	// No symbols should be created
	if _, ok := s.Symbols.At(0x1000); ok {
		t.Error("Entry symbol should not be created")
	}
	if _, ok := s.Symbols.At(0x1005); ok {
		t.Error("Subroutine symbol should not be created")
	}
}

func TestUpdateFlagsNoXRefs(t *testing.T) {
	// Test that UpdateRegions | UpdateSymbols skips xref creation
	data := []byte{
		0x20, 0x05, 0x10, // JSR $1005
		0x60, // RTS
		0x60, // RTS (subroutine at $1005)
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})

	analyzer := NewAnalyzer(s, UpdateRegions|UpdateSymbols)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Regions should be marked as code
	r := s.Regions.RegionAt(0x1000)
	if r == nil || r.Type != regions.RegionCode {
		t.Error("Expected 0x1000 to be marked as code")
	}

	// Symbols should be created
	if _, ok := s.Symbols.At(0x1000); !ok {
		t.Error("Entry symbol should be created")
	}
	if _, ok := s.Symbols.At(0x1005); !ok {
		t.Error("Subroutine symbol should be created")
	}

	// XRefs should NOT be created
	xrefs := s.XRefs.From(0x1000)
	if len(xrefs) != 0 {
		t.Error("XRefs should not be created with UpdateRegions|UpdateSymbols")
	}
}

func TestUpdateFlagsNoRegions(t *testing.T) {
	// Test that UpdateSymbols | UpdateXRefs skips region marking
	data := []byte{
		0xA9, 0x42, // LDA #$42
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})

	// Reset regions to data
	s.Regions.SetRegions([]regions.Region{
		{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData},
	})

	analyzer := NewAnalyzer(s, UpdateSymbols|UpdateXRefs)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Symbols should be created
	if _, ok := s.Symbols.At(0x1000); !ok {
		t.Error("Entry symbol should be created")
	}

	// Regions should NOT be marked as code (still data)
	r := s.Regions.RegionAt(0x1000)
	if r != nil && r.Type == regions.RegionCode {
		t.Error("Regions should not be updated with UpdateSymbols|UpdateXRefs")
	}
}

func TestIsInstructionAt(t *testing.T) {
	// Program: LDA #$42, NOP, RTS
	// Instruction boundaries at: $1000 (LDA), $1002 (NOP), $1003 (RTS)
	// Non-boundaries at: $1001 (operand of LDA)
	data := []byte{
		0xA9, 0x42, // LDA #$42
		0xEA, // NOP
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Test instruction boundaries
	tests := []struct {
		addr     uint16
		expected bool
	}{
		{0x1000, true},  // LDA instruction start
		{0x1001, false}, // Operand of LDA, not an instruction start
		{0x1002, true},  // NOP instruction start
		{0x1003, true},  // RTS instruction start
		{0x1004, false}, // Beyond code
		{0x0FFF, false}, // Before code
	}

	for _, tc := range tests {
		got := analyzer.IsInstructionAt(tc.addr)
		if got != tc.expected {
			t.Errorf("IsInstructionAt(0x%04X): got %v, want %v", tc.addr, got, tc.expected)
		}
	}
}

func TestInstructionAddresses(t *testing.T) {
	// Program with jumps to test non-sequential discovery
	// 1000: 4C 10 10  JMP $1010
	// ... padding ...
	// 1010: EA        NOP
	// 1011: 60        RTS
	data := make([]byte, 0x12)
	data[0x00] = 0x4C // JMP abs
	data[0x01] = 0x10 // low byte of $1010
	data[0x02] = 0x10 // high byte of $1010
	data[0x10] = 0xEA // NOP
	data[0x11] = 0x60 // RTS

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	addrs := analyzer.InstructionAddresses()

	// Should have 3 instruction addresses: $1000, $1010, $1011
	if len(addrs) != 3 {
		t.Errorf("Expected 3 instruction addresses, got %d", len(addrs))
	}

	// Verify sorted order
	for i := 1; i < len(addrs); i++ {
		if addrs[i] <= addrs[i-1] {
			t.Errorf("InstructionAddresses not sorted: %04X <= %04X", addrs[i], addrs[i-1])
		}
	}

	// Verify expected addresses present
	expected := []uint16{0x1000, 0x1010, 0x1011}
	if len(addrs) == len(expected) {
		for i, addr := range addrs {
			if addr != expected[i] {
				t.Errorf("InstructionAddresses[%d]: got 0x%04X, want 0x%04X", i, addr, expected[i])
			}
		}
	}
}

func TestInstructionAddressesEmpty(t *testing.T) {
	// Analyzer with no analysis performed
	data := []byte{0xEA, 0x60}
	s := newTestState(data, 0x1000, []uint16{})
	analyzer := NewAnalyzer(s, UpdateAll)

	// Don't call Analyze - test empty case
	addrs := analyzer.InstructionAddresses()

	if len(addrs) != 0 {
		t.Errorf("Expected empty slice, got %d addresses", len(addrs))
	}
}

func TestInstructionBoundariesInterface(t *testing.T) {
	// Verify Analyzer implements InstructionBoundaries interface
	data := []byte{0xEA, 0x60}
	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	// This assignment verifies the interface is implemented
	var _ InstructionBoundaries = analyzer

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Use through interface
	var boundaries InstructionBoundaries = analyzer
	if !boundaries.IsInstructionAt(0x1000) {
		t.Error("Interface method IsInstructionAt should return true for 0x1000")
	}
	if len(boundaries.InstructionAddresses()) != 2 {
		t.Error("Interface method InstructionAddresses should return 2 addresses")
	}
}

func TestIsInstructionDataAt(t *testing.T) {
	// Program with various instruction sizes:
	// 1000: 20 34 12  JSR $1234 (3 bytes)
	// 1003: A9 42     LDA #$42  (2 bytes)
	// 1005: EA        NOP       (1 byte)
	// 1006: 60        RTS       (1 byte)
	data := []byte{
		0x20, 0x34, 0x12, // JSR $1234 (target out of bounds, that's ok)
		0xA9, 0x42, // LDA #$42
		0xEA, // NOP
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	tests := []struct {
		addr              uint16
		isInstruction     bool
		isInstructionData bool
		desc              string
	}{
		// JSR $1234 at $1000 (3 bytes)
		{0x1000, true, false, "JSR opcode byte"},
		{0x1001, false, true, "JSR low address byte"},
		{0x1002, false, true, "JSR high address byte"},

		// LDA #$42 at $1003 (2 bytes)
		{0x1003, true, false, "LDA opcode byte"},
		{0x1004, false, true, "LDA immediate operand"},

		// NOP at $1005 (1 byte)
		{0x1005, true, false, "NOP (1-byte instruction)"},

		// RTS at $1006 (1 byte)
		{0x1006, true, false, "RTS (1-byte instruction)"},

		// Addresses outside instruction range
		{0x1007, false, false, "beyond code"},
		{0x0FFF, false, false, "before code"},
	}

	for _, tc := range tests {
		gotInstr := analyzer.IsInstructionAt(tc.addr)
		gotData := analyzer.IsInstructionDataAt(tc.addr)

		if gotInstr != tc.isInstruction {
			t.Errorf("IsInstructionAt(0x%04X) [%s]: got %v, want %v",
				tc.addr, tc.desc, gotInstr, tc.isInstruction)
		}
		if gotData != tc.isInstructionData {
			t.Errorf("IsInstructionDataAt(0x%04X) [%s]: got %v, want %v",
				tc.addr, tc.desc, gotData, tc.isInstructionData)
		}
	}
}

func TestIsInstructionDataAtEmpty(t *testing.T) {
	// Analyzer with no analysis performed
	data := []byte{0xEA, 0x60}
	s := newTestState(data, 0x1000, []uint16{})
	analyzer := NewAnalyzer(s, UpdateAll)

	// Don't call Analyze - test empty case
	if analyzer.IsInstructionDataAt(0x1000) {
		t.Error("IsInstructionDataAt should return false before analysis")
	}
	if analyzer.IsInstructionDataAt(0x1001) {
		t.Error("IsInstructionDataAt should return false before analysis")
	}
}

func TestIsInstructionDataAtMutuallyExclusive(t *testing.T) {
	// Verify that no address can be both an instruction start and instruction data
	// Program: JSR $1010, RTS, ... subroutine at $1010: LDA #$00, RTS
	data := make([]byte, 0x14)
	data[0x00] = 0x20 // JSR
	data[0x01] = 0x10 // low byte of $1010
	data[0x02] = 0x10 // high byte of $1010
	data[0x03] = 0x60 // RTS
	data[0x10] = 0xA9 // LDA #$00
	data[0x11] = 0x00 // immediate value
	data[0x12] = 0x60 // RTS

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Check all analyzed addresses
	for _, addr := range analyzer.InstructionAddresses() {
		if analyzer.IsInstructionDataAt(addr) {
			t.Errorf("Address 0x%04X is both instruction start and instruction data", addr)
		}
	}
}

func TestHardwareSymbolVICBorder(t *testing.T) {
	// Program: LDA $D020 (VIC border color register)
	// 1000: AD 20 D0  LDA $D020
	// 1003: 60        RTS
	data := []byte{
		0xAD, 0x20, 0xD0, // LDA $D020
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify VIC_BORDER symbol created at $D020
	sym, ok := s.Symbols.At(0xD020)
	if !ok || sym.Name != "VIC_BORDER" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected VIC_BORDER symbol at $D020, got %v", sym)
	}
}

func TestHardwareSymbolCIA1(t *testing.T) {
	// Program: STA $DC00 (CIA1 data port A - keyboard)
	// 1000: 8D 00 DC  STA $DC00
	// 1003: 60        RTS
	data := []byte{
		0x8D, 0x00, 0xDC, // STA $DC00
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify CIA1_PRA symbol created at $DC00
	sym, ok := s.Symbols.At(0xDC00)
	if !ok || sym.Name != "CIA1_PRA" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected CIA1_PRA symbol at $DC00, got %v", sym)
	}
}

func TestHardwareSymbolSID(t *testing.T) {
	// Program: STA $D418 (SID volume register)
	// 1000: 8D 18 D4  STA $D418
	// 1003: 60        RTS
	data := []byte{
		0x8D, 0x18, 0xD4, // STA $D418
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify SID_VOLUME symbol created at $D418
	sym, ok := s.Symbols.At(0xD418)
	if !ok || sym.Name != "SID_VOLUME" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected SID_VOLUME symbol at $D418, got %v", sym)
	}
}

func TestHardwareSymbolIndexed(t *testing.T) {
	// Program: LDA $D000,X (VIC sprite X position with index)
	// 1000: BD 00 D0  LDA $D000,X
	// 1003: 60        RTS
	data := []byte{
		0xBD, 0x00, 0xD0, // LDA $D000,X
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify VIC_SPR0_X symbol created at $D000
	sym, ok := s.Symbols.At(0xD000)
	if !ok || sym.Name != "VIC_SPR0_X" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected VIC_SPR0_X symbol at $D000, got %v", sym)
	}
}

func TestHardwareSymbolNoSymbolsFlag(t *testing.T) {
	// Test that hardware symbols are NOT created when UpdateSymbols flag is not set
	// Program: LDA $D020
	data := []byte{
		0xAD, 0x20, 0xD0, // LDA $D020
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	s.Symbols = symbols.NewTable()

	// Only update regions and xrefs, NOT symbols
	analyzer := NewAnalyzer(s, UpdateRegions|UpdateXRefs)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify no symbol created at $D020
	if _, ok := s.Symbols.At(0xD020); ok {
		t.Error("Expected no symbols at $D020 when UpdateSymbols not set")
	}
}

func TestHardwareSymbolNonHardwareAddress(t *testing.T) {
	// Program: LDA $C000 (not a hardware register)
	// 1000: AD 00 C0  LDA $C000
	// 1003: 60        RTS
	data := []byte{
		0xAD, 0x00, 0xC0, // LDA $C000
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify no hardware symbol created at $C000
	if sym, ok := s.Symbols.At(0xC000); ok && sym.Source == symbols.SourceAuto {
		t.Errorf("Unexpected C64ROM symbol at non-hardware address $C000: %s", sym.Name)
	}
}

func TestHardwareSymbolVICMirror(t *testing.T) {
	// Program: LDA $D120 (mirrored VIC border color register)
	// VIC-II mirrors every 64 bytes, so $D120 ($D100 + $20) maps to $D020
	// 1000: AD 20 D1  LDA $D120
	// 1003: 60        RTS
	data := []byte{
		0xAD, 0x20, 0xD1, // LDA $D120
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify VIC_BORDER symbol created at mirrored address $D120
	sym, ok := s.Symbols.At(0xD120)
	if !ok || sym.Name != "VIC_BORDER" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected VIC_BORDER symbol at mirrored address $D120, got %v", sym)
	}
}

func TestHardwareSymbolSIDMirror(t *testing.T) {
	// Program: STA $D520 (mirrored SID register)
	// SID mirrors every 32 bytes, so $D520 ($D500 + $00) maps to $D400 (SID_V1_FREQ_LO)
	// 1000: 8D 00 D5  STA $D500
	// 1003: 60        RTS
	data := []byte{
		0x8D, 0x00, 0xD5, // STA $D500
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify SID_V1_FREQ_LO symbol created at mirrored address $D500
	sym, ok := s.Symbols.At(0xD500)
	if !ok || sym.Name != "SID_V1_FREQ_LO" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected SID_V1_FREQ_LO symbol at mirrored address $D500, got %v", sym)
	}
}

func TestHardwareSymbolCIA1Mirror(t *testing.T) {
	// Program: LDA $DC10 (mirrored CIA1 register)
	// CIA1 mirrors every 16 bytes, so $DC10 maps to $DC00 (CIA1_PRA)
	// 1000: AD 10 DC  LDA $DC10
	// 1003: 60        RTS
	data := []byte{
		0xAD, 0x10, 0xDC, // LDA $DC10
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify CIA1_PRA symbol created at mirrored address $DC10
	sym, ok := s.Symbols.At(0xDC10)
	if !ok || sym.Name != "CIA1_PRA" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected CIA1_PRA symbol at mirrored address $DC10, got %v", sym)
	}
}

func TestHardwareSymbolCIA2Mirror(t *testing.T) {
	// Program: STA $DD20 (mirrored CIA2 register)
	// CIA2 mirrors every 16 bytes, so $DD20 maps to $DD00 (CIA2_PRA)
	// 1000: 8D 20 DD  STA $DD20
	// 1003: 60        RTS
	data := []byte{
		0x8D, 0x20, 0xDD, // STA $DD20
		0x60, // RTS
	}

	s := newTestState(data, 0x1000, []uint16{0x1000})
	analyzer := NewAnalyzer(s, UpdateAll)

	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verify CIA2_PRA symbol created at mirrored address $DD20
	sym, ok := s.Symbols.At(0xDD20)
	if !ok || sym.Name != "CIA2_PRA" || sym.Type != symbols.SymbolByte || sym.Source != symbols.SourceAuto {
		t.Errorf("Expected CIA2_PRA symbol at mirrored address $DD20, got %v", sym)
	}
}

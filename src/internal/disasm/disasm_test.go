package disasm

import (
	"errors"
	"os"
	"strings"
	"testing"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"
	"opcodeoracle/internal/symbols"
)

func TestDisassembleCode(t *testing.T) {
	// Create state with simple code: LDA #$00; STA $D020; RTS
	data := []byte{
		0xA9, 0x00, // LDA #$00
		0x8D, 0x20, 0xD0, // STA $D020
		0x60, // RTS
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0805, regions.RegionCode)

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0806)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check output contains expected mnemonics
	if !strings.Contains(output, "LDA") {
		t.Errorf("Output should contain LDA, got:\n%s", output)
	}
	if !strings.Contains(output, "STA") {
		t.Errorf("Output should contain STA, got:\n%s", output)
	}
	if !strings.Contains(output, "RTS") {
		t.Errorf("Output should contain RTS, got:\n%s", output)
	}
	if !strings.Contains(output, "#$00") {
		t.Errorf("Output should contain #$00, got:\n%s", output)
	}
	if !strings.Contains(output, "$D020") {
		t.Errorf("Output should contain $D020, got:\n%s", output)
	}
}

func TestDisassembleWithLabels(t *testing.T) {
	// Create state with a labeled instruction
	data := []byte{
		0x4C, 0x00, 0x08, // JMP $0800 (infinite loop)
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Symbols.Add(0x0800, symbols.Symbol{
		Name:   "MAIN",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "MAIN:") {
		t.Errorf("Output should contain MAIN: label, got:\n%s", output)
	}
	if !strings.Contains(output, "JMP") {
		t.Errorf("Output should contain JMP, got:\n%s", output)
	}
}

func TestDisassembleBranch(t *testing.T) {
	// Create state with a branch instruction: BNE $0805 with symbol comment
	data := []byte{
		0xCA,       // DEX
		0xD0, 0x02, // BNE +2 (to 0x0805)
		0xEA, // NOP
		0x60, // RTS (at 0x0805)
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0804, regions.RegionCode)
	s.Symbols.Add(0x0805, symbols.Symbol{
		Name:   "L_0805",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceAuto,
	})

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0805)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check that branch shows numeric address with symbol in comment
	if !strings.Contains(output, "BNE $0805") {
		t.Errorf("Output should contain BNE $0805, got:\n%s", output)
	}
	if !strings.Contains(output, "; L_0805") {
		t.Errorf("Output should contain ; L_0805 comment, got:\n%s", output)
	}
}

func TestDisassembleBackwardsBranch(t *testing.T) {
	// Create state with a backwards branch: BNE back to start
	data := []byte{
		0xCA,       // DEX at 0x0800
		0xD0, 0xFD, // BNE -3 (to 0x0800)
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Symbols.Add(0x0800, symbols.Symbol{
		Name:   "LOOP",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check that branch shows numeric address with symbol in comment
	if !strings.Contains(output, "BNE $0800") {
		t.Errorf("Output should contain BNE $0800 for backwards branch, got:\n%s", output)
	}
	if !strings.Contains(output, "; LOOP") {
		t.Errorf("Output should contain ; LOOP comment, got:\n%s", output)
	}
}

func TestDisassembleBranchWithSymbolAndAnnotation(t *testing.T) {
	// Create state with a branch instruction that has both a target symbol and an annotation
	data := []byte{
		0xCA,       // DEX at 0x0800
		0xD0, 0xFD, // BNE -3 (to 0x0800)
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Symbols.Add(0x0800, symbols.Symbol{
		Name:   "loop_start",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})
	// Add annotation to the branch instruction
	s.Annotations.Set(0x0801, "Loop until X is zero", author.User)

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check that branch shows symbol in first comment
	if !strings.Contains(output, "; loop_start") {
		t.Errorf("Output should contain ; loop_start comment, got:\n%s", output)
	}
	// Check that annotation is also present
	if !strings.Contains(output, "; Loop until X is zero") {
		t.Errorf("Output should contain ; Loop until X is zero comment, got:\n%s", output)
	}

	// Verify the symbol appears before the annotation (on the same line as the instruction)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "BNE") {
			if !strings.Contains(line, "; loop_start") {
				t.Errorf("BNE line should have symbol comment on same line, got:\n%s", line)
			}
			if strings.Contains(line, "Loop until X is zero") {
				t.Errorf("Annotation should NOT be on the same line as BNE instruction, got:\n%s", line)
			}
			break
		}
	}
}

func TestDisassembleZeroPageSymbolComment(t *testing.T) {
	// Create state with zero page operand and symbol
	data := []byte{
		0xA5, 0x10, // LDA $10
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0801, regions.RegionCode)
	s.Symbols.Add(0x0010, symbols.Symbol{
		Name:   "ZPVAL",
		Type:   symbols.SymbolByte,
		Source: symbols.SourceUser,
	})

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0802)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "LDA $10") {
		t.Errorf("Output should contain LDA $10, got:\n%s", output)
	}
	if !strings.Contains(output, "; ZPVAL") {
		t.Errorf("Output should contain ; ZPVAL comment, got:\n%s", output)
	}

	// Verify the symbol appears on the same line as the instruction
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "LDA $10") {
			if !strings.Contains(line, "ZPVAL") {
				t.Errorf("LDA line should include ZPVAL comment, got:\n%s", line)
			}
			break
		}
	}
}

func TestDisassembleData(t *testing.T) {
	// Create state with data region
	data := []byte{0x48, 0x45, 0x4C, 0x4C, 0x4F, 0x00}
	s := state.NewState(data, 0x0900, nil, "test.prg")
	// Default is data region

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0900, 0x0906)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, ".BYTE") {
		t.Errorf("Output should contain .BYTE, got:\n%s", output)
	}
	if !strings.Contains(output, "$48") {
		t.Errorf("Output should contain $48, got:\n%s", output)
	}
	// Check ASCII comment
	if !strings.Contains(output, "HELLO") {
		t.Errorf("Output should contain HELLO in ASCII comment, got:\n%s", output)
	}
}

func TestDisassembleDataWithSymbol(t *testing.T) {
	// Create state with labeled data
	data := []byte{0x42, 0x00, 0x10}
	s := state.NewState(data, 0x0900, nil, "test.prg")
	s.Symbols.Add(0x0900, symbols.Symbol{
		Name:   "FLAG",
		Type:   symbols.SymbolByte,
		Source: symbols.SourceUser,
	})
	// SymbolWord is now expanded to _LO and _HI byte symbols
	s.Symbols.Add(0x0901, symbols.Symbol{
		Name:   "PTR",
		Type:   symbols.SymbolWord,
		Source: symbols.SourceUser,
	})

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0900, 0x0903)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "FLAG:") || !strings.Contains(output, ".BYTE $42") {
		t.Errorf("Output should contain FLAG: label and .BYTE $42, got:\n%s", output)
	}
	// Word symbols are now expanded to _LO and _HI byte symbols
	if !strings.Contains(output, "PTR_LO:") || !strings.Contains(output, ".BYTE $00") {
		t.Errorf("Output should contain PTR_LO: label and .BYTE $00, got:\n%s", output)
	}
	if !strings.Contains(output, "PTR_HI:") || !strings.Contains(output, ".BYTE $10") {
		t.Errorf("Output should contain PTR_HI: label and .BYTE $10, got:\n%s", output)
	}
}

func TestDisassembleWithAnnotations(t *testing.T) {
	// Create state with inline annotation
	data := []byte{0xA9, 0x00, 0x60}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Annotations.Set(0x0800, "Load zero", author.User)

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "; Load zero") {
		t.Errorf("Output should contain ; Load zero, got:\n%s", output)
	}
}

func TestDisassembleWithHeadline(t *testing.T) {
	// Create state with headline
	data := []byte{0xA9, 0x00, 0x60}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Headlines.Set(0x0800, "Main entry point", author.User)

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "Main entry point") {
		t.Errorf("Output should contain Main entry point, got:\n%s", output)
	}
	if !strings.Contains(output, "---") {
		t.Errorf("Output should contain headline separator, got:\n%s", output)
	}
}

func TestDisassembleIllegalOpcode(t *testing.T) {
	// Create state with illegal opcode in code region
	data := []byte{0x02} // Illegal opcode
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0800, regions.RegionCode)

	d := NewDisassembler(s, nil)
	_, err := d.Disassemble(0x0800, 0x0801)

	if err == nil {
		t.Fatal("Expected error for illegal opcode")
	}

	var illegalErr *IllegalOpcodeError
	if !errors.As(err, &illegalErr) {
		t.Fatalf("Expected IllegalOpcodeError, got: %v", err)
	}
	if illegalErr.Address != 0x0800 {
		t.Errorf("Expected address 0x0800, got: 0x%04X", illegalErr.Address)
	}
	if illegalErr.Opcode != 0x02 {
		t.Errorf("Expected opcode 0x02, got: 0x%02X", illegalErr.Opcode)
	}

	// Check that it unwraps to ErrIllegalOpcode
	if !errors.Is(err, ErrIllegalOpcode) {
		t.Error("Error should unwrap to ErrIllegalOpcode")
	}
}

func TestDisassembleMidInstruction(t *testing.T) {
	// Create state with a 3-byte instruction: LDA $D020
	data := []byte{
		0xAD, 0x20, 0xD0, // LDA $D020
		0x60, // RTS
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0803, regions.RegionCode)

	// Create a mock boundaries implementation
	boundaries := &mockBoundaries{
		instructionAddrs: map[uint16]bool{0x0800: true, 0x0803: true},
		operandAddrs:     map[uint16]bool{0x0801: true, 0x0802: true},
	}

	d := NewDisassembler(s, boundaries)

	// Disassembling at instruction start should work
	_, err := d.Disassemble(0x0800, 0x0804)
	if err != nil {
		t.Fatalf("Disassemble from instruction start failed: %v", err)
	}

	// Disassembling starting at operand byte should now succeed with warning
	output, err := d.Disassemble(0x0801, 0x0804)
	if err != nil {
		t.Fatalf("Disassemble from mid-instruction should succeed: %v", err)
	}

	// Check output contains warning comment
	if !strings.Contains(output, "; WARNING: mid-instruction byte at $0801") {
		t.Errorf("Output should contain warning comment, got:\n%s", output)
	}

	// Check output contains .BYTE directive for the operand byte
	if !strings.Contains(output, ".BYTE $20") {
		t.Errorf("Output should contain .BYTE $20 for operand byte, got:\n%s", output)
	}

	// Check that it continues to disassemble the RTS
	if !strings.Contains(output, "RTS") {
		t.Errorf("Output should continue with RTS instruction, got:\n%s", output)
	}
}

// mockBoundaries implements analysis.InstructionBoundaries for testing.
type mockBoundaries struct {
	instructionAddrs map[uint16]bool
	operandAddrs     map[uint16]bool
}

func (m *mockBoundaries) IsInstructionAt(addr uint16) bool {
	return m.instructionAddrs[addr]
}

func (m *mockBoundaries) IsInstructionDataAt(addr uint16) bool {
	return m.operandAddrs[addr]
}

func (m *mockBoundaries) InstructionAddresses() []uint16 {
	addrs := make([]uint16, 0, len(m.instructionAddrs))
	for addr := range m.instructionAddrs {
		addrs = append(addrs, addr)
	}
	return addrs
}

func TestDisassembleAddressOutOfRange(t *testing.T) {
	data := []byte{0xEA}
	s := state.NewState(data, 0x0800, nil, "test.prg")

	d := NewDisassembler(s, nil)

	// Test start address out of range
	_, err := d.Disassemble(0x0700, 0x0701)
	if !errors.Is(err, ErrAddressOutOfRange) {
		t.Errorf("Expected ErrAddressOutOfRange for invalid start, got: %v", err)
	}
	var addrErr *AddressOutOfRangeError
	if !errors.As(err, &addrErr) {
		t.Fatalf("Expected AddressOutOfRangeError, got: %v", err)
	}
	if addrErr.Address != 0x0700 {
		t.Errorf("Expected address 0x0700, got: 0x%04X", addrErr.Address)
	}

	// Test end address out of range
	_, err = d.Disassemble(0x0800, 0x0900)
	if !errors.Is(err, ErrAddressOutOfRange) {
		t.Errorf("Expected ErrAddressOutOfRange for invalid end, got: %v", err)
	}
	if !errors.As(err, &addrErr) {
		t.Fatalf("Expected AddressOutOfRangeError, got: %v", err)
	}
	// End is exclusive, so invalid end is 0x08FF (0x0900-1)
	if addrErr.Address != 0x08FF {
		t.Errorf("Expected address 0x08FF (end-1), got: 0x%04X", addrErr.Address)
	}
}

func TestDisassembleInvalidRange(t *testing.T) {
	data := []byte{0xEA, 0xEA, 0xEA}
	s := state.NewState(data, 0x0800, nil, "test.prg")

	d := NewDisassembler(s, nil)

	// Test start > end
	_, err := d.Disassemble(0x0802, 0x0800)
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("Expected ErrInvalidRange for start > end, got: %v", err)
	}
	var rangeErr *InvalidRangeError
	if !errors.As(err, &rangeErr) {
		t.Fatalf("Expected InvalidRangeError, got: %v", err)
	}
	if rangeErr.Start != 0x0802 || rangeErr.End != 0x0800 {
		t.Errorf("Expected start=0x0802 end=0x0800, got: start=0x%04X end=0x%04X", rangeErr.Start, rangeErr.End)
	}
}

func TestDisassembleMixedRegions(t *testing.T) {
	// Create state with code followed by data
	data := []byte{
		0xA9, 0x00, // LDA #$00
		0x60,       // RTS
		0x48, 0x49, // Data: "HI"
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	// 0x0803-0x0804 remains data

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0805)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Should have both code and data
	if !strings.Contains(output, "LDA") {
		t.Errorf("Output should contain LDA, got:\n%s", output)
	}
	if !strings.Contains(output, ".BYTE") {
		t.Errorf("Output should contain .BYTE for data, got:\n%s", output)
	}
	if !strings.Contains(output, "HI") {
		t.Errorf("Output should contain HI in ASCII, got:\n%s", output)
	}
}

func TestDisassembleAllAddressingModes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"Implied", []byte{0xEA}, "NOP"},
		{"Accumulator", []byte{0x0A}, "ASL A"},
		{"Immediate", []byte{0xA9, 0x42}, "#$42"},
		{"ZeroPage", []byte{0xA5, 0x10}, "$10"},
		{"ZeroPageX", []byte{0xB5, 0x10}, "$10,X"},
		{"ZeroPageY", []byte{0xB6, 0x10}, "$10,Y"},
		{"Absolute", []byte{0xAD, 0x00, 0x10}, "$1000"},
		{"AbsoluteX", []byte{0xBD, 0x00, 0x10}, "$1000,X"},
		{"AbsoluteY", []byte{0xB9, 0x00, 0x10}, "$1000,Y"},
		{"Indirect", []byte{0x6C, 0xFC, 0xFF}, "($FFFC)"},
		{"IndexedIndirect", []byte{0xA1, 0x10}, "($10,X)"},
		{"IndirectIndexed", []byte{0xB1, 0x10}, "($10),Y"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := state.NewState(tc.data, 0x0800, nil, "test.prg")
			s.Regions.Set(0x0800, 0x0800+uint16(len(tc.data))-1, regions.RegionCode)

			d := NewDisassembler(s, nil)
			output, err := d.Disassemble(0x0800, 0x0800+uint16(len(tc.data)))
			if err != nil {
				t.Fatalf("Disassemble failed: %v", err)
			}

			if !strings.Contains(output, tc.expected) {
				t.Errorf("Output should contain %s, got:\n%s", tc.expected, output)
			}
		})
	}
}

func TestCalculateBranchTarget(t *testing.T) {
	tests := []struct {
		name     string
		pc       uint16
		operand  byte
		expected uint16
	}{
		{"Forward branch +2", 0x0800, 0x02, 0x0804},
		{"Forward branch +10", 0x0800, 0x0A, 0x080C},
		{"Backward branch -3", 0x0803, 0xFD, 0x0802},
		{"Backward branch -128", 0x0880, 0x80, 0x0802},
		{"Zero offset", 0x0800, 0x00, 0x0802},
		{"Max forward +127", 0x0800, 0x7F, 0x0881},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateBranchTarget(tc.pc, tc.operand)
			if result != tc.expected {
				t.Errorf("calculateBranchTarget(%04X, %02X) = %04X, want %04X",
					tc.pc, tc.operand, result, tc.expected)
			}
		})
	}
}

func TestToASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"Printable", []byte{0x48, 0x45, 0x4C, 0x4C, 0x4F}, "HELLO"},
		{"With null", []byte{0x48, 0x49, 0x00}, "HI."},
		{"Control chars", []byte{0x01, 0x1F, 0x7F}, "..."},
		{"Space", []byte{0x20, 0x20}, "  "},
		{"Tilde", []byte{0x7E}, "~"},
		{"High bit", []byte{0x80, 0xFF}, ".."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toASCII(tc.input)
			if result != tc.expected {
				t.Errorf("toASCII(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestDisassembleEmptyRange(t *testing.T) {
	data := []byte{0xEA}
	s := state.NewState(data, 0x0800, nil, "test.prg")

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0800) // Empty range
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if output != "" {
		t.Errorf("Expected empty output for empty range, got: %q", output)
	}
}

func TestDisassembleMultipleInlineAnnotations(t *testing.T) {
	data := []byte{0xA9, 0x00}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0801, regions.RegionCode)
	// Now we can only have one annotation per author, so use both authors
	s.Annotations.Set(0x0800, "First comment", author.User)
	s.Annotations.Set(0x0800, "Second comment", author.Assistant)

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0802)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "; First comment") {
		t.Errorf("Output should contain first comment, got:\n%s", output)
	}
	if !strings.Contains(output, "; Second comment") {
		t.Errorf("Output should contain second comment, got:\n%s", output)
	}

	// Verify they're on separate lines
	lines := strings.Split(output, "\n")
	foundFirst := false
	foundSecond := false
	for _, line := range lines {
		if strings.Contains(line, "First comment") {
			foundFirst = true
		}
		if strings.Contains(line, "Second comment") {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Error("Each comment should be on a separate line")
	}
}

func TestDisassembleWithXRefs(t *testing.T) {
	// Create state with code that has cross-references
	data := []byte{
		0xA9, 0x00, // LDA #$00 at 0x0800
		0x20, 0x05, 0x08, // JSR $0805 at 0x0802
		0x4C, 0x00, 0x08, // JMP $0800 at 0x0805
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0807, regions.RegionCode)

	// Add symbols
	s.Symbols.Add(0x0800, symbols.Symbol{
		Name:   "MAIN",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})
	s.Symbols.Add(0x0805, symbols.Symbol{
		Name:   "LOOP",
		Type:   symbols.SymbolLabel,
		Source: symbols.SourceUser,
	})

	// Add xrefs: JSR from 0x0802 to 0x0805, JMP from 0x0805 to 0x0800
	s.XRefs.Add(0x0802, 0x0805, "call")
	s.XRefs.Add(0x0805, 0x0800, "jump")

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0808)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check xref to MAIN from JMP at LOOP
	if !strings.Contains(output, "; xref: jump from $0805 (LOOP)") {
		t.Errorf("Output should contain xref to MAIN, got:\n%s", output)
	}

	// Check xref to LOOP from JSR
	if !strings.Contains(output, "; xref: call from $0802") {
		t.Errorf("Output should contain xref to LOOP, got:\n%s", output)
	}
}

func TestDisassembleWithMultipleXRefs(t *testing.T) {
	// Create state with multiple xrefs to same address
	data := []byte{
		0xA9, 0x00, // LDA #$00 at 0x0800 (target)
		0xD0, 0xFC, // BNE $0800 at 0x0802
		0x4C, 0x00, 0x08, // JMP $0800 at 0x0804
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0806, regions.RegionCode)

	// Add xrefs: both branch and jump to 0x0800
	s.XRefs.Add(0x0802, 0x0800, "branch")
	s.XRefs.Add(0x0804, 0x0800, "jump")

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0807)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check both xrefs are present on separate lines
	if !strings.Contains(output, "; xref: branch from $0802") {
		t.Errorf("Output should contain branch xref, got:\n%s", output)
	}
	if !strings.Contains(output, "; xref: jump from $0804") {
		t.Errorf("Output should contain jump xref, got:\n%s", output)
	}

	// Count xref lines to verify they're on separate lines
	xrefCount := strings.Count(output, "; xref:")
	if xrefCount != 2 {
		t.Errorf("Expected 2 separate xref lines, got %d in:\n%s", xrefCount, output)
	}
}

func TestDisassembleXRefOnSameLine(t *testing.T) {
	// Verify xref appears on same line as instruction when no annotation
	data := []byte{
		0xA9, 0x00, // LDA #$00 at 0x0800
		0x4C, 0x00, 0x08, // JMP $0800 at 0x0802
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0804, regions.RegionCode)

	s.XRefs.Add(0x0802, 0x0800, "jump")

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0805)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check that xref is on the same line as LDA
	lines := strings.Split(output, "\n")
	foundXRefOnInstrLine := false
	for _, line := range lines {
		if strings.Contains(line, "LDA") && strings.Contains(line, "; xref:") {
			foundXRefOnInstrLine = true
			break
		}
	}
	if !foundXRefOnInstrLine {
		t.Errorf("Xref should be on same line as instruction, got:\n%s", output)
	}
}

func TestDisassembleXRefWithAnnotation(t *testing.T) {
	// Verify xref appears on continuation line when annotation is present
	data := []byte{
		0xA9, 0x00, // LDA #$00 at 0x0800
		0x4C, 0x00, 0x08, // JMP $0800 at 0x0802
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0804, regions.RegionCode)
	s.Annotations.Set(0x0800, "Load accumulator", author.User)

	s.XRefs.Add(0x0802, 0x0800, "jump")

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x0805)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check that annotation is on the same line as LDA
	lines := strings.Split(output, "\n")
	foundAnnotationOnInstrLine := false
	foundXRefOnSeparateLine := false
	for _, line := range lines {
		if strings.Contains(line, "LDA") && strings.Contains(line, "Load accumulator") {
			foundAnnotationOnInstrLine = true
		}
		if strings.Contains(line, "; xref:") && !strings.Contains(line, "LDA") {
			foundXRefOnSeparateLine = true
		}
	}
	if !foundAnnotationOnInstrLine {
		t.Errorf("Annotation should be on same line as instruction, got:\n%s", output)
	}
	if !foundXRefOnSeparateLine {
		t.Errorf("Xref should be on separate line when annotation present, got:\n%s", output)
	}
}

func TestDisassembleOperandXRefs(t *testing.T) {
	// Test xrefs to operand bytes (self-modifying code)
	data := []byte{
		0xAD, 0x00, 0x10, // LDA $1000 at 0x0800 (3-byte instruction)
		0x8D, 0x01, 0x08, // STA $0801 at 0x0803 (modifies operand byte 1)
		0x8D, 0x02, 0x08, // STA $0802 at 0x0806 (modifies operand byte 2)
		0x60, // RTS
	}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0809, regions.RegionCode)

	// Add xrefs to operand bytes of the LDA instruction
	s.XRefs.Add(0x0803, 0x0801, "write") // STA modifies low byte of LDA operand
	s.XRefs.Add(0x0806, 0x0802, "write") // STA modifies high byte of LDA operand

	d := NewDisassembler(s, nil)
	output, err := d.Disassemble(0x0800, 0x080A)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check for xref+1 (first operand byte)
	if !strings.Contains(output, "; xref+1:") {
		t.Errorf("Output should contain xref+1 for operand byte 1, got:\n%s", output)
	}
	if !strings.Contains(output, "write from $0803") {
		t.Errorf("Output should contain write from $0803, got:\n%s", output)
	}

	// Check for xref+2 (second operand byte)
	if !strings.Contains(output, "; xref+2:") {
		t.Errorf("Output should contain xref+2 for operand byte 2, got:\n%s", output)
	}
	if !strings.Contains(output, "write from $0806") {
		t.Errorf("Output should contain write from $0806, got:\n%s", output)
	}

	// Check for self-modifying code indicator
	if !strings.Contains(output, "[instruction data modified]") {
		t.Errorf("Output should indicate instruction data is modified, got:\n%s", output)
	}
}

func TestIntegrationWithStateFile(t *testing.T) {
	// Skip if testdata file doesn't exist
	testFile := "../../../testdata/Nippon.opcodeoracle.json"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Testdata file not found")
	}

	s, err := stateio.Load(testFile)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	d := NewDisassembler(s, nil)

	// Disassemble a small range
	start := s.Binary.Start()
	end := start + 50
	if end > s.Binary.End()+1 {
		end = s.Binary.End() + 1
	}

	output, err := d.Disassemble(start, end)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Basic sanity checks
	if output == "" {
		t.Error("Output should not be empty")
	}

	// Should contain either code (mnemonics) or data (.BYTE)
	hasCode := strings.Contains(output, "LDA") || strings.Contains(output, "STA") ||
		strings.Contains(output, "JMP") || strings.Contains(output, "JSR") ||
		strings.Contains(output, "NOP") || strings.Contains(output, "RTS")
	hasData := strings.Contains(output, ".BYTE")

	if !hasCode && !hasData {
		t.Errorf("Output should contain code or data, got:\n%s", output)
	}

	t.Logf("Disassembly output:\n%s", output)
}

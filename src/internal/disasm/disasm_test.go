package disasm

import (
	"errors"
	"os"
	"strings"
	"testing"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/binary"
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

	d := NewDisassembler(s)
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

	d := NewDisassembler(s)
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
	// Create state with a branch instruction: BNE L_0805
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

	d := NewDisassembler(s)
	output, err := d.Disassemble(0x0800, 0x0805)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	// Check that branch target is resolved to label
	if !strings.Contains(output, "BNE L_0805") {
		t.Errorf("Output should contain BNE L_0805, got:\n%s", output)
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

	d := NewDisassembler(s)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "BNE LOOP") {
		t.Errorf("Output should contain BNE LOOP for backwards branch, got:\n%s", output)
	}
}

func TestDisassembleData(t *testing.T) {
	// Create state with data region
	data := []byte{0x48, 0x45, 0x4C, 0x4C, 0x4F, 0x00}
	s := state.NewState(data, 0x0900, nil, "test.prg")
	// Default is data region

	d := NewDisassembler(s)
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
	s.Symbols.Add(0x0901, symbols.Symbol{
		Name:   "PTR",
		Type:   symbols.SymbolWord,
		Source: symbols.SourceUser,
	})

	d := NewDisassembler(s)
	output, err := d.Disassemble(0x0900, 0x0903)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "FLAG: .BYTE $42") {
		t.Errorf("Output should contain FLAG: .BYTE $42, got:\n%s", output)
	}
	if !strings.Contains(output, "PTR: .WORD $1000") {
		t.Errorf("Output should contain PTR: .WORD $1000, got:\n%s", output)
	}
}

func TestDisassembleWithAnnotations(t *testing.T) {
	// Create state with inline annotation
	data := []byte{0xA9, 0x00, 0x60}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Annotations.Add(0x0800, annotations.AnnotationInline, "Load zero", "")

	d := NewDisassembler(s)
	output, err := d.Disassemble(0x0800, 0x0803)
	if err != nil {
		t.Fatalf("Disassemble failed: %v", err)
	}

	if !strings.Contains(output, "; Load zero") {
		t.Errorf("Output should contain ; Load zero, got:\n%s", output)
	}
}

func TestDisassembleWithHeadline(t *testing.T) {
	// Create state with headline annotation
	data := []byte{0xA9, 0x00, 0x60}
	s := state.NewState(data, 0x0800, nil, "test.prg")
	s.Regions.Set(0x0800, 0x0802, regions.RegionCode)
	s.Annotations.Add(0x0800, annotations.AnnotationHeadline, "Main entry point", "")

	d := NewDisassembler(s)
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

	d := NewDisassembler(s)
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

func TestDisassembleAddressOutOfRange(t *testing.T) {
	data := []byte{0xEA}
	s := state.NewState(data, 0x0800, nil, "test.prg")

	d := NewDisassembler(s)

	// Test start address out of range
	_, err := d.Disassemble(0x0700, 0x0701)
	if !errors.Is(err, binary.ErrAddressOutOfRange) {
		t.Errorf("Expected ErrAddressOutOfRange for invalid start, got: %v", err)
	}

	// Test end address out of range
	_, err = d.Disassemble(0x0800, 0x0900)
	if !errors.Is(err, binary.ErrAddressOutOfRange) {
		t.Errorf("Expected ErrAddressOutOfRange for invalid end, got: %v", err)
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

	d := NewDisassembler(s)
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

			d := NewDisassembler(s)
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

	d := NewDisassembler(s)
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
	s.Annotations.Add(0x0800, annotations.AnnotationInline, "First comment", "")
	s.Annotations.Add(0x0800, annotations.AnnotationInline, "Second comment", "")

	d := NewDisassembler(s)
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

	d := NewDisassembler(s)

	// Disassemble a small range
	start := s.Binary.Origin
	end := start + 50
	if end > s.Binary.Origin+uint16(len(s.Binary.Data)) {
		end = s.Binary.Origin + uint16(len(s.Binary.Data))
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

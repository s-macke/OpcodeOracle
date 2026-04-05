package disasm

import (
	"strings"
	"testing"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/asm/x86"
	binfile "opcodeoracle/internal/binary"
)

func TestDisassembleCodeOnly(t *testing.T) {
	bin := binfile.New(
		[]byte{0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := analysis.NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	lines, err := NewDisassembler().Disassemble(bin, result)
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].Kind != LineCode || lines[0].Text != "nop" {
		t.Fatalf("first line = %#v", lines[0])
	}
	if lines[1].Kind != LineCode || lines[1].Text != "ret" {
		t.Fatalf("second line = %#v", lines[1])
	}
}

func TestDisassemblePureDataChunksTo16Bytes(t *testing.T) {
	bin := binfile.New(
		[]byte{
			0x41, 0x42, 0x00, 0xff, 0x20, 0x21, 0x30, 0x31,
			0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39,
			0x7a,
		},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		nil,
	)

	lines, err := NewDisassembler().Disassemble(bin, analysis.Result{
		Instructions: map[uint32]x86.Instruction{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].Kind != LineData || len(lines[0].Bytes) != 16 {
		t.Fatalf("first data line = %#v", lines[0])
	}
	if lines[1].Kind != LineData || len(lines[1].Bytes) != 1 {
		t.Fatalf("second data line = %#v", lines[1])
	}
	if lines[0].Comment != "AB.. !0123456789" {
		t.Fatalf("ascii comment = %q", lines[0].Comment)
	}
}

func TestDisassembleDataCarriesIntoNextSegment(t *testing.T) {
	bin := binfile.New(
		make([]byte, 32),
		x86.NewFarAddress(0x1000, 0xfff8),
		x86.NewFarAddress(0x1000, 0xfffe),
		nil,
	)

	lines, err := NewDisassembler().Disassemble(bin, analysis.Result{
		Instructions: map[uint32]x86.Instruction{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].Address != x86.NewFarAddress(0x1fff, 0x0008) {
		t.Fatalf("first line address = %s", lines[0].Address.String())
	}
	if lines[1].Address != x86.NewFarAddress(0x2000, 0x0008) {
		t.Fatalf("second line address = %s", lines[1].Address.String())
	}
}

func TestDisassembleMixedCodeAndData(t *testing.T) {
	bin := binfile.New(
		[]byte{0x90, 0x82, 0xc8, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{
			x86.NewFarAddress(0x1000, 0x0100),
			x86.NewFarAddress(0x1000, 0x0103),
		},
	)

	result, err := analysis.NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	lines, err := NewDisassembler().Disassemble(bin, result)
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 3 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].Kind != LineCode || lines[0].Text != "nop" {
		t.Fatalf("line 0 = %#v", lines[0])
	}
	if lines[1].Kind != LineData || lines[1].Text != "db 82, c8" {
		t.Fatalf("line 1 = %#v", lines[1])
	}
	if lines[2].Kind != LineCode || lines[2].Text != "ret" {
		t.Fatalf("line 2 = %#v", lines[2])
	}
}

func TestDisassembleDoesNotEmitOperandBytesAsData(t *testing.T) {
	bin := binfile.New(
		[]byte{0xb8, 0x34, 0x12, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := analysis.NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	lines, err := NewDisassembler().Disassemble(bin, result)
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 {
		t.Fatalf("lines = %d", len(lines))
	}
	if lines[0].Kind != LineCode || lines[0].Text != "mov ax, 1234" {
		t.Fatalf("line 0 = %#v", lines[0])
	}
	if lines[1].Kind != LineCode || lines[1].Text != "ret" {
		t.Fatalf("line 1 = %#v", lines[1])
	}
}

func TestStringFormatsDataAsciiComment(t *testing.T) {
	lines := []Line{
		{
			Kind:    LineData,
			Address: x86.NewFarAddress(0x1000, 0x0100),
			Text:    "db 41, 42, 00",
			Comment: "AB.",
		},
	}

	got := NewDisassembler().String(lines)
	want := "1000:0100  db 41, 42, 00  ; |AB.|"
	if got != want {
		t.Fatalf("string = %q, want %q", got, want)
	}
}

func TestStringFormatsMultipleLinesInAddressOrder(t *testing.T) {
	lines := []Line{
		{
			Kind:    LineCode,
			Address: x86.NewFarAddress(0x1000, 0x0100),
			Text:    "nop",
		},
		{
			Kind:    LineData,
			Address: x86.NewFarAddress(0x1000, 0x0101),
			Text:    "db 41",
			Comment: "A",
		},
	}

	got := NewDisassembler().String(lines)
	if !strings.Contains(got, "1000:0100  nop\n1000:0101  db 41  ; |A|") {
		t.Fatalf("string = %q", got)
	}
}

package analysis

import (
	"testing"

	"opcodeoracle/internal/asm/x86"
	binfile "opcodeoracle/internal/binary"
)

func TestAnalyzeLinearFallthrough(t *testing.T) {
	bin := binfile.New(
		[]byte{0x90, 0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0101, 0x0102)
}

func TestAnalyzeConditionalJumpFollowsTargetAndFallthrough(t *testing.T) {
	bin := binfile.New(
		[]byte{0x74, 0x02, 0x90, 0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0102, 0x0103, 0x0104)
}

func TestAnalyzeCallFollowsTargetAndFallthrough(t *testing.T) {
	bin := binfile.New(
		[]byte{0xe8, 0x01, 0x00, 0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0103, 0x0104)
}

func TestAnalyzeJumpFollowsTargetOnly(t *testing.T) {
	bin := binfile.New(
		[]byte{0xeb, 0x01, 0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0103)
	if hasInstruction(result, 0x1000, 0x0102) {
		t.Fatalf("unexpected fallthrough decode at 1000:0102")
	}
}

func TestAnalyzeReturnStopsTraversal(t *testing.T) {
	bin := binfile.New(
		[]byte{0xc3, 0x90},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100)
	if hasInstruction(result, 0x1000, 0x0101) {
		t.Fatalf("unexpected instruction after return")
	}
}

func TestAnalyzeInterruptFollowsFallthrough(t *testing.T) {
	bin := binfile.New(
		[]byte{0xcd, 0x10, 0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0102, 0x0103)
}

func TestAnalyzeIndirectCallIsUnresolvedAndNotFollowed(t *testing.T) {
	bin := binfile.New(
		[]byte{0xff, 0x16, 0x34, 0x12, 0x90},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0104)
	if len(result.Unresolved) != 1 || result.Unresolved[0] != x86.NewFarAddress(0x1000, 0x0100) {
		t.Fatalf("unresolved = %#v", result.Unresolved)
	}
}

func TestAnalyzeMarksOperandBytes(t *testing.T) {
	bin := binfile.New(
		[]byte{0xb8, 0x34, 0x12, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertOperandByte(t, result, 0x1000, 0x0101)
	assertOperandByte(t, result, 0x1000, 0x0102)
}

func TestAnalyzeSkipsDecodeInsideOperandBytes(t *testing.T) {
	bin := binfile.New(
		[]byte{0xb8, 0x34, 0x12, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{
			x86.NewFarAddress(0x1000, 0x0100),
			x86.NewFarAddress(0x1000, 0x0101),
		},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0103)
	if hasInstruction(result, 0x1000, 0x0101) {
		t.Fatalf("decoded inside operand bytes")
	}
}

func TestAnalyzeIgnoresOutOfImageTargets(t *testing.T) {
	bin := binfile.New(
		[]byte{0xe9, 0xfd, 0x7f},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100)
}

func TestAnalyzeCOMBinaryFromDefaultEntryPoint(t *testing.T) {
	bin := binfile.New(
		[]byte{0x90, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0101)
}

func TestAnalyzeDBStopsPathAndWarns(t *testing.T) {
	bin := binfile.New(
		[]byte{0x90, 0x6d, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100)
	if len(result.DecodeStops) != 1 || result.DecodeStops[0].Address != x86.NewFarAddress(0x1000, 0x0101) {
		t.Fatalf("decode stops = %#v", result.DecodeStops)
	}
	if result.DecodeStops[0].Err == nil || result.DecodeStops[0].Err.Error() != "x86: unsupported opcode sequence starting with 6d" {
		t.Fatalf("decode stop err = %v", result.DecodeStops[0].Err)
	}
	if hasInstruction(result, 0x1000, 0x0101) || hasInstruction(result, 0x1000, 0x0102) {
		t.Fatalf("db path should not continue")
	}
}

func TestAnalyzeOp32PrefixStopsPathAndWarns(t *testing.T) {
	bin := binfile.New(
		[]byte{0x66, 0xa5, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Instructions) != 0 {
		t.Fatalf("instructions = %#v", result.Instructions)
	}
	if len(result.DecodeStops) != 1 || result.DecodeStops[0].Address != x86.NewFarAddress(0x1000, 0x0100) {
		t.Fatalf("decode stops = %#v", result.DecodeStops)
	}
	if result.DecodeStops[0].Err == nil || result.DecodeStops[0].Err.Error() != "x86: unsupported operand-size override prefix 66" {
		t.Fatalf("decode stop err = %v", result.DecodeStops[0].Err)
	}
}

func TestAnalyzeDecodeErrorStopsOnlyThatPath(t *testing.T) {
	bin := binfile.New(
		[]byte{
			0x90, // nop at 0100
			0xc3, // ret at 0101
			0x90, // padding at 0102
			0x8e, // truncated instruction at 0103
		},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{
			x86.NewFarAddress(0x1000, 0x0100),
			x86.NewFarAddress(0x1000, 0x0103),
		},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0101)
	if len(result.DecodeStops) != 1 || result.DecodeStops[0].Address != x86.NewFarAddress(0x1000, 0x0103) {
		t.Fatalf("decode stops = %#v", result.DecodeStops)
	}
	if result.DecodeStops[0].Err == nil {
		t.Fatal("missing decode stop error")
	}
}

func TestAnalyzeSetCCFallsThrough(t *testing.T) {
	bin := binfile.New(
		[]byte{0x0f, 0x94, 0xc0, 0xc3},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	assertInstructionOffsets(t, result, 0x0100, 0x0103)
}

func TestAnalyzeUnsupportedExtendedOpcodeStopsPath(t *testing.T) {
	bin := binfile.New(
		[]byte{0x0f, 0x31},
		x86.NewFarAddress(0x1000, 0x0100),
		x86.NewFarAddress(0x1000, 0xfffe),
		[]x86.FarAddress{x86.NewFarAddress(0x1000, 0x0100)},
	)

	result, err := NewAnalyzer().Analyze(bin)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Instructions) != 0 {
		t.Fatalf("instructions = %#v", result.Instructions)
	}
	if len(result.DecodeStops) != 1 || result.DecodeStops[0].Address != x86.NewFarAddress(0x1000, 0x0100) {
		t.Fatalf("decode stops = %#v", result.DecodeStops)
	}
}

func assertInstructionOffsets(t *testing.T, result Result, offsets ...uint16) {
	t.Helper()
	if len(result.Instructions) != len(offsets) {
		t.Fatalf("instruction count = %d, want %d", len(result.Instructions), len(offsets))
	}
	for _, offset := range offsets {
		if !hasInstruction(result, 0x1000, offset) {
			t.Fatalf("missing instruction at 1000:%04x", offset)
		}
	}
}

func hasInstruction(result Result, segment uint16, offset uint16) bool {
	_, ok := result.Instructions[x86.NewFarAddress(segment, offset).Linear()]
	return ok
}

func assertOperandByte(t *testing.T, result Result, segment uint16, offset uint16) {
	t.Helper()
	if !result.OperandBytes[x86.NewFarAddress(segment, offset).Linear()] {
		t.Fatalf("missing operand byte mark at %04x:%04x", segment, offset)
	}
}

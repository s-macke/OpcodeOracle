package x86

import (
	"strings"
	"testing"
)

func TestDecodeConditionalJump(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x74, 0x05}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Address.Segment != 0x1000 || inst.Address.Offset != 0x0000 {
		t.Fatalf("address = %+v", inst.Address)
	}
	if inst.Mnemonic != "jz" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Flow != FlowConditionalJump {
		t.Fatalf("unexpected flow metadata: %+v", inst)
	}
	if inst.Target == nil || inst.Target.Near == nil || *inst.Target.Near != 0x0007 {
		t.Fatalf("unexpected target: %+v", inst.Target)
	}
	if inst.Text != "jz 0007" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeNearCall(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xe8, 0x34, 0x12}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Flow != FlowCall {
		t.Fatalf("unexpected flow metadata: %+v", inst)
	}
	if inst.Target == nil || inst.Target.Near == nil || *inst.Target.Near != 0x1237 {
		t.Fatalf("unexpected target: %+v", inst.Target)
	}
	if inst.Text != "call 1237" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeFarJump(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xea, 0x34, 0x12, 0x78, 0x56}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Target == nil || inst.Target.Far == nil {
		t.Fatalf("missing far target: %+v", inst.Target)
	}
	if inst.Target.Far.Offset != 0x1234 || inst.Target.Far.Segment != 0x5678 {
		t.Fatalf("unexpected far target: %+v", inst.Target.Far)
	}
	if inst.Text != "jmp 5678:1234" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeModRMWithSegmentOverride(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x26, 0x8b, 0x40, 0x04}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if len(inst.Prefixes) != 1 || inst.Prefixes[0] != PrefixES {
		t.Fatalf("prefixes = %+v", inst.Prefixes)
	}
	if got := inst.Text; got != "mov ax, es:[bx+si+4]" {
		t.Fatalf("text = %q", got)
	}
	if len(inst.Operands) != 2 || inst.Operands[1].Memory == nil || inst.Operands[1].Memory.SegmentOverride == nil {
		t.Fatalf("unexpected operands: %+v", inst.Operands)
	}
}

func TestDecodeModRMWithFSOverride(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x64, 0x8b, 0x06, 0x34, 0x12}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if len(inst.Prefixes) != 1 || inst.Prefixes[0] != PrefixFS {
		t.Fatalf("prefixes = %+v", inst.Prefixes)
	}
	if got := inst.Text; got != "mov ax, fs:[1234]" {
		t.Fatalf("text = %q", got)
	}
	if len(inst.Operands) != 2 || inst.Operands[1].Memory == nil || inst.Operands[1].Memory.SegmentOverride == nil || *inst.Operands[1].Memory.SegmentOverride != RegFS {
		t.Fatalf("unexpected operands: %+v", inst.Operands)
	}
}

func TestDecodeModRMWithGSOverride(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x65, 0x8b, 0x06, 0x34, 0x12}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if len(inst.Prefixes) != 1 || inst.Prefixes[0] != PrefixGS {
		t.Fatalf("prefixes = %+v", inst.Prefixes)
	}
	if got := inst.Text; got != "mov ax, gs:[1234]" {
		t.Fatalf("text = %q", got)
	}
	if len(inst.Operands) != 2 || inst.Operands[1].Memory == nil || inst.Operands[1].Memory.SegmentOverride == nil || *inst.Operands[1].Memory.SegmentOverride != RegGS {
		t.Fatalf("unexpected operands: %+v", inst.Operands)
	}
}

func TestDecodeGroupedImmediateOpcode(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x83, 0xc0, 0x10}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "add" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "add ax, 10" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeMovImmediateToMemory(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xc6, 0x06, 0x34, 0x12, 0x7f}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "mov" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "mov byte ptr [1234], 7f" {
		t.Fatalf("text = %q", inst.Text)
	}
	if inst.Operands[0].Memory == nil || inst.Operands[0].Memory.DirectAddress == nil || *inst.Operands[0].Memory.DirectAddress != 0x1234 {
		t.Fatalf("unexpected memory operand: %+v", inst.Operands[0])
	}
}

func TestDecodeIndirectFarCall(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xff, 0x1e, 0x34, 0x12}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Flow != FlowCall {
		t.Fatalf("unexpected flow metadata: %+v", inst)
	}
	if inst.Target == nil || !inst.Target.Indirect || inst.Target.Kind != TargetFar {
		t.Fatalf("unexpected target: %+v", inst.Target)
	}
	if inst.Text != "call far, [1234]" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeStringInstruction(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xf3, 0xa5}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "movsw" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "repz movsw" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeRepzOutsb(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xf3, 0x6e}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "outsb" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "repz outsb" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeReturnWithImmediate(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xc2, 0x10, 0x00}, NewFarAddress(0x1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Flow != FlowReturn {
		t.Fatalf("flow = %q", inst.Flow)
	}
	if inst.Text != "ret 0010" {
		t.Fatalf("text = %q", inst.Text)
	}
	if inst.Length != 3 || inst.NextAddress.Offset != 3 || inst.NextAddress.Segment != 0x1000 {
		t.Fatalf("bad next address: len=%d next=%+v", inst.Length, inst.NextAddress)
	}
}

func TestDetailsStringShowsSegmentedNearTarget(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x74, 0x05}, NewFarAddress(0x1234, 0))
	if err != nil {
		t.Fatal(err)
	}

	details := inst.DetailsString()
	if !strings.Contains(details, "Address:     1234:0000") {
		t.Fatalf("missing segmented address in details: %q", details)
	}
	if !strings.Contains(details, "Target:      near 1234:0007 indirect=false") {
		t.Fatalf("missing segmented near target in details: %q", details)
	}
}

func TestFarAddressLinear(t *testing.T) {
	addr := NewFarAddress(0x1234, 0x5678)

	if got := addr.Linear(); got != 0x179b8 {
		t.Fatalf("linear = 0x%x", got)
	}
}

func TestDecodeUsesLogicalAddressButStartsAtMemoryZero(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x74, 0x05}, NewFarAddress(0x2000, 0x0100))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Address != NewFarAddress(0x2000, 0x0100) {
		t.Fatalf("address = %+v", inst.Address)
	}
	if inst.NextAddress != NewFarAddress(0x2000, 0x0102) {
		t.Fatalf("next address = %+v", inst.NextAddress)
	}
	if inst.Target == nil || inst.Target.Near == nil || *inst.Target.Near != 0x0107 {
		t.Fatalf("unexpected target: %+v", inst.Target)
	}
}

func TestDecodeSetZRegister(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x0f, 0x94, 0xc0}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "setz" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "setz al" {
		t.Fatalf("text = %q", inst.Text)
	}
	if inst.Flow != FlowNone {
		t.Fatalf("flow = %q", inst.Flow)
	}
}

func TestDecodeSetNZMemory(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x0f, 0x95, 0x06, 0x34, 0x12}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "setnz" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "setnz byte ptr [1234]" {
		t.Fatalf("text = %q", inst.Text)
	}
	if len(inst.Operands) != 1 || inst.Operands[0].Memory == nil {
		t.Fatalf("operands = %#v", inst.Operands)
	}
}

func TestDecodeSetGMemoryWithDisplacement(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x0f, 0x9f, 0x40, 0x04}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "setg" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "setg byte ptr [bx+si+4]" {
		t.Fatalf("text = %q", inst.Text)
	}
}

func TestDecodeUnsupportedExtendedOpcodeReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0x0f, 0x31}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported extended opcode 0f 31") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeUnsupportedOpcodeReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0x6d}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported opcode sequence starting with 6d") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeUnsupportedOpcode82ReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0x82}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported opcode sequence starting with 82") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeUnsupportedOpcodeC8ReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xc8}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported opcode sequence starting with c8") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeUnsupportedOpcodeC9ReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xc9}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported opcode sequence starting with c9") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeUnsupportedOpcodeD6ReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xd6}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported opcode sequence starting with d6") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeUnsupportedOp32PrefixReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0x66, 0xa5}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported operand-size override prefix 66") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeDanglingPrefixReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xf3}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dangling prefix f3") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeIllegalF6GroupReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xf6, 0xc8}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "illegal grouped opcode f6 /1") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeIllegalFEGroupReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xfe, 0xd8}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "illegal grouped opcode fe /3") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeIllegalFFGroupReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xff, 0xf8}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "illegal grouped opcode ff /7") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeInvalidLockPrefixReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xf0, 0x90}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid lock prefix on nop") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeInvalidRepPrefixReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0xf3, 0x90}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid repz prefix on nop") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeInvalidSegmentOverrideOnRegisterInstructionReturnsError(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0x64, 0x90}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid segment override prefix fs on nop") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeTruncatedExtendedOpcodeReturnsEOF(t *testing.T) {
	dec := NewDecoder()
	_, err := dec.Decode([]byte{0x0f}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodePushImm8(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x6a, 0xff}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "push" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "push ff" {
		t.Fatalf("text = %q", inst.Text)
	}
	if len(inst.Operands) != 1 || inst.Operands[0].Kind != OperandImmediate || inst.Operands[0].Immediate != 0xff {
		t.Fatalf("operands = %#v", inst.Operands)
	}
}

func TestDecodePushImm16(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0x68, 0x34, 0x12}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "push" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "push 1234" {
		t.Fatalf("text = %q", inst.Text)
	}
	if len(inst.Operands) != 1 || inst.Operands[0].Kind != OperandImmediate || inst.Operands[0].Immediate != 0x1234 {
		t.Fatalf("operands = %#v", inst.Operands)
	}
}

func TestDecodeShiftByteImmediateCount(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xc0, 0xe0, 0x04}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "shl" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "shl al, 04" {
		t.Fatalf("text = %q", inst.Text)
	}
	if len(inst.Operands) != 2 || inst.Operands[0].Register != RegAL || inst.Operands[1].Immediate != 0x04 {
		t.Fatalf("operands = %#v", inst.Operands)
	}
}

func TestDecodeShiftWordImmediateCount(t *testing.T) {
	dec := NewDecoder()
	inst, err := dec.Decode([]byte{0xc1, 0xe0, 0x04}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}

	if inst.Mnemonic != "shl" {
		t.Fatalf("mnemonic = %q", inst.Mnemonic)
	}
	if inst.Text != "shl ax, 04" {
		t.Fatalf("text = %q", inst.Text)
	}
	if len(inst.Operands) != 2 || inst.Operands[0].Register != RegAX || inst.Operands[1].Immediate != 0x04 {
		t.Fatalf("operands = %#v", inst.Operands)
	}
}

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

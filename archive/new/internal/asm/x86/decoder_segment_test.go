package x86

import (
	"strings"
	"testing"
)

func TestDecodeFSRegisterEncoding(t *testing.T) {
	dec := NewDecoder()

	inst, err := dec.Decode([]byte{0x8e, 0xe0}, NewFarAddress(0x1000, 0x0000))
	if err != nil {
		t.Fatal(err)
	}
	if inst.Text != "mov fs, ax" {
		t.Fatalf("text = %q", inst.Text)
	}
	if len(inst.Operands) != 2 || inst.Operands[0].Register != RegFS || inst.Operands[1].Register != RegAX {
		t.Fatalf("operands = %#v", inst.Operands)
	}
}

func TestDecodeInvalidSegmentRegisterEncodingReturnsError(t *testing.T) {
	dec := NewDecoder()

	_, err := dec.Decode([]byte{0x8e, 0xf0}, NewFarAddress(0x1000, 0x0000))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid segment register encoding 6") {
		t.Fatalf("error = %v", err)
	}
}

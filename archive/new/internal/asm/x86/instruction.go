package x86

type Instruction struct {
	Address     FarAddress
	NextAddress FarAddress
	Length      uint8
	Bytes       []byte
	Opcode      byte
	Prefixes    []Prefix
	Mnemonic    Mnemonic
	Operands    []Operand
	Addressing  AddressingKind
	Flow        FlowKind
	Target      *Target
	Text        string
}

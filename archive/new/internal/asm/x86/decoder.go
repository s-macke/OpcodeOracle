package x86

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errUnexpectedEOF = errors.New("x86: unexpected end of input")
)

const (
	flagPrefix = 1 << iota
)

type opcodeEntry struct {
	mnemonic Mnemonic
	decode   decodeFunc
	flags    int
	group    []Mnemonic
}

type decodeFunc func(*decodeState) error

type decodeState struct {
	data        []byte
	start       uint16
	pos         uint16
	inst        Instruction
	segOverride *Register
}

func (d *Decoder) Decode(memory []byte, address FarAddress) (Instruction, error) {
	if len(memory) == 0 {
		return Instruction{}, errUnexpectedEOF
	}

	s := &decodeState{
		data:  memory,
		start: 0,
		pos:   0,
		inst: Instruction{
			Address: address,
		},
	}

	var entry opcodeEntry
	for {
		opcode, err := s.readByte()
		if err != nil {
			return Instruction{}, err
		}
		s.inst.Opcode = opcode
		entry = opcodeTable[opcode]
		s.inst.Mnemonic = entry.mnemonic

		if entry.flags&flagPrefix == 0 {
			break
		}

		pfx := prefixFromOpcode(opcode)
		if pfx != "" {
			s.inst.Prefixes = append(s.inst.Prefixes, pfx)
			if reg := prefixSegmentRegister(pfx); reg != nil {
				s.segOverride = reg
			}
		}

		if int(s.pos) >= len(s.data) {
			s.finalize()
			return s.inst, nil
		}
	}

	if entry.decode != nil {
		if err := entry.decode(s); err != nil {
			return Instruction{}, err
		}
	}

	s.classify()
	s.finalize()
	return s.inst, nil
}

func (s *decodeState) finalize() {
	s.inst.NextAddress = NewFarAddress(s.inst.Address.Segment, s.inst.Address.Offset+s.pos)
	s.inst.Length = uint8(s.pos - s.start)
	s.inst.Bytes = append([]byte(nil), s.data[s.start:s.pos]...)
	s.inst.Addressing = classifyAddressing(s.inst.Operands)
	s.inst.Text = s.inst.AssemblerString()
}

func (s *decodeState) classify() {
	s.inst.Flow = FlowNone
	switch s.inst.Mnemonic {
	case "call":
		s.inst.Flow = FlowCall
	case "jmp":
		s.inst.Flow = FlowJump
	case "ret", "retf", "iret":
		s.inst.Flow = FlowReturn
	case "int", "into":
		s.inst.Flow = FlowInterrupt
	case "loopne", "loope", "loop", "jcxz":
		s.inst.Flow = FlowConditionalJump
	default:
		if strings.HasPrefix(string(s.inst.Mnemonic), "j") && s.inst.Mnemonic != "jmp" {
			s.inst.Flow = FlowConditionalJump
		}
	}
}

func (s *decodeState) readByte() (byte, error) {
	if int(s.pos) >= len(s.data) {
		return 0, errUnexpectedEOF
	}
	b := s.data[s.pos]
	s.pos++
	return b, nil
}

func (s *decodeState) peekByte() (byte, error) {
	if int(s.pos) >= len(s.data) {
		return 0, errUnexpectedEOF
	}
	return s.data[s.pos], nil
}

func (s *decodeState) readUint16() (uint16, error) {
	lo, err := s.readByte()
	if err != nil {
		return 0, err
	}
	hi, err := s.readByte()
	if err != nil {
		return 0, err
	}
	return uint16(lo) | uint16(hi)<<8, nil
}

func (s *decodeState) readInt8() (int8, error) {
	v, err := s.readByte()
	return int8(v), err
}

func (s *decodeState) readInt16() (int16, error) {
	v, err := s.readUint16()
	return int16(v), err
}

func (s *decodeState) withOperands(ops ...Operand) {
	s.inst.Operands = append(s.inst.Operands, ops...)
}

func (s *decodeState) relativeTarget(delta int16) Operand {
	base := s.inst.Address.Offset + s.pos
	resolved := uint16(int32(base) + int32(delta))
	rel := &RelativeRef{
		BaseOffset: base,
		Delta:      delta,
		Resolved:   resolved,
	}
	s.inst.Target = &Target{
		Kind: TargetNear,
		Near: &resolved,
	}
	return Operand{
		Kind:     OperandRelative,
		Width:    Width16,
		Relative: rel,
	}
}

func (s *decodeState) farTarget(seg, off uint16) Operand {
	addr := new(FarAddress)
	*addr = NewFarAddress(seg, off)
	s.inst.Target = &Target{
		Kind: TargetFar,
		Far:  addr,
	}
	return Operand{
		Kind:  OperandFar,
		Width: Width16,
		Far:   addr,
	}
}

func classifyAddressing(ops []Operand) AddressingKind {
	if len(ops) == 0 {
		return AddressingNone
	}
	hasReg := false
	hasMem := false
	hasImm := false
	hasRel := false
	for _, op := range ops {
		switch op.Kind {
		case OperandRegister:
			hasReg = true
		case OperandMemory:
			hasMem = true
		case OperandImmediate:
			hasImm = true
		case OperandRelative, OperandFar:
			hasRel = true
		}
	}
	switch {
	case hasRel:
		return AddressingRelative
	case hasMem && !hasReg && !hasImm:
		return AddressingMemory
	case hasImm && !hasReg && !hasMem:
		return AddressingImmediate
	case hasReg && !hasMem && !hasImm:
		return AddressingRegister
	default:
		return AddressingMixed
	}
}

var (
	byteRegs  = []Register{RegAL, RegCL, RegDL, RegBL, RegAH, RegCH, RegDH, RegBH}
	wordRegs  = []Register{RegAX, RegCX, RegDX, RegBX, RegSP, RegBP, RegSI, RegDI}
	segRegs   = []Register{RegES, RegCS, RegSS, RegDS}
	indexRegs = []BaseAddressExpr{
		BaseBXSI, BaseBXDI, BaseBPSI, BaseBPDI, BaseSI, BaseDI, BaseBP, BaseBX,
	}
	conditions = []Mnemonic{
		"jo", "jno", "jb", "jae", "jz", "jnz", "jbe", "ja",
		"js", "jns", "jp", "jnp", "jl", "jge", "jle", "jg",
	}
	table8x  = []Mnemonic{"add", "or", "adc", "sbb", "and", "sub", "xor", "cmp"}
	tableDX  = []Mnemonic{"rol", "ror", "rcl", "rcr", "shl", "shr", "shl", "sar"}
	tableF67 = []Mnemonic{"test", "illegal", "not", "neg", "mul", "imul", "div", "idiv"}
	tableFE  = []Mnemonic{"inc", "dec", "illegal", "illegal", "illegal", "illegal", "illegal", "illegal"}
	tableFF  = []Mnemonic{"inc", "dec", "call", "call", "jmp", "jmp", "push", "illegal"}
)

func regOperand(reg Register, width OperandWidth) Operand {
	return Operand{Kind: OperandRegister, Width: width, Register: reg}
}

func immOperand(v uint16, width OperandWidth) Operand {
	return Operand{Kind: OperandImmediate, Width: width, Immediate: v}
}

func textOperand(text string) Operand {
	return Operand{Kind: OperandText, Text: text}
}

func memoryOperand(mem *MemoryRef) Operand {
	return Operand{Kind: OperandMemory, Width: mem.Size, Memory: mem}
}

func (s *decodeState) decodeRM(modrm byte, width OperandWidth, ptr PointerQualifier) (Operand, error) {
	mod := modrm & 0xc0
	rm := modrm & 0x07

	if mod == 0xc0 {
		if width == Width8 {
			return regOperand(byteRegs[rm], Width8), nil
		}
		return regOperand(wordRegs[rm], Width16), nil
	}

	mem := &MemoryRef{
		Size:    width,
		Pointer: ptr,
	}
	if s.segOverride != nil {
		mem.SegmentOverride = s.segOverride
	}

	switch mod {
	case 0x00:
		if rm == 6 {
			addr, err := s.readUint16()
			if err != nil {
				return Operand{}, err
			}
			mem.DirectAddress = &addr
			mem.Base = BaseDirect
			return memoryOperand(mem), nil
		}
		mem.Base = indexRegs[rm]
	case 0x40:
		mem.Base = indexRegs[rm]
		disp, err := s.readInt8()
		if err != nil {
			return Operand{}, err
		}
		mem.Displacement = int16(disp)
	case 0x80:
		mem.Base = indexRegs[rm]
		disp, err := s.readInt16()
		if err != nil {
			return Operand{}, err
		}
		mem.Displacement = disp
	}
	return memoryOperand(mem), nil
}

func decodeBR8(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	dst, err := s.decodeRM(modrm, Width8, PtrNone)
	if err != nil {
		return err
	}
	src := regOperand(byteRegs[(modrm&0x38)>>3], Width8)
	s.withOperands(dst, src)
	return nil
}

func decodeR8B(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	src, err := s.decodeRM(modrm, Width8, PtrNone)
	if err != nil {
		return err
	}
	dst := regOperand(byteRegs[(modrm&0x38)>>3], Width8)
	s.withOperands(dst, src)
	return nil
}

func decodeWR16(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	dst, err := s.decodeRM(modrm, Width16, PtrNone)
	if err != nil {
		return err
	}
	src := regOperand(wordRegs[(modrm&0x38)>>3], Width16)
	s.withOperands(dst, src)
	return nil
}

func decodeR16W(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	src, err := s.decodeRM(modrm, Width16, PtrNone)
	if err != nil {
		return err
	}
	dst := regOperand(wordRegs[(modrm&0x38)>>3], Width16)
	s.withOperands(dst, src)
	return nil
}

func decodeALD8(s *decodeState) error {
	v, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(regOperand(RegAL, Width8), immOperand(uint16(v), Width8))
	return nil
}

func decodeAXD16(s *decodeState) error {
	v, err := s.readUint16()
	if err != nil {
		return err
	}
	s.withOperands(regOperand(RegAX, Width16), immOperand(v, Width16))
	return nil
}

func decodePushPopSeg(s *decodeState) error {
	var reg Register
	switch s.inst.Opcode {
	case 0x06, 0x07:
		reg = RegES
	case 0x0e:
		reg = RegCS
	case 0x16, 0x17:
		reg = RegSS
	case 0x1e, 0x1f:
		reg = RegDS
	default:
		return fmt.Errorf("x86: invalid segment opcode %02x", s.inst.Opcode)
	}
	s.withOperands(regOperand(reg, Width16))
	return nil
}

func decodeDataByte(s *decodeState) error {
	s.withOperands(immOperand(uint16(s.inst.Opcode), Width8))
	return nil
}

func decodeWordReg(s *decodeState) error {
	s.withOperands(regOperand(wordRegs[s.inst.Opcode&0x07], Width16))
	return nil
}

func decodeCondJump(s *decodeState) error {
	disp, err := s.readInt8()
	if err != nil {
		return err
	}
	s.inst.Mnemonic = conditions[s.inst.Opcode&0x0f]
	s.withOperands(s.relativeTarget(int16(disp)))
	return nil
}

func decodeBD8Group(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	s.inst.Mnemonic = table8x[(modrm&0x38)>>3]
	dst, err := s.decodeRM(modrm, Width8, PtrByte)
	if err != nil {
		return err
	}
	imm, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(dst, immOperand(uint16(imm), Width8))
	return nil
}

func decodeWD16Group(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	s.inst.Mnemonic = table8x[(modrm&0x38)>>3]
	dst, err := s.decodeRM(modrm, Width16, PtrWord)
	if err != nil {
		return err
	}
	imm, err := s.readUint16()
	if err != nil {
		return err
	}
	s.withOperands(dst, immOperand(imm, Width16))
	return nil
}

func decodeWD8Group(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	s.inst.Mnemonic = table8x[(modrm&0x38)>>3]
	dst, err := s.decodeRM(modrm, Width16, PtrWord)
	if err != nil {
		return err
	}
	imm, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(dst, immOperand(uint16(imm), Width8))
	return nil
}

func decodeMovBD8(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	dst, err := s.decodeRM(modrm, Width8, PtrByte)
	if err != nil {
		return err
	}
	imm, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(dst, immOperand(uint16(imm), Width8))
	return nil
}

func decodeMovWD16(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	dst, err := s.decodeRM(modrm, Width16, PtrWord)
	if err != nil {
		return err
	}
	imm, err := s.readUint16()
	if err != nil {
		return err
	}
	s.withOperands(dst, immOperand(imm, Width16))
	return nil
}

func decodeWS(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	dst, err := s.decodeRM(modrm, Width16, PtrNone)
	if err != nil {
		return err
	}
	src := regOperand(segRegs[(modrm&0x38)>>3], Width16)
	s.withOperands(dst, src)
	return nil
}

func decodeSW(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	src, err := s.decodeRM(modrm, Width16, PtrNone)
	if err != nil {
		return err
	}
	dst := regOperand(segRegs[(modrm&0x38)>>3], Width16)
	s.withOperands(dst, src)
	return nil
}

func decodeW(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	op, err := s.decodeRM(modrm, Width16, PtrWord)
	if err != nil {
		return err
	}
	s.withOperands(op)
	return nil
}

func decodeB(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	op, err := s.decodeRM(modrm, Width8, PtrByte)
	if err != nil {
		return err
	}
	s.withOperands(op)
	return nil
}

func decodeString(base Mnemonic) decodeFunc {
	return func(s *decodeState) error {
		if s.inst.Opcode&0x01 == 0x01 {
			s.inst.Mnemonic = base + "w"
		} else {
			s.inst.Mnemonic = base + "b"
		}
		return nil
	}
}

func decodeXchgAX(s *decodeState) error {
	s.withOperands(regOperand(RegAX, Width16), regOperand(wordRegs[s.inst.Opcode&0x07], Width16))
	return nil
}

func decodeFar(s *decodeState) error {
	off, err := s.readUint16()
	if err != nil {
		return err
	}
	seg, err := s.readUint16()
	if err != nil {
		return err
	}
	s.withOperands(s.farTarget(seg, off))
	return nil
}

func decodeALMem(s *decodeState) error {
	addr, err := s.readUint16()
	if err != nil {
		return err
	}
	mem := &MemoryRef{DirectAddress: &addr, Size: Width8}
	if s.segOverride != nil {
		mem.SegmentOverride = s.segOverride
	}
	s.withOperands(regOperand(RegAL, Width8), memoryOperand(mem))
	return nil
}

func decodeAXMem(s *decodeState) error {
	addr, err := s.readUint16()
	if err != nil {
		return err
	}
	mem := &MemoryRef{DirectAddress: &addr, Size: Width16}
	if s.segOverride != nil {
		mem.SegmentOverride = s.segOverride
	}
	s.withOperands(regOperand(RegAX, Width16), memoryOperand(mem))
	return nil
}

func decodeMemAL(s *decodeState) error {
	addr, err := s.readUint16()
	if err != nil {
		return err
	}
	mem := &MemoryRef{DirectAddress: &addr, Size: Width8}
	if s.segOverride != nil {
		mem.SegmentOverride = s.segOverride
	}
	s.withOperands(memoryOperand(mem), regOperand(RegAL, Width8))
	return nil
}

func decodeMemAX(s *decodeState) error {
	addr, err := s.readUint16()
	if err != nil {
		return err
	}
	mem := &MemoryRef{DirectAddress: &addr, Size: Width16}
	if s.segOverride != nil {
		mem.SegmentOverride = s.segOverride
	}
	s.withOperands(memoryOperand(mem), regOperand(RegAX, Width16))
	return nil
}

func decodeRD(s *decodeState) error {
	idx := s.inst.Opcode & 0x0f
	if idx > 7 {
		v, err := s.readUint16()
		if err != nil {
			return err
		}
		s.withOperands(regOperand(wordRegs[s.inst.Opcode&0x07], Width16), immOperand(v, Width16))
		return nil
	}
	v, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(regOperand(byteRegs[s.inst.Opcode&0x07], Width8), immOperand(uint16(v), Width8))
	return nil
}

func decodeD16(s *decodeState) error {
	v, err := s.readUint16()
	if err != nil {
		return err
	}
	s.withOperands(immOperand(v, Width16))
	return nil
}

func decodeInt3(s *decodeState) error {
	s.withOperands(immOperand(3, Width8))
	return nil
}

func decodeD8(s *decodeState) error {
	v, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(immOperand(uint16(v), Width8))
	return nil
}

func decodeBit1(width OperandWidth, ptr PointerQualifier, table []Mnemonic) decodeFunc {
	return func(s *decodeState) error {
		modrm, err := s.readByte()
		if err != nil {
			return err
		}
		s.inst.Mnemonic = table[(modrm&0x38)>>3]
		op, err := s.decodeRM(modrm, width, ptr)
		if err != nil {
			return err
		}
		s.withOperands(op, immOperand(1, Width8))
		return nil
	}
}

func decodeBitCL(width OperandWidth, ptr PointerQualifier, table []Mnemonic) decodeFunc {
	return func(s *decodeState) error {
		modrm, err := s.readByte()
		if err != nil {
			return err
		}
		s.inst.Mnemonic = table[(modrm&0x38)>>3]
		op, err := s.decodeRM(modrm, width, ptr)
		if err != nil {
			return err
		}
		s.withOperands(op, regOperand(RegCL, Width8))
		return nil
	}
}

func decodeDisp(s *decodeState) error {
	disp, err := s.readInt8()
	if err != nil {
		return err
	}
	s.withOperands(s.relativeTarget(int16(disp)))
	return nil
}

func decodeEscape(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	op, err := s.decodeRM(modrm, Width16, PtrNone)
	if err != nil {
		return err
	}
	s.withOperands(immOperand(uint16(s.inst.Opcode&0x07), Width8), op)
	return nil
}

func decodeAdjust(s *decodeState) error {
	v, err := s.readByte()
	if err != nil {
		return err
	}
	if v != 10 {
		s.withOperands(immOperand(uint16(v), Width8))
	}
	return nil
}

func decodeD8AL(s *decodeState) error {
	v, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(immOperand(uint16(v), Width8), regOperand(RegAL, Width8))
	return nil
}

func decodeD8AX(s *decodeState) error {
	v, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(immOperand(uint16(v), Width8), regOperand(RegAX, Width16))
	return nil
}

func decodeAXD8(s *decodeState) error {
	v, err := s.readByte()
	if err != nil {
		return err
	}
	s.withOperands(regOperand(RegAX, Width16), immOperand(uint16(v), Width8))
	return nil
}

func decodeDisp16(s *decodeState) error {
	disp, err := s.readInt16()
	if err != nil {
		return err
	}
	s.withOperands(s.relativeTarget(disp))
	return nil
}

func decodeFarIndirect(s *decodeState) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	op, err := s.decodeRM(modrm, Width16, PtrNone)
	if err != nil {
		return err
	}
	s.inst.Target = &Target{Kind: TargetFar, Indirect: true}
	s.withOperands(textOperand("far"), op)
	return nil
}

func decodePortDX(s *decodeState) error {
	switch s.inst.Opcode {
	case 0xec:
		s.withOperands(regOperand(RegAL, Width8), regOperand(RegDX, Width16))
	case 0xed:
		s.withOperands(regOperand(RegAX, Width16), regOperand(RegDX, Width16))
	case 0xee:
		s.withOperands(regOperand(RegDX, Width16), regOperand(RegAL, Width8))
	case 0xef:
		s.withOperands(regOperand(RegDX, Width16), regOperand(RegAX, Width16))
	}
	return nil
}

func decodeF6(s *decodeState) error {
	modrm, err := s.peekByte()
	if err != nil {
		return err
	}
	if (modrm & 0x38) == 0x00 {
		s.inst.Mnemonic = "test"
		return decodeBD8Group(s)
	}
	return decodeGroupedUnary(s, Width8, PtrByte, tableF67)
}

func decodeF7(s *decodeState) error {
	modrm, err := s.peekByte()
	if err != nil {
		return err
	}
	if (modrm & 0x38) == 0x00 {
		s.inst.Mnemonic = "test"
		return decodeWD16Group(s)
	}
	return decodeGroupedUnary(s, Width16, PtrWord, tableF67)
}

func decodeGroupedUnary(s *decodeState, width OperandWidth, ptr PointerQualifier, table []Mnemonic) error {
	modrm, err := s.readByte()
	if err != nil {
		return err
	}
	s.inst.Mnemonic = table[(modrm&0x38)>>3]
	op, err := s.decodeRM(modrm, width, ptr)
	if err != nil {
		return err
	}
	s.withOperands(op)
	return nil
}

func decodeFF(s *decodeState) error {
	modrm, err := s.peekByte()
	if err != nil {
		return err
	}
	group := (modrm & 0x38) >> 3
	if group == 3 || group == 5 {
		s.inst.Mnemonic = tableFF[group]
		return decodeFarIndirect(s)
	}

	if err := decodeGroupedUnary(s, Width16, PtrWord, tableFF); err != nil {
		return err
	}
	if s.inst.Mnemonic == "call" || s.inst.Mnemonic == "jmp" {
		s.inst.Target = &Target{Kind: TargetNear, Indirect: true}
	}
	return nil
}

func decodeBiosCall(s *decodeState) error {
	next, err := s.peekByte()
	if err != nil {
		return fmt.Errorf("x86: bioscall prefix missing payload: %w", err)
	}
	if next != 0xf1 {
		s.inst.Mnemonic = "db"
		s.withOperands(immOperand(0xf1, Width8))
		return nil
	}

	_, _ = s.readByte()
	var addr uint32
	for shift := 0; shift < 32; shift += 8 {
		b, err := s.readByte()
		if err != nil {
			return err
		}
		addr |= uint32(b) << shift
	}
	s.inst.Mnemonic = "bios"
	s.withOperands(textOperand(fmt.Sprintf("%08x", addr)))
	return nil
}

func prefixFromOpcode(op byte) Prefix {
	switch op {
	case 0x26:
		return PrefixES
	case 0x2e:
		return PrefixCS
	case 0x36:
		return PrefixSS
	case 0x3e:
		return PrefixDS
	case 0x66:
		return PrefixOp32
	case 0xf0:
		return PrefixLock
	case 0xf2:
		return PrefixRepNZ
	case 0xf3:
		return PrefixRepZ
	default:
		return ""
	}
}

func prefixSegmentRegister(p Prefix) *Register {
	switch p {
	case PrefixES:
		r := RegES
		return &r
	case PrefixCS:
		r := RegCS
		return &r
	case PrefixSS:
		r := RegSS
		return &r
	case PrefixDS:
		r := RegDS
		return &r
	default:
		return nil
	}
}

var opcodeTable = func() [256]opcodeEntry {
	var t [256]opcodeEntry
	set := func(op byte, m Mnemonic, f decodeFunc, flags int) {
		t[op] = opcodeEntry{mnemonic: m, decode: f, flags: flags}
	}

	set(0x00, "add", decodeBR8, 0)
	set(0x01, "add", decodeWR16, 0)
	set(0x02, "add", decodeR8B, 0)
	set(0x03, "add", decodeR16W, 0)
	set(0x04, "add", decodeALD8, 0)
	set(0x05, "add", decodeAXD16, 0)
	set(0x06, "push", decodePushPopSeg, 0)
	set(0x07, "pop", decodePushPopSeg, 0)
	set(0x08, "or", decodeBR8, 0)
	set(0x09, "or", decodeWR16, 0)
	set(0x0a, "or", decodeR8B, 0)
	set(0x0b, "or", decodeR16W, 0)
	set(0x0c, "or", decodeALD8, 0)
	set(0x0d, "or", decodeAXD16, 0)
	set(0x0e, "push", decodePushPopSeg, 0)
	set(0x0f, "db", decodeDataByte, 0)
	set(0x10, "adc", decodeBR8, 0)
	set(0x11, "adc", decodeWR16, 0)
	set(0x12, "adc", decodeR8B, 0)
	set(0x13, "adc", decodeR16W, 0)
	set(0x14, "adc", decodeALD8, 0)
	set(0x15, "adc", decodeAXD16, 0)
	set(0x16, "push", decodePushPopSeg, 0)
	set(0x17, "pop", decodePushPopSeg, 0)
	set(0x18, "sbb", decodeBR8, 0)
	set(0x19, "sbb", decodeWR16, 0)
	set(0x1a, "sbb", decodeR8B, 0)
	set(0x1b, "sbb", decodeR16W, 0)
	set(0x1c, "sbb", decodeALD8, 0)
	set(0x1d, "sbb", decodeAXD16, 0)
	set(0x1e, "push", decodePushPopSeg, 0)
	set(0x1f, "pop", decodePushPopSeg, 0)
	set(0x20, "and", decodeBR8, 0)
	set(0x21, "and", decodeWR16, 0)
	set(0x22, "and", decodeR8B, 0)
	set(0x23, "and", decodeR16W, 0)
	set(0x24, "and", decodeALD8, 0)
	set(0x25, "and", decodeAXD16, 0)
	set(0x26, "es", nil, flagPrefix)
	set(0x27, "daa", nil, 0)
	set(0x28, "sub", decodeBR8, 0)
	set(0x29, "sub", decodeWR16, 0)
	set(0x2a, "sub", decodeR8B, 0)
	set(0x2b, "sub", decodeR16W, 0)
	set(0x2c, "sub", decodeALD8, 0)
	set(0x2d, "sub", decodeAXD16, 0)
	set(0x2e, "cs", nil, flagPrefix)
	set(0x2f, "das", nil, 0)
	set(0x30, "xor", decodeBR8, 0)
	set(0x31, "xor", decodeWR16, 0)
	set(0x32, "xor", decodeR8B, 0)
	set(0x33, "xor", decodeR16W, 0)
	set(0x34, "xor", decodeALD8, 0)
	set(0x35, "xor", decodeAXD16, 0)
	set(0x36, "ss", nil, flagPrefix)
	set(0x37, "aaa", nil, 0)
	set(0x38, "cmp", decodeBR8, 0)
	set(0x39, "cmp", decodeWR16, 0)
	set(0x3a, "cmp", decodeR8B, 0)
	set(0x3b, "cmp", decodeR16W, 0)
	set(0x3c, "cmp", decodeALD8, 0)
	set(0x3d, "cmp", decodeAXD16, 0)
	set(0x3e, "ds", nil, flagPrefix)
	set(0x3f, "aas", nil, 0)
	for op := byte(0x40); op <= 0x47; op++ {
		set(op, "inc", decodeWordReg, 0)
	}
	for op := byte(0x48); op <= 0x4f; op++ {
		set(op, "dec", decodeWordReg, 0)
	}
	for op := byte(0x50); op <= 0x57; op++ {
		set(op, "push", decodeWordReg, 0)
	}
	for op := byte(0x58); op <= 0x5f; op++ {
		set(op, "pop", decodeWordReg, 0)
	}
	for op := byte(0x60); op <= 0x65; op++ {
		set(op, "db", decodeDataByte, 0)
	}
	set(0x66, "32", nil, flagPrefix)
	for op := byte(0x67); op <= 0x6f; op++ {
		set(op, "db", decodeDataByte, 0)
	}
	for op := byte(0x70); op <= 0x7f; op++ {
		set(op, "j", decodeCondJump, 0)
	}
	set(0x80, "", decodeBD8Group, 0)
	set(0x81, "", decodeWD16Group, 0)
	set(0x82, "db", decodeDataByte, 0)
	set(0x83, "", decodeWD8Group, 0)
	set(0x84, "test", decodeBR8, 0)
	set(0x85, "test", decodeWR16, 0)
	set(0x86, "xchg", decodeBR8, 0)
	set(0x87, "xchg", decodeWR16, 0)
	set(0x88, "mov", decodeBR8, 0)
	set(0x89, "mov", decodeWR16, 0)
	set(0x8a, "mov", decodeR8B, 0)
	set(0x8b, "mov", decodeR16W, 0)
	set(0x8c, "mov", decodeWS, 0)
	set(0x8d, "lea", decodeR16W, 0)
	set(0x8e, "mov", decodeSW, 0)
	set(0x8f, "pop", decodeW, 0)
	set(0x90, "nop", nil, 0)
	for op := byte(0x91); op <= 0x97; op++ {
		set(op, "xchg", decodeXchgAX, 0)
	}
	set(0x98, "cbw", nil, 0)
	set(0x99, "cwd", nil, 0)
	set(0x9a, "call", decodeFar, 0)
	set(0x9b, "wait", nil, 0)
	set(0x9c, "pushf", nil, 0)
	set(0x9d, "popf", nil, 0)
	set(0x9e, "sahf", nil, 0)
	set(0x9f, "lahf", nil, 0)
	set(0xa0, "mov", decodeALMem, 0)
	set(0xa1, "mov", decodeAXMem, 0)
	set(0xa2, "mov", decodeMemAL, 0)
	set(0xa3, "mov", decodeMemAX, 0)
	set(0xa4, "movs", decodeString("movs"), 0)
	set(0xa5, "movs", decodeString("movs"), 0)
	set(0xa6, "cmps", decodeString("cmps"), 0)
	set(0xa7, "cmps", decodeString("cmps"), 0)
	set(0xa8, "test", decodeALD8, 0)
	set(0xa9, "test", decodeAXD16, 0)
	set(0xaa, "stos", decodeString("stos"), 0)
	set(0xab, "stos", decodeString("stos"), 0)
	set(0xac, "lods", decodeString("lods"), 0)
	set(0xad, "lods", decodeString("lods"), 0)
	set(0xae, "scas", decodeString("scas"), 0)
	set(0xaf, "scas", decodeString("scas"), 0)
	for op := byte(0xb0); op <= 0xbf; op++ {
		set(op, "mov", decodeRD, 0)
	}
	set(0xc0, "db", decodeDataByte, 0)
	set(0xc1, "db", decodeDataByte, 0)
	set(0xc2, "ret", decodeD16, 0)
	set(0xc3, "ret", nil, 0)
	set(0xc4, "les", decodeR16W, 0)
	set(0xc5, "lds", decodeR16W, 0)
	set(0xc6, "mov", decodeMovBD8, 0)
	set(0xc7, "mov", decodeMovWD16, 0)
	set(0xc8, "db", decodeDataByte, 0)
	set(0xc9, "db", decodeDataByte, 0)
	set(0xca, "retf", decodeD16, 0)
	set(0xcb, "retf", nil, 0)
	set(0xcc, "int", decodeInt3, 0)
	set(0xcd, "int", decodeD8, 0)
	set(0xce, "into", nil, 0)
	set(0xcf, "iret", nil, 0)
	set(0xd0, "", decodeBit1(Width8, PtrByte, tableDX), 0)
	set(0xd1, "", decodeBit1(Width16, PtrWord, tableDX), 0)
	set(0xd2, "", decodeBitCL(Width8, PtrByte, tableDX), 0)
	set(0xd3, "", decodeBitCL(Width16, PtrWord, tableDX), 0)
	set(0xd4, "aam", decodeAdjust, 0)
	set(0xd5, "aad", decodeAdjust, 0)
	set(0xd6, "db", decodeDataByte, 0)
	set(0xd7, "xlat", nil, 0)
	for op := byte(0xd8); op <= 0xdf; op++ {
		set(op, "esc", decodeEscape, 0)
	}
	set(0xe0, "loopne", decodeDisp, 0)
	set(0xe1, "loope", decodeDisp, 0)
	set(0xe2, "loop", decodeDisp, 0)
	set(0xe3, "jcxz", decodeDisp, 0)
	set(0xe4, "in", decodeALD8, 0)
	set(0xe5, "in", decodeAXD8, 0)
	set(0xe6, "out", decodeD8AL, 0)
	set(0xe7, "out", decodeD8AX, 0)
	set(0xe8, "call", decodeDisp16, 0)
	set(0xe9, "jmp", decodeDisp16, 0)
	set(0xea, "jmp", decodeFar, 0)
	set(0xeb, "jmp", decodeDisp, 0)
	set(0xec, "in", decodePortDX, 0)
	set(0xed, "in", decodePortDX, 0)
	set(0xee, "out", decodePortDX, 0)
	set(0xef, "out", decodePortDX, 0)
	set(0xf0, "lock", nil, flagPrefix)
	set(0xf1, "", decodeBiosCall, 0)
	set(0xf2, "repnz", nil, flagPrefix)
	set(0xf3, "repz", nil, flagPrefix)
	set(0xf4, "hlt", nil, 0)
	set(0xf5, "cmc", nil, 0)
	set(0xf6, "", decodeF6, 0)
	set(0xf7, "", decodeF7, 0)
	set(0xf8, "clc", nil, 0)
	set(0xf9, "stc", nil, 0)
	set(0xfa, "cli", nil, 0)
	set(0xfb, "sti", nil, 0)
	set(0xfc, "cld", nil, 0)
	set(0xfd, "std", nil, 0)
	set(0xfe, "", func(s *decodeState) error { return decodeGroupedUnary(s, Width8, PtrByte, tableFE) }, 0)
	set(0xff, "", decodeFF, 0)
	return t
}()

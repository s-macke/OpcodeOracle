package disasm

import (
	"errors"
	"fmt"
)

var ErrIllegalOpcode = errors.New("illegal opcode")

// IllegalOpcodeError provides details about an illegal opcode encountered during disassembly.
type IllegalOpcodeError struct {
	Address uint16
	Opcode  byte
}

func (e *IllegalOpcodeError) Error() string {
	return fmt.Sprintf("illegal opcode $%02X at $%04X", e.Opcode, e.Address)
}

func (e *IllegalOpcodeError) Unwrap() error {
	return ErrIllegalOpcode
}

var ErrMidInstruction = errors.New("address is mid-instruction")

// MidInstructionError provides details when disassembly is attempted at an operand byte.
type MidInstructionError struct {
	Address uint16
}

func (e *MidInstructionError) Error() string {
	return fmt.Sprintf("cannot disassemble at $%04X: address is mid-instruction (operand byte)", e.Address)
}

func (e *MidInstructionError) Unwrap() error {
	return ErrMidInstruction
}

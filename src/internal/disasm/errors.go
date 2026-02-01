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

var ErrAddressOutOfRange = errors.New("address out of range")

// AddressOutOfRangeError provides details when an address is outside binary bounds.
type AddressOutOfRangeError struct {
	Address    uint16
	RangeStart uint16
	RangeEnd   uint16
}

func (e *AddressOutOfRangeError) Error() string {
	return fmt.Sprintf("address $%04X is out of range ($%04X-$%04X)",
		e.Address, e.RangeStart, e.RangeEnd)
}

func (e *AddressOutOfRangeError) Unwrap() error {
	return ErrAddressOutOfRange
}

var ErrInvalidRange = errors.New("invalid address range")

// InvalidRangeError provides details when start > end.
type InvalidRangeError struct {
	Start uint16
	End   uint16
}

func (e *InvalidRangeError) Error() string {
	return fmt.Sprintf("start address $%04X is greater than end address $%04X",
		e.Start, e.End)
}

func (e *InvalidRangeError) Unwrap() error {
	return ErrInvalidRange
}

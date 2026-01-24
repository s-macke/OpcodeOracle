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

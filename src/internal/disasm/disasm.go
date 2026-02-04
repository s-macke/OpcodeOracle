// Package disasm provides a MOS6502 disassembler.
package disasm

import (
	"strings"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

// Disassembler formats MOS6502 machine code as assembly text.
type Disassembler interface {
	// Disassemble formats the address range [start, end) as assembly text.
	Disassemble(start, end uint16) (string, error)
}

// disassembler implements the Disassembler interface.
type disassembler struct {
	state      *state.State
	boundaries analysis.InstructionBoundaries // optional, may be nil
}

const (
	instructionCommentCol = 38
	dataAsciiCol          = 95
)

// NewDisassembler creates a disassembler that reads from the given state.
// The boundaries parameter is optional and provides instruction boundary information
// from flow analysis. When provided, the disassembler will error if asked to
// disassemble at an operand byte address.
func NewDisassembler(s *state.State, boundaries analysis.InstructionBoundaries) Disassembler {
	return &disassembler{state: s, boundaries: boundaries}
}

// Disassemble formats the address range [start, end) as assembly text.
func (d *disassembler) Disassemble(start, end uint16) (string, error) {
	// Calculate binary bounds
	origin := d.state.Binary.Start()
	binaryEnd := d.state.Binary.End()

	// Validate start address
	if start < origin || start > binaryEnd {
		return "", &AddressOutOfRangeError{
			Address:    start,
			RangeStart: origin,
			RangeEnd:   binaryEnd,
		}
	}

	// Validate end address (end is exclusive, so check end-1)
	if end > start && (end-1 < origin || end-1 > binaryEnd) {
		return "", &AddressOutOfRangeError{
			Address:    end - 1,
			RangeStart: origin,
			RangeEnd:   binaryEnd,
		}
	}

	// Validate range order
	if start > end {
		return "", &InvalidRangeError{Start: start, End: end}
	}

	var sb strings.Builder
	addr := start
	needsBlankLine := false

	for addr < end {
		regionType := d.state.Regions.At(addr)

		if regionType == regions.RegionCode {
			output, size, err := d.formatCodeAt(addr, &needsBlankLine)
			if err != nil {
				return sb.String(), err
			}
			sb.WriteString(output)
			addr += uint16(size)
		} else {
			output, size := d.formatDataAt(addr, end, &needsBlankLine)
			sb.WriteString(output)
			addr += uint16(size)
		}
	}

	return sb.String(), nil
}

// getSymbol returns the symbol at addr, if any.
func (d *disassembler) getSymbol(addr uint16) (symbols.Symbol, bool) {
	return d.state.Symbols.At(addr)
}

func (d *disassembler) labelAt(addr uint16) string {
	if sym, ok := d.getSymbol(addr); ok {
		return sym.Name
	}
	return ""
}

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
	// Left column layout for address/label prefix in code/data/unknown output.
	addressWidth    = 5                                  // "$XXXX"
	labelFieldWidth = 18                                 // "label:" padded field
	leftColumnWidth = addressWidth + 1 + labelFieldWidth // address + separator + label field

	// Data row layout: "$XXXX ... .BYTE $XX,...,$XX  ; \"ASCII\""
	dataRowMaxBytes        = 16
	dataByteTokenWidth     = 3 // "$XX"
	dataByteSeparatorWidth = 1 // ","
	minCommentGapWidth     = 2 // padToColumn minimum spacing before ';'

	maxDataHexWidth = dataRowMaxBytes*dataByteTokenWidth + (dataRowMaxBytes-1)*dataByteSeparatorWidth

	instructionCommentCol = 38
	dataAsciiCol          = leftColumnWidth + len(".BYTE ") + maxDataHexWidth + minCommentGapWidth

	codeInstrIndent = 4
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
	// Validate range order
	if start > end {
		return "", &InvalidRangeError{Start: start, End: end}
	}

	var sb strings.Builder
	addr := start
	needsBlankLine := false

	for addr < end {
		if !d.hasBinaryByte(addr) {
			output, size := d.formatUnknownSpan(addr, end, &needsBlankLine)
			sb.WriteString(output)
			addr += uint16(size)
			continue
		}

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

func (d *disassembler) hasBinaryByte(addr uint16) bool {
	offset := int(addr) - int(d.state.Binary.Origin)
	return offset >= 0 && offset < len(d.state.Binary.Data)
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

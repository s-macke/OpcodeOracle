package disasm

import (
	"fmt"

	"opcodeoracle/internal/asm"
	"opcodeoracle/internal/symbols"
)

// getOperandAddress extracts the target address from an instruction operand.
// Returns the address and true if the addressing mode uses a 16-bit address, false otherwise.
func getOperandAddress(mode asm.AddrMode, operand []byte) (uint16, bool) {
	switch mode {
	case asm.AddrAbsolute, asm.AddrAbsoluteX, asm.AddrAbsoluteY, asm.AddrIndirect:
		if len(operand) < 2 {
			return 0, false
		}
		return uint16(operand[0]) | uint16(operand[1])<<8, true
	case asm.AddrZeroPage, asm.AddrZeroPageX, asm.AddrZeroPageY, asm.AddrIndexedIndirect, asm.AddrIndirectIndexed:
		if len(operand) < 1 {
			return 0, false
		}
		return uint16(operand[0]), true
	}
	return 0, false
}

// formatOperandWithSymbol formats the operand and returns an optional symbol for comments.
func (d *disassembler) formatOperandWithSymbol(def asm.OpcodeDef, operand []byte, pc uint16) (string, string) {
	if def.Mode == asm.AddrRelative && len(operand) >= 1 {
		// Always show numeric target address (symbol will be in comment)
		target := calculateBranchTarget(pc, operand[0])
		sym, ok := d.getSymbolOfTypes(target, symbols.SymbolLabel, symbols.SymbolSubroutine, symbols.SymbolEntry)
		if !ok {
			return fmt.Sprintf(" $%04X", target), ""
		}
		return fmt.Sprintf(" $%04X", target), "Branch to " + sym.Name
	}

	// JSR/JMP with absolute addressing: keep numeric operand, put symbol in comment.
	if def.Mode == asm.AddrAbsolute && (def.Op == asm.JSR || def.Op == asm.JMP) && len(operand) >= 2 {
		target := uint16(operand[0]) | uint16(operand[1])<<8
		allowed := []symbols.SymbolType{symbols.SymbolLabel, symbols.SymbolSubroutine, symbols.SymbolEntry}
		if def.Op == asm.JSR {
			allowed = []symbols.SymbolType{symbols.SymbolSubroutine, symbols.SymbolEntry}
		}
		sym, ok := d.getSymbolOfTypes(target, allowed...)
		if !ok {
			return def.FormatOperand(operand), ""
		}
		if def.Op == asm.JSR {
			return def.FormatOperand(operand), "Call " + sym.Name
		}
		return def.FormatOperand(operand), "Jump to " + sym.Name
	}
	operandStr := def.FormatOperand(operand)

	if opAddr, ok := getOperandAddress(def.Mode, operand); ok {
		if sym, found := d.getSymbolOfTypes(opAddr, symbols.SymbolByte, symbols.SymbolLabel, symbols.SymbolSubroutine, symbols.SymbolEntry); found {
			return operandStr, sym.Name
		}
	}
	return operandStr, ""
}

// calculateBranchTarget computes the target address for a relative branch.
func calculateBranchTarget(pc uint16, operand byte) uint16 {
	// Branch is relative to the address AFTER the branch instruction (PC + 2)
	nextPC := pc + 2
	if operand > 0x7F {
		// Negative offset (two's complement)
		return nextPC - uint16(256-int(operand))
	}
	return nextPC + uint16(operand)
}

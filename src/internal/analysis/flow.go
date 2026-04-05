// Package analysis provides 6502 code analysis algorithms.
package analysis

import (
	"fmt"
	"sort"

	"opcodeoracle/internal/asm"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
	"opcodeoracle/internal/xrefs"
)

// UpdateFlags controls which state components are updated during analysis.
type UpdateFlags uint8

const (
	UpdateRegions UpdateFlags = 1 << iota // Mark code regions
	UpdateSymbols                         // Add auto-generated symbols
	UpdateXRefs                           // Add cross-references

	// Convenience combinations
	UpdateAll       = UpdateRegions | UpdateSymbols | UpdateXRefs
	UpdateXRefsOnly = UpdateXRefs
)

// InstructionClass categorizes instructions by their control flow behavior.
type InstructionClass int

const (
	ClassSequential InstructionClass = iota // Normal instruction, continues to next
	ClassJump                               // Unconditional jump (JMP)
	ClassBranch                             // Conditional branch (BCC, BCS, BEQ, etc.)
	ClassCall                               // Subroutine call (JSR)
	ClassReturn                             // Return from subroutine (RTS, RTI)
	ClassTerminal                           // Terminal instruction (BRK)
	ClassIllegal                            // Illegal/undocumented opcode
)

// Analyzer performs control flow analysis on 6502 code.
type Analyzer struct {
	state        *state.State
	flags        UpdateFlags
	visited      map[uint16]bool
	operandBytes map[uint16]bool // tracks operand byte addresses (2nd/3rd bytes of instructions)
	queue        []uint16
}

type Analyzer struct {
	visited      map[uint32]bool
	operandBytes map[uint32]bool // tracks operand byte addresses
	queue        []uint32
}

type seedMode uint8

const (
	seedEntryPointsOnly seedMode = iota
	seedAllKnownCode
)

// NewAnalyzer creates a new flow analyzer for the given state.
// The flags parameter controls which state components are updated during analysis.
func NewAnalyzer(s *state.State, flags UpdateFlags) *Analyzer {
	return &Analyzer{
		state:        s,
		flags:        flags,
		visited:      make(map[uint16]bool),
		operandBytes: make(map[uint16]bool),
		queue:        make([]uint16, 0),
	}
}

// Analyze performs flow analysis starting from all entry points and known subroutines.
// It populates the RegionTable, SymbolTable, and XRefTable.
func (a *Analyzer) Analyze() error {
	a.ensureXRefs()
	a.seed(seedAllKnownCode)
	return a.run()
}

func (a *Analyzer) AnalyzeFromEntryPoints() error {
	a.ensureXRefs()
	a.seed(seedEntryPointsOnly)
	return a.run()
}

func (a *Analyzer) seed(mode seedMode) {
	// Seed the queue with all entry points.
	for _, ep := range a.state.EntryPoints {
		a.seedEntryPoint(ep)
	}
	for _, addr := range a.state.ExtraCodeAddresses {
		if contains16(a.state.EntryPoints, addr) {
			continue
		}
		a.enqueue(addr)
	}

	if mode != seedAllKnownCode {
		return
	}

	// Also seed from existing subroutine and label symbols in the symbol table.
	// This allows user-defined or imported symbols to drive analysis.
	for _, addr := range a.symbolSeedAddresses() {
		a.enqueue(addr)
	}

	// Also seed from existing code regions.
	// This ensures previously-identified code is re-analyzed (e.g., for xref rebuilding).
	for _, region := range a.state.Regions.Regions() {
		if region.Type == regions.RegionCode {
			a.enqueue(region.Start)
		}
	}
}

func (a *Analyzer) seedEntryPoint(ep uint16) {
	a.enqueue(ep)
	// Add entry point symbol if not already present.
	if a.flags&UpdateSymbols != 0 {
		a.state.Symbols.Add(ep, symbols.Symbol{
			Name:   fmt.Sprintf("ENTRY_%04X", ep),
			Type:   symbols.SymbolEntry,
			Source: symbols.SourceAuto,
		})
	}
}

func contains16(addrs []uint16, addr uint16) bool {
	for _, candidate := range addrs {
		if candidate == addr {
			return true
		}
	}
	return false
}

// symbolSeedAddresses returns symbol addresses that should seed analysis in deterministic order.
func (a *Analyzer) symbolSeedAddresses() []uint16 {
	var addrs []uint16

	for addr, sym := range a.state.Symbols.All() {
		if sym.Type == symbols.SymbolSubroutine || sym.Type == symbols.SymbolLabel || sym.Type == symbols.SymbolEntry {
			addrs = append(addrs, addr)
		}
	}

	sort.Slice(addrs, func(i, j int) bool {
		return addrs[i] < addrs[j]
	})

	return addrs
}

// AnalyzeFrom performs flow analysis starting from a single address.
// It does not use entry points or existing symbols - only the given address.
func (a *Analyzer) AnalyzeFrom(addr uint16) error {
	a.ensureXRefs()
	a.enqueue(addr)
	return a.run()
}

// ensureXRefs ensures the XRefs table exists.
func (a *Analyzer) ensureXRefs() {
	if a.state.XRefs == nil {
		a.state.XRefs = xrefs.NewTable()
	}
}

// run processes the analysis queue until empty.
func (a *Analyzer) run() error {
	for len(a.queue) > 0 {
		addr := a.queue[0]
		a.queue = a.queue[1:]

		if err := a.process(addr); err != nil {
			// Log warning but continue analysis
			continue
		}
	}

	return nil
}

// process handles a single address in the analysis queue.
func (a *Analyzer) process(addr uint16) error {
	// Skip if already visited
	if a.visited[addr] {
		return nil
	}
	if a.isHardDataLock(addr) {
		return nil
	}

	// Check if address is within binary bounds
	if !a.inBounds(addr) {
		return fmt.Errorf("address %04X out of bounds", addr)
	}

	// Read opcode
	opcode, err := a.state.Binary.ReadByte(addr)
	if err != nil {
		return err
	}

	// Get instruction definition
	def := asm.Opcodes[opcode]

	// Check for illegal opcode
	if def.IsIllegal() {
		// Don't mark as visited, don't mark as code, just skip
		return fmt.Errorf("illegal opcode %02X at %04X", opcode, addr)
	}

	// Verify we can read the full instruction
	instrEnd := addr + uint16(def.Size) - 1
	if !a.inBounds(instrEnd) {
		return fmt.Errorf("instruction at %04X extends beyond binary", addr)
	}
	for scan := addr; ; scan++ {
		if a.isHardDataLock(scan) {
			return fmt.Errorf("instruction at %04X intersects locked data region", addr)
		}
		if scan == instrEnd {
			break
		}
	}

	// Mark address as visited
	a.visited[addr] = true

	// Mark operand bytes (2nd and 3rd bytes of multi-byte instructions)
	for i := uint16(1); i < uint16(def.Size); i++ {
		a.operandBytes[addr+i] = true
	}

	// Mark instruction bytes as code
	if a.flags&UpdateRegions != 0 {
		a.state.Regions.SetWithSource(addr, instrEnd, regions.RegionCode, regions.RegionSourceAuto)
	}

	// Classify and handle the instruction
	class := classify(def)

	switch class {
	case ClassSequential:
		// Add data access xrefs for memory-accessing instructions
		if hasStaticAddress(def.Mode) {
			if target, ok := a.readOperandAddress(addr, def.Mode); ok {
				if a.flags&UpdateXRefs != 0 {
					if isMemoryRead(def.Op) {
						a.state.XRefs.Add(addr, target, xrefs.XRefRead)
					} else if isMemoryWrite(def.Op) {
						a.state.XRefs.Add(addr, target, xrefs.XRefWrite)
					} else if isReadModifyWrite(def.Op) {
						a.state.XRefs.Add(addr, target, xrefs.XRefRead)
						a.state.XRefs.Add(addr, target, xrefs.XRefWrite)
					}
				}
				if a.flags&UpdateSymbols != 0 {
					a.addHardwareSymbol(target)
				}
			}
		}
		// Continue to next instruction
		a.enqueue(addr + uint16(def.Size))

	case ClassJump:
		if def.Mode == asm.AddrAbsolute {
			// JMP absolute - follow target
			target, err := a.state.Binary.ReadWord(addr + 1)
			if err != nil {
				return err
			}
			if a.flags&UpdateXRefs != 0 {
				a.state.XRefs.Add(addr, target, xrefs.XRefJump)
			}
			a.addLabel(target)
			a.enqueue(target)
		}
		// JMP indirect - cannot follow statically, just mark as code (already done)

	case ClassBranch:
		// Read branch offset
		offset, err := a.state.Binary.ReadByte(addr + 1)
		if err != nil {
			return err
		}
		target := calculateBranchTarget(addr, offset)
		if a.flags&UpdateXRefs != 0 {
			a.state.XRefs.Add(addr, target, xrefs.XRefBranch)
		}
		a.addLabel(target)
		// Enqueue both branch target and fall-through
		a.enqueue(target)
		a.enqueue(addr + uint16(def.Size))

	case ClassCall:
		// JSR - read target address
		target, err := a.state.Binary.ReadWord(addr + 1)
		if err != nil {
			return err
		}
		if a.flags&UpdateXRefs != 0 {
			a.state.XRefs.Add(addr, target, xrefs.XRefCall)
		}
		a.addSubroutine(target)
		// Enqueue both subroutine and return address
		a.enqueue(target)
		a.enqueue(addr + uint16(def.Size))

	case ClassReturn, ClassTerminal:
		// End of path - don't enqueue anything

	case ClassIllegal:
		// Already handled above
	}

	return nil
}

// classify returns the instruction class for the given opcode definition.
func classify(def asm.OpcodeDef) InstructionClass {
	switch def.Op {
	case asm.JMP:
		return ClassJump
	case asm.BCC, asm.BCS, asm.BEQ, asm.BMI, asm.BNE, asm.BPL, asm.BVC, asm.BVS:
		return ClassBranch
	case asm.JSR:
		return ClassCall
	case asm.RTS, asm.RTI:
		return ClassReturn
	case asm.BRK:
		return ClassTerminal
	case asm.MnemonicIllegal:
		return ClassIllegal
	default:
		return ClassSequential
	}
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

// inBounds checks if an address is within the binary's address range.
func (a *Analyzer) inBounds(addr uint16) bool {
	origin := a.state.Binary.Start()
	end := a.state.Binary.End()
	return addr >= origin && addr <= end
}

// enqueue adds an address to the analysis queue if not already visited.
func (a *Analyzer) enqueue(addr uint16) {
	if !a.visited[addr] && a.inBounds(addr) && !a.isHardDataLock(addr) {
		a.queue = append(a.queue, addr)
	}
}

func (a *Analyzer) isHardDataLock(addr uint16) bool {
	return a.state.Regions.At(addr) == regions.RegionData &&
		a.state.Regions.SourceAt(addr) != regions.RegionSourceAuto
}

// addLabel adds an auto-generated label symbol at the target address.
func (a *Analyzer) addLabel(addr uint16) {
	if a.flags&UpdateSymbols != 0 {
		a.state.Symbols.Add(addr, symbols.Symbol{
			Name:   fmt.Sprintf("L_%04X", addr),
			Type:   symbols.SymbolLabel,
			Source: symbols.SourceAuto,
		})
	}
}

// addSubroutine adds an auto-generated subroutine symbol at the target address.
func (a *Analyzer) addSubroutine(addr uint16) {
	if a.flags&UpdateSymbols != 0 {
		a.state.Symbols.Add(addr, symbols.Symbol{
			Name:   fmt.Sprintf("SUB_%04X", addr),
			Type:   symbols.SymbolSubroutine,
			Source: symbols.SourceAuto,
		})
	}
}

// addHardwareSymbol adds a C64 hardware symbol at the target address if known.
func (a *Analyzer) addHardwareSymbol(addr uint16) {
	if a.flags&UpdateSymbols == 0 {
		return
	}
	if name, ok := asm.LookupC64Hardware(addr); ok {
		a.state.Symbols.Add(addr, symbols.Symbol{
			Name:   name,
			Type:   symbols.SymbolByte,
			Source: symbols.SourceAuto,
		})
	}
}

// hasStaticAddress returns true if the addressing mode has a statically-known address operand.
func hasStaticAddress(mode asm.AddrMode) bool {
	switch mode {
	case asm.AddrZeroPage, asm.AddrZeroPageX, asm.AddrZeroPageY:
		return true
	case asm.AddrAbsolute, asm.AddrAbsoluteX, asm.AddrAbsoluteY:
		return true
	}
	return false
}

// readOperandAddress reads the operand address for zero page or absolute addressing modes.
func (a *Analyzer) readOperandAddress(addr uint16, mode asm.AddrMode) (uint16, bool) {
	switch mode {
	case asm.AddrZeroPage, asm.AddrZeroPageX, asm.AddrZeroPageY:
		b, err := a.state.Binary.ReadByte(addr + 1)
		if err != nil {
			return 0, false
		}
		return uint16(b), true
	case asm.AddrAbsolute, asm.AddrAbsoluteX, asm.AddrAbsoluteY:
		w, err := a.state.Binary.ReadWord(addr + 1)
		if err != nil {
			return 0, false
		}
		return w, true
	}
	return 0, false
}

// isMemoryRead returns true if the instruction reads from memory.
func isMemoryRead(m asm.Mnemonic) bool {
	switch m {
	case asm.LDA, asm.LDX, asm.LDY:
		return true
	case asm.ADC, asm.SBC, asm.AND, asm.EOR, asm.ORA:
		return true
	case asm.CMP, asm.CPX, asm.CPY, asm.BIT:
		return true
	}
	return false
}

// isMemoryWrite returns true if the instruction writes to memory.
func isMemoryWrite(m asm.Mnemonic) bool {
	switch m {
	case asm.STA, asm.STX, asm.STY:
		return true
	}
	return false
}

// isReadModifyWrite returns true if the instruction reads, modifies, and writes memory.
func isReadModifyWrite(m asm.Mnemonic) bool {
	switch m {
	case asm.INC, asm.DEC, asm.ASL, asm.LSR, asm.ROL, asm.ROR:
		return true
	}
	return false
}

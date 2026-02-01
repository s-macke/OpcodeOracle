package analysis

import "sort"

// InstructionBoundaries provides access to instruction boundary information.
type InstructionBoundaries interface {
	// IsInstructionAt returns true if the given address is the start of an instruction.
	IsInstructionAt(addr uint16) bool

	// IsInstructionDataAt returns true if the given address is an operand byte of an instruction.
	IsInstructionDataAt(addr uint16) bool

	// InstructionAddresses returns all addresses that are instruction starts.
	InstructionAddresses() []uint16
}

// IsInstructionAt returns true if addr is the start of an analyzed instruction.
func (a *Analyzer) IsInstructionAt(addr uint16) bool {
	return a.visited[addr]
}

// IsInstructionDataAt returns true if addr is an operand byte of an analyzed instruction.
func (a *Analyzer) IsInstructionDataAt(addr uint16) bool {
	return a.operandBytes[addr]
}

// InstructionAddresses returns all instruction start addresses found during analysis.
// The returned slice is sorted in ascending order.
func (a *Analyzer) InstructionAddresses() []uint16 {
	addrs := make([]uint16, 0, len(a.visited))
	for addr := range a.visited {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool { return addrs[i] < addrs[j] })
	return addrs
}

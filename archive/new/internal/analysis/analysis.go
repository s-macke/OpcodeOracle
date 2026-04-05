package analysis

import (
	"fmt"

	"opcodeoracle/internal/asm/x86"
	binfile "opcodeoracle/internal/binary"
)

type Analyzer struct {
	visited      map[uint32]bool
	operandBytes map[uint32]bool
	queue        []x86.FarAddress
}

type Result struct {
	Instructions map[uint32]x86.Instruction
	Visited      map[uint32]bool
	OperandBytes map[uint32]bool
	Unresolved   []x86.FarAddress
	DecodeStops  []DecodeStop
}

type DecodeStop struct {
	Address x86.FarAddress
	Err     error
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		visited:      make(map[uint32]bool),
		operandBytes: make(map[uint32]bool),
	}
}

func (a *Analyzer) Analyze(bin binfile.Binary) (Result, error) {
	a.reset(bin.EntryPoints)

	result := Result{
		Instructions: make(map[uint32]x86.Instruction),
		Visited:      a.visited,
		OperandBytes: a.operandBytes,
	}

	dec := x86.NewDecoder()
	unresolvedSeen := make(map[uint32]bool)

	for len(a.queue) > 0 {
		addr := a.dequeue()
		linear := addr.Linear()
		if a.visited[linear] || a.operandBytes[linear] {
			continue
		}

		data, err := bin.DataAt(addr)
		if err != nil {
			continue
		}

		inst, err := dec.Decode(data, addr)
		if err != nil {
			result.DecodeStops = append(result.DecodeStops, DecodeStop{
				Address: addr,
				Err:     err,
			})
			continue
		}
		if inst.Mnemonic == "db" {
			result.DecodeStops = append(result.DecodeStops, DecodeStop{
				Address: addr,
				Err:     fmt.Errorf("fallback db decode: %s", inst.Text),
			})
			continue
		}
		if hasPrefix(inst, x86.PrefixOp32) {
			result.DecodeStops = append(result.DecodeStops, DecodeStop{
				Address: addr,
				Err:     fmt.Errorf("unsupported operand-size override prefix 66: %s", inst.Text),
			})
			continue
		}

		a.visited[linear] = true
		result.Instructions[linear] = inst
		a.markOperandBytes(inst)

		if inst.Target != nil && inst.Target.Indirect {
			if !unresolvedSeen[linear] {
				result.Unresolved = append(result.Unresolved, addr)
				unresolvedSeen[linear] = true
			}
		}

		for _, next := range nextAddresses(inst) {
			nextLinear := next.Linear()
			if a.visited[nextLinear] || a.operandBytes[nextLinear] {
				continue
			}
			if _, err := bin.DataAt(next); err != nil {
				continue
			}
			a.enqueue(next)
		}
	}

	return result, nil
}

func (a *Analyzer) reset(entryPoints []x86.FarAddress) {
	a.visited = make(map[uint32]bool)
	a.operandBytes = make(map[uint32]bool)
	a.queue = a.queue[:0]
	for _, ep := range entryPoints {
		a.enqueue(ep)
	}
}

func (a *Analyzer) enqueue(addr x86.FarAddress) {
	a.queue = append(a.queue, addr)
}

func (a *Analyzer) dequeue() x86.FarAddress {
	addr := a.queue[0]
	a.queue = a.queue[1:]
	return addr
}

func (a *Analyzer) markOperandBytes(inst x86.Instruction) {
	for i := uint16(1); i < uint16(inst.Length); i++ {
		addr := x86.NewFarAddress(inst.Address.Segment, inst.Address.Offset+i)
		a.operandBytes[addr.Linear()] = true
	}
}

func nextAddresses(inst x86.Instruction) []x86.FarAddress {
	switch inst.Flow {
	case x86.FlowNone:
		return []x86.FarAddress{inst.NextAddress}
	case x86.FlowConditionalJump:
		next := []x86.FarAddress{inst.NextAddress}
		if target, ok := directTarget(inst); ok {
			next = append(next, target)
		}
		return next
	case x86.FlowCall:
		next := []x86.FarAddress{inst.NextAddress}
		if target, ok := directTarget(inst); ok {
			next = append(next, target)
		}
		return next
	case x86.FlowJump:
		if target, ok := directTarget(inst); ok {
			return []x86.FarAddress{target}
		}
		return nil
	case x86.FlowInterrupt:
		return []x86.FarAddress{inst.NextAddress}
	case x86.FlowReturn:
		return nil
	default:
		return nil
	}
}

func directTarget(inst x86.Instruction) (x86.FarAddress, bool) {
	if inst.Target == nil || inst.Target.Indirect {
		return x86.FarAddress{}, false
	}
	if inst.Target.Far != nil {
		return *inst.Target.Far, true
	}
	if inst.Target.Near != nil {
		return x86.NewFarAddress(inst.Address.Segment, *inst.Target.Near), true
	}
	return x86.FarAddress{}, false
}

func hasPrefix(inst x86.Instruction, prefix x86.Prefix) bool {
	for _, p := range inst.Prefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

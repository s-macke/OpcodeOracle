package x86

import (
	"fmt"
	"strings"
)

func (inst Instruction) AssemblerString() string {
	mnemonic := string(inst.Mnemonic)
	if mnemonic == "" {
		mnemonic = "db"
	}

	var prefixParts []string
	for _, p := range inst.Prefixes {
		if p == PrefixES || p == PrefixCS || p == PrefixSS || p == PrefixDS || p == PrefixFS || p == PrefixGS {
			continue
		}
		prefixParts = append(prefixParts, string(p))
	}

	if len(prefixParts) > 0 {
		mnemonic = strings.Join(append(prefixParts, mnemonic), " ")
	}

	if len(inst.Operands) == 0 {
		return mnemonic
	}

	ops := make([]string, 0, len(inst.Operands))
	for _, op := range inst.Operands {
		ops = append(ops, formatAssemblyOperand(op))
	}
	return mnemonic + " " + strings.Join(ops, ", ")
}

func formatAssemblyOperand(op Operand) string {
	switch op.Kind {
	case OperandRegister:
		return string(op.Register)
	case OperandImmediate:
		return formatAssemblyImmediate(op.Immediate, op.Width)
	case OperandMemory:
		return formatAssemblyMemory(op.Memory)
	case OperandRelative:
		return fmt.Sprintf("%04x", op.Relative.Resolved)
	case OperandFar:
		return op.Far.String()
	case OperandText:
		return op.Text
	default:
		return op.Text
	}
}

func formatAssemblyImmediate(v uint16, width OperandWidth) string {
	if width == Width16 {
		return fmt.Sprintf("%04x", v)
	}
	return fmt.Sprintf("%02x", byte(v))
}

func formatAssemblyMemory(mem *MemoryRef) string {
	var b strings.Builder
	if mem.Pointer != PtrNone {
		b.WriteString(string(mem.Pointer))
		b.WriteByte(' ')
	}
	if mem.SegmentOverride != nil {
		b.WriteString(string(*mem.SegmentOverride))
		b.WriteByte(':')
	}
	b.WriteByte('[')
	switch {
	case mem.DirectAddress != nil:
		b.WriteString(fmt.Sprintf("%04x", *mem.DirectAddress))
	case mem.Base != BaseDirect:
		b.WriteString(string(mem.Base))
		if mem.Displacement != 0 {
			if mem.Displacement < 0 {
				b.WriteString(fmt.Sprintf("-%x", -mem.Displacement))
			} else {
				b.WriteString(fmt.Sprintf("+%x", mem.Displacement))
			}
		}
	default:
		if mem.Displacement < 0 {
			b.WriteString(fmt.Sprintf("-%x", -mem.Displacement))
		} else {
			b.WriteString(fmt.Sprintf("%x", mem.Displacement))
		}
	}
	b.WriteByte(']')
	return b.String()
}

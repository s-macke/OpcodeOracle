package x86

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (inst Instruction) DetailsString() string {
	var b strings.Builder
	writeInstructionDetails(&b, inst)
	return b.String()
}

func writeInstructionDetails(w io.Writer, inst Instruction) {
	fmt.Fprintf(w, "Text:        %s\n", inst.Text)
	fmt.Fprintf(w, "Address:     %s\n", inst.Address.String())
	fmt.Fprintf(w, "NextAddress: %s\n", inst.NextAddress.String())
	fmt.Fprintf(w, "Segment:     0x%04x\n", inst.Address.Segment)
	fmt.Fprintf(w, "Offset:      0x%04x\n", inst.Address.Offset)
	fmt.Fprintf(w, "NextOffset:  0x%04x\n", inst.NextAddress.Offset)
	fmt.Fprintf(w, "Length:      %d\n", inst.Length)
	fmt.Fprintf(w, "Bytes:       %s\n", formatDetailBytes(inst.Bytes))
	fmt.Fprintf(w, "Opcode:      0x%02x\n", inst.Opcode)
	fmt.Fprintf(w, "Mnemonic:    %s\n", inst.Mnemonic)
	fmt.Fprintf(w, "Addressing:  %s\n", inst.Addressing)
	fmt.Fprintf(w, "Flow:        %s\n", inst.Flow)
	if len(inst.Prefixes) > 0 {
		fmt.Fprintf(w, "Prefixes:    %s\n", joinDetailPrefixes(inst.Prefixes))
	}
	if inst.Target != nil {
		fmt.Fprintf(w, "Target:      %s\n", formatDetailTargetForInstruction(inst, inst.Target))
	}
	for i, op := range inst.Operands {
		fmt.Fprintf(w, "Operand[%d]:  %s\n", i, formatDetailOperand(op))
	}
}

func formatDetailBytes(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02x", b))
	}
	return strings.Join(parts, " ")
}

func joinDetailPrefixes(prefixes []Prefix) string {
	parts := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		parts = append(parts, string(p))
	}
	return strings.Join(parts, ", ")
}

func formatDetailTargetForInstruction(inst Instruction, target *Target) string {
	switch {
	case target.Far != nil:
		return fmt.Sprintf("far %s indirect=%t", target.Far.String(), target.Indirect)
	case target.Near != nil:
		return fmt.Sprintf("near %s indirect=%t", NewFarAddress(inst.Address.Segment, *target.Near).String(), target.Indirect)
	default:
		return fmt.Sprintf("%s indirect=%t", target.Kind, target.Indirect)
	}
}

func formatDetailOperand(op Operand) string {
	switch op.Kind {
	case OperandRegister:
		return "register " + string(op.Register)
	case OperandImmediate:
		return "immediate " + strconv.FormatUint(uint64(op.Immediate), 16)
	case OperandRelative:
		return fmt.Sprintf("relative target=%04x delta=%d", op.Relative.Resolved, op.Relative.Delta)
	case OperandFar:
		return "far " + op.Far.String()
	case OperandMemory:
		if op.Memory.DirectAddress != nil {
			return fmt.Sprintf("memory direct=%04x size=%s", *op.Memory.DirectAddress, op.Memory.Size)
		}
		return fmt.Sprintf("memory base=%s disp=%d size=%s", op.Memory.Base, op.Memory.Displacement, op.Memory.Size)
	case OperandText:
		return "text " + op.Text
	default:
		return op.Text
	}
}

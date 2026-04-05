package disasm

import (
	"fmt"
	"strings"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/asm/x86"
	binfile "opcodeoracle/internal/binary"
)

const maxDataBytesPerLine = 16

type LineKind string

const (
	LineCode LineKind = "code"
	LineData LineKind = "data"
)

type Line struct {
	Kind    LineKind
	Address x86.FarAddress
	Bytes   []byte
	Text    string
	Comment string
}

type Disassembler struct{}

func NewDisassembler() *Disassembler {
	return &Disassembler{}
}

func (d *Disassembler) Disassemble(bin binfile.Binary, result analysis.Result) ([]Line, error) {
	starts := make(map[uint32]x86.Instruction, len(result.Instructions))
	codeBytes := make(map[uint32]bool)
	for linear, inst := range result.Instructions {
		starts[linear] = inst
		for i := 0; i < int(inst.Length); i++ {
			codeBytes[inst.Address.Linear()+uint32(i)] = true
		}
	}

	lines := make([]Line, 0, len(result.Instructions))
	for index := 0; index < len(bin.Data); {
		addr := offsetAddress(bin.Origin, index)
		linear := addr.Linear()

		if inst, ok := starts[linear]; ok {
			lines = append(lines, Line{
				Kind:    LineCode,
				Address: inst.Address,
				Bytes:   append([]byte(nil), inst.Bytes...),
				Text:    inst.Text,
			})
			index += int(inst.Length)
			continue
		}

		end := index
		for end < len(bin.Data) && end-index < maxDataBytesPerLine {
			nextAddr := offsetAddress(bin.Origin, end)
			nextLinear := nextAddr.Linear()
			if _, ok := starts[nextLinear]; ok {
				break
			}
			if codeBytes[nextLinear] {
				break
			}
			end++
		}
		if end == index {
			return nil, fmt.Errorf("disasm: byte at %s is marked as code but not as an instruction start", addr.String())
		}

		chunk := append([]byte(nil), bin.Data[index:end]...)
		lines = append(lines, Line{
			Kind:    LineData,
			Address: addr,
			Bytes:   chunk,
			Text:    formatDB(chunk),
			Comment: formatASCII(chunk),
		})
		index = end
	}

	return lines, nil
}

func (d *Disassembler) String(lines []Line) string {
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line.Address.String())
		b.WriteString("  ")
		b.WriteString(line.Text)
		if line.Comment != "" {
			b.WriteString("  ; |")
			b.WriteString(line.Comment)
			b.WriteString("|")
		}
	}
	return b.String()
}

func formatDB(data []byte) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("%02x", b))
	}
	return "db " + strings.Join(parts, ", ")
}

func formatASCII(data []byte) string {
	var b strings.Builder
	for _, v := range data {
		if v >= 0x20 && v <= 0x7e {
			b.WriteByte(v)
			continue
		}
		b.WriteByte('.')
	}
	return b.String()
}

func offsetAddress(origin x86.FarAddress, offset int) x86.FarAddress {
	return x86.NewFarAddress(origin.Segment, origin.Offset+uint16(offset))
}

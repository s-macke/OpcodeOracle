# MOS6502 Disassembler Specification

This document specifies the binary reading, parsing, and flow analysis functionality for the OpcodeOracle MOS6502 disassembler.

See [cli.md](cli.md) for command line interface specification.
See [export.md](export.md) for output format specification.

## Binary File Reading

### Loading Process

1. Open the binary file in read-only mode
2. Read entire file contents into a byte buffer
3. Validate file size is non-zero and fits within 64KB address space from origin
4. Create a memory map structure associating each byte with its virtual address

### Memory Mapping

The disassembler maintains a virtual 64KB address space mirroring the MOS6502's memory model:

```go
type MemoryMap struct {
    Data      []byte           // Raw binary data
    Origin    uint16           // Load address
    Size      int              // Size of loaded binary
    ByteType  []ByteCategory   // Classification for each byte
}

type ByteCategory int

const (
    ByteUnknown     ByteCategory = iota  // Not yet analyzed
    ByteCode                              // Part of an instruction
    ByteData                              // Data (not executed)
    ByteEntryPoint                        // Entry point marker
)
```

### Address Translation

```
file_offset = virtual_address - origin
virtual_address = file_offset + origin
```

## Flow-Following Disassembly Algorithm

The disassembler uses a recursive traversal algorithm that follows the program's control flow rather than linear disassembly.

### Algorithm Overview

```
1. Initialize work queue with entry point address
2. While work queue is not empty:
   a. Pop address from queue
   b. If address already visited or outside binary bounds, skip
   c. Decode instruction at address using opcodes table
   d. Mark instruction bytes as code
   e. Based on instruction type:
      - Sequential (most instructions): add next address to queue
      - Unconditional branch (JMP): add target to queue, do NOT add next
      - Conditional branch (Bxx): add both target AND next to queue
      - Subroutine call (JSR): add target to queue, add return address to queue
      - Return (RTS, RTI): do not add any address (end of path)
      - Break (BRK): do not add any address (end of path)
3. Mark all non-code bytes as data
```

### Instruction Classification

| Category           | Instructions                                                                                                                                                                                                          | Behavior                                   |
|--------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------|
| Sequential         | ADC, AND, ASL, BIT, CLC, CLD, CLI, CLV, CMP, CPX, CPY, DEC, DEX, DEY, EOR, INC, INX, INY, LDA, LDX, LDY, LSR, NOP, ORA, PHA, PHP, PLA, PLP, ROL, ROR, SBC, SEC, SED, SEI, STA, STX, STY, TAX, TAY, TSX, TXA, TXS, TYA | Continue to next instruction               |
| Unconditional Jump | JMP                                                                                                                                                                                                                   | Follow jump target only                    |
| Conditional Branch | BCC, BCS, BEQ, BMI, BNE, BPL, BVC, BVS                                                                                                                                                                                | Follow both branch target and fall-through |
| Subroutine Call    | JSR                                                                                                                                                                                                                   | Follow subroutine and return address       |
| Return             | RTS, RTI                                                                                                                                                                                                              | End of execution path                      |
| Break              | BRK                                                                                                                                                                                                                   | End of execution path                      |

### Branch Target Calculation

For relative addressing (branch instructions):
```
if operand > 0x7F:
    target = PC + 2 - (256 - operand)  // Negative offset
else:
    target = PC + 2 + operand           // Positive offset
```

Where PC is the address of the branch instruction.

### Label Generation

Labels are automatically generated for:
- Branch/jump targets: `L_XXXX` (where XXXX is the hex address)
- Subroutine entry points (JSR targets): `SUB_XXXX`

User-defined symbol names from the state file override auto-generated labels.

## Error Handling

| Error Condition | Behavior |
|-----------------|----------|
| File not found | Exit with error message |
| File read error | Exit with error message |
| Entry point outside binary | Exit with error message |
| Origin causes overflow | Exit with error message |
| Invalid address during flow analysis | Log warning, skip address |
| Illegal opcode encountered | Treat as 1-byte data, continue if reachable |

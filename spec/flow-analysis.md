# Flow Analysis Specification

This document specifies the flow analysis algorithm for OpcodeOracle.

See [disassembler.md](disassembler.md) for binary loading and memory mapping.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Flow analysis is a recursive traversal algorithm that follows the program's control flow rather than performing linear disassembly. Starting from entry points, it traces all reachable code paths, decoding instructions and recording their relationships in the state tables.

## Algorithm

```
1. Initialize work queue with entry point addresses
2. While work queue is not empty:
   a. Pop address from queue
   b. If address already visited or outside binary bounds, skip
   c. Decode instruction at address using opcodes table
   d. Mark instruction bytes as code
   e. Record cross-references for control flow instructions
   f. Generate symbols for targets
   g. Based on instruction type:
      - Sequential (most instructions): add next address to queue
      - Unconditional branch (JMP): add target to queue, do NOT add next
      - Conditional branch (Bxx): add both target AND next to queue
      - Subroutine call (JSR): add target to queue, add return address to queue
      - Return (RTS, RTI): do not add any address (end of path)
      - Break (BRK): do not add any address (end of path)
3. Mark all non-code bytes as data
```

## Instruction Classification

| Category           | Instructions                                                                                                                                                                                                          | Behavior                                   |
|--------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------|
| Sequential         | ADC, AND, ASL, BIT, CLC, CLD, CLI, CLV, CMP, CPX, CPY, DEC, DEX, DEY, EOR, INC, INX, INY, LDA, LDX, LDY, LSR, NOP, ORA, PHA, PHP, PLA, PLP, ROL, ROR, SBC, SEC, SED, SEI, STA, STX, STY, TAX, TAY, TSX, TXA, TXS, TYA | Continue to next instruction               |
| Unconditional Jump | JMP                                                                                                                                                                                                                   | Follow jump target only                    |
| Conditional Branch | BCC, BCS, BEQ, BMI, BNE, BPL, BVC, BVS                                                                                                                                                                                | Follow both branch target and fall-through |
| Subroutine Call    | JSR                                                                                                                                                                                                                   | Follow subroutine and return address       |
| Return             | RTS, RTI                                                                                                                                                                                                              | End of execution path                      |
| Break              | BRK                                                                                                                                                                                                                   | End of execution path                      |

## Branch Target Calculation

For relative addressing (branch instructions):
```
if operand > 0x7F:
    target = PC + 2 - (256 - operand)  // Negative offset
else:
    target = PC + 2 + operand           // Positive offset
```

Where PC is the address of the branch instruction.

## State Population

Flow analysis populates the following state tables as it traverses code:

### RegionTable

As instructions are decoded:
- Each instruction's bytes are marked as `code` in the RegionTable
- After analysis completes, all bytes not marked as code remain as `data`

See [regions-table.md](regions-table.md) for RegionTable interface.

### SymbolTable

Auto-generated symbols are created for control flow targets:

| Target Type      | Symbol Type  | Naming Pattern |
|------------------|--------------|----------------|
| JMP target       | `label`      | `L_XXXX`       |
| Branch target    | `label`      | `L_XXXX`       |
| JSR target       | `subroutine` | `SUB_XXXX`     |

All auto-generated symbols have `Source: auto`. User-defined symbols override auto-generated names in output.

See [symbol-table.md](symbol-table.md) for SymbolTable interface.

### XRefTable

Cross-references are recorded as instructions are decoded:

| Instruction | XRef Type |
|-------------|-----------|
| JMP         | `jump`    |
| JSR         | `call`    |
| Bxx         | `branch`  |

Each xref records the `From` address (instruction location) and `To` address (target).

See [xref-table.md](xref-table.md) for XRefTable interface.

### EntryPoints

- Initial entry points are provided by user or derived from binary format
- JSR targets are added as entry points for subroutine analysis
- Each entry point seeds the work queue for traversal

See [entrypoint-table.md](entrypoint-table.md) for entry point methods.

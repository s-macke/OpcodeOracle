# Flow Analysis Specification

This document specifies the flow analysis algorithm for OpcodeOracle.

See [disassembler.md](disassembler.md) for binary loading and memory mapping.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Flow analysis is a recursive traversal algorithm that follows the program's control flow rather than performing linear disassembly. Starting from entry points, it traces all reachable code paths, decoding instructions and recording their relationships in the state tables.

## Implementation

The flow analyzer is implemented in `src/internal/analysis/flow.go`.

### Types

```go
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

type Analyzer struct {
    state   *state.State
    visited map[uint16]bool
    queue   []uint16
}
```

### Public API

```go
// NewAnalyzer creates a new flow analyzer for the given state.
func NewAnalyzer(s *state.State) *Analyzer

// Analyze performs flow analysis starting from all entry points and known subroutines.
// Seeds the work queue from:
// - state.EntryPoints
// - state.ExtraCodeAddresses
// - Existing symbols with type: SymbolSubroutine, SymbolLabel, SymbolEntry
// - Existing code regions (region start addresses)
func (a *Analyzer) Analyze() error

// AnalyzeFromEntryPoints performs flow analysis using only state.EntryPoints and
// state.ExtraCodeAddresses as seeds. Only entryPoints regenerate entry symbols.
// Existing symbols/code regions do not seed traversal.
func (a *Analyzer) AnalyzeFromEntryPoints() error

// AnalyzeFrom performs flow analysis starting from a single address.
// Does not use entry points or existing symbols - only the given address.
// Can be called multiple times to incrementally discover code.
func (a *Analyzer) AnalyzeFrom(addr uint16) error
```

## Algorithm

```
1. Initialize work queue with seed addresses:
   - For Analyze(): entry points + extra code addresses + existing subroutine/label symbols
   - For AnalyzeFrom(addr): only the given address
2. While work queue is not empty:
   a. Pop address from queue
   b. If address already visited or outside binary bounds, skip
   c. If address is in forced-data range, skip
   d. Decode instruction at address using opcodes table
   e. If illegal opcode, skip (do not mark as code)
   f. If instruction bytes intersect forced-data range, skip
   g. Mark instruction bytes as code
   h. Record cross-references for control flow instructions
   i. Generate symbols for targets
   j. Based on instruction type:
      - Sequential (most instructions): add next address to queue
      - Unconditional jump (JMP absolute): add target to queue, do NOT add next
      - Unconditional jump (JMP indirect): do NOT add any address (cannot follow statically)
      - Conditional branch (Bxx): add both target AND next to queue
      - Subroutine call (JSR): add target to queue, add return address to queue
      - Return (RTS, RTI): do not add any address (end of path)
      - Break (BRK): do not add any address (end of path)
3. All bytes not marked as code remain as data
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
- Any address in a non-auto `data` region (`source=user|assistant`) remains `data` (hard lock)

See [regions-table.md](regions-table.md) for RegionTable interface.

### SymbolTable

Auto-generated symbols are created for control flow targets:

| Target Type      | Symbol Type  | Naming Pattern |
|------------------|--------------|----------------|
| Entry point      | `entry`      | `ENTRY_XXXX`   |
| JMP target       | `label`      | `L_XXXX`       |
| Branch target    | `label`      | `L_XXXX`       |
| JSR target       | `subroutine` | `SUB_XXXX`     |

All auto-generated symbols have `Source: auto`. User-defined symbols override auto-generated names in output.

`state.ExtraCodeAddresses` seed traversal but do not create `entry` symbols at those addresses.

Existing symbols in the table (from user input, imports, or previous analysis) with types `subroutine`, `label`, or `entry` are used as additional seed points for `Analyze()`.

For strict rebuild operations, `AnalyzeFromEntryPoints()` ignores existing symbols and existing code regions as seeds and uses only `state.EntryPoints` plus `state.ExtraCodeAddresses`.

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

- Initial entry points are provided by user or derived from binary format (stored in `state.EntryPoints`)
- Each entry point seeds the work queue for `Analyze()`
- Entry point symbols (`ENTRY_XXXX`) are auto-generated for each entry point
- `state.ExtraCodeAddresses` also seed the work queue, but do not generate entry symbols

Note: JSR targets are NOT added to `state.EntryPoints` - they are followed during analysis and receive `SUB_XXXX` symbols instead.

## Handling Special Cases

### Illegal Opcodes

When an illegal opcode is encountered:
- The instruction is NOT marked as code
- The address is NOT marked as visited
- Analysis continues with remaining queue items
- No error is returned (graceful degradation)

### Indirect JMP

`JMP ($XXXX)` cannot be followed statically because the target address is read from memory at runtime. The instruction is marked as code, but no xref or target symbol is created, and analysis of that path ends.

### Out-of-Bounds Targets

If a jump/branch/call target is outside the binary's address range:
- The xref and symbol are still created (if within bounds check)
- The target is not enqueued for analysis
- Analysis continues with remaining paths

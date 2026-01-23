# Cross-Reference Specification

This document specifies the cross-reference system for OpcodeOracle.

See [state.md](state.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Cross-references (xrefs) track relationships between memory addresses in the binary. They record where code jumps, calls, branches, or accesses data, enabling navigation and understanding of program flow.

## Type Definition

### XRefType

```go
type XRefType string

const (
    XRefCall   XRefType = "call"   // JSR target
    XRefJump   XRefType = "jump"   // JMP target
    XRefBranch XRefType = "branch" // Conditional branch target
    XRefRead   XRefType = "read"   // Memory read
    XRefWrite  XRefType = "write"  // Memory write
)
```

| Value    | Description                                      |
|----------|--------------------------------------------------|
| `call`   | Subroutine call (JSR instruction target)         |
| `jump`   | Unconditional jump (JMP instruction target)      |
| `branch` | Conditional branch (BEQ, BNE, etc. target)       |
| `read`   | Memory read (LDA, LDX, LDY, etc. operand)        |
| `write`  | Memory write (STA, STX, STY, etc. operand)       |

## Struct Definition

### XRef

```go
type XRef struct {
    From uint16
    To   uint16
    Type XRefType
}
```

| Field  | Type     | Description                              |
|--------|----------|------------------------------------------|
| `From` | uint16   | Source address (where the reference is)  |
| `To`   | uint16   | Target address (what is being referenced)|
| `Type` | XRefType | Type of reference                        |

## XRefTable Interface

```go
type XRefTable interface {
    To(addr uint16) []XRef
    From(addr uint16) []XRef
    Add(from, to uint16, refType XRefType)
    Remove(from, to uint16)
    All() []XRef
}
```

| Method                                   | Description                              |
|------------------------------------------|------------------------------------------|
| `To(addr uint16) []XRef`                 | Get all references to address            |
| `From(addr uint16) []XRef`               | Get all references from address          |
| `Add(from, to uint16, refType XRefType)` | Add cross-reference                      |
| `Remove(from, to uint16)`                | Remove cross-reference                   |
| `All() []XRef`                           | Get all cross-references                 |

## Cross-Reference Rules

- Multiple references can exist between the same from/to pair (different types)
- References are typically auto-generated during disassembly
- References to addresses outside the binary (e.g., ROM routines) are valid
- Removing code does not automatically remove associated xrefs

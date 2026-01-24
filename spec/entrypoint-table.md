# Entry Point Specification

This document specifies the entry point system for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

Entry points are known code execution starting addresses in the binary. They serve as seeds for disassembly, telling the analyzer where code begins. Common entry points include:

- Program start address (reset vector)
- Interrupt handlers (IRQ, NMI, BRK)
- Jump targets discovered during analysis
- User-specified entry points for known routines

## EntryPoint Methods

Entry point management is provided directly on the State interface:

```go
// Entry point methods on State interface
EntryPoints() []uint16
AddEntryPoint(addr uint16)
RemoveEntryPoint(addr uint16)
HasEntryPoint(addr uint16) bool
```

| Method                           | Description                            |
|----------------------------------|----------------------------------------|
| `EntryPoints() []uint16`         | Get list of entry point addresses      |
| `AddEntryPoint(addr uint16)`     | Add entry point (no-op if exists)      |
| `RemoveEntryPoint(addr uint16)`  | Remove entry point (no-op if not exists) |
| `HasEntryPoint(addr uint16) bool`| Check if address is an entry point     |

## Usage

Entry points drive the disassembly process. When analyzing a binary:

1. Add known entry points (reset vector, documented routines)
2. The analyzer follows code flow from each entry point
3. New entry points may be discovered (jump targets, call destinations)
4. Each entry point is processed until all reachable code is traced

## Behavior

- Entry points are stored as a sorted list of unique addresses
- `AddEntryPoint()` is idempotent: adding an existing entry point has no effect
- `RemoveEntryPoint()` is safe: removing a non-existent entry point has no effect
- Entry points outside the binary address range are allowed but ignored during analysis

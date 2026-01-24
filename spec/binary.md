# Binary Specification

This document specifies the binary data interface for OpcodeOracle.

See [state-file.md](state-file.md) for the JSON file format.
See [state-interface.md](state-interface.md) for the full state interface.

## Overview

The Binary struct holds the raw binary data, entry points, and provides methods to read bytes and words at virtual addresses. It handles address translation from the virtual address space (based on the origin/load address) to offsets within the binary data.

## Type Definitions

```go
type EntryPoint uint16

type Binary struct {
    Data        []byte       // Raw binary data
    Origin      uint16       // Load address
    EntryPoints []EntryPoint // Code execution starting addresses
}

func (b *Binary) ReadByte(addr uint16) (byte, error)
func (b *Binary) ReadWord(addr uint16) (uint16, error)
func (b *Binary) IsEntryPoint(addr uint16) bool
```

### Fields

| Field         | Type           | Description                          |
|---------------|----------------|--------------------------------------|
| `Data`        | `[]byte`       | Raw binary data as byte array        |
| `Origin`      | `uint16`       | Load address of binary in memory     |
| `EntryPoints` | `[]EntryPoint` | Code execution starting addresses    |

### Methods

| Method                                  | Description                                |
|-----------------------------------------|--------------------------------------------|
| `ReadByte(addr uint16) (byte, error)`   | Read byte at virtual address               |
| `ReadWord(addr uint16) (uint16, error)` | Read little-endian word at virtual address |
| `IsEntryPoint(addr uint16) bool`        | Check if address is an entry point         |

## Address Translation

Virtual addresses are translated to binary offsets:

```
offset = addr - Origin
```

For example, with `Origin = 0x0801`:
- Address `0x0801` reads offset `0`
- Address `0x0810` reads offset `15`

## Entry Points

Entry points are known code execution starting addresses in the binary. They serve as seeds for disassembly, telling the analyzer where code begins. Common entry points include:

- Program start address (reset vector)
- Interrupt handlers (IRQ, NMI, BRK)
- Jump targets discovered during analysis
- User-specified entry points for known routines

### Entry Point Behavior

- Entry points are stored as a list of unique addresses
- Entry points drive the disassembly process: the analyzer follows code flow from each entry point

## Error Conditions

| Condition                | Error                  |
|--------------------------|------------------------|
| Address below Origin     | `ErrAddressOutOfRange` |
| Address beyond binary end| `ErrAddressOutOfRange` |

## Usage Example

```go
state, _ := LoadState("game.orc")

// Read a byte
b, err := state.Binary.ReadByte(0x0810)
if err != nil {
    return err
}

// Read a little-endian word
w, err := state.Binary.ReadWord(0x0820)
if err != nil {
    return err
}
fmt.Printf("Word at $0820: $%04X\n", w)

// Iterate entry points
for _, ep := range state.Binary.EntryPoints {
    fmt.Printf("Entry point: $%04X\n", ep)
}
```

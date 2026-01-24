# OpcodeOracle - Project Specification

An agentic system for reverse engineering legacy computer code, focusing on MOS6502 assembler for the Commodore 64.

**Language:** Go

## Project Goals

1. Provide accurate disassembly of MOS6502 binary code
2. Use flow analysis to distinguish code from data
3. Generate readable, reassemblable output
4. Support agentic/AI-assisted reverse engineering workflows

## Feature Roadmap

### Phase 1: Foundation

Core data structures and state management - must be implemented first.

| Feature                    | Status  | Specification                              |
|----------------------------|---------|--------------------------------------------|
| Core type definitions      | Planned | [architecture.md](architecture.md)         |
| State file format          | Planned | [state-file.md](state-file.md)                       |
| Symbol table               | Planned | [symbol-table.md](symbol-table.md)         |
| Annotation table           | Planned | [annotation-table.md](annotation-table.md) |
| Region table               | Planned | [regions-table.md](regions-table.md)       |
| Cross-reference table      | Planned | [xref-table.md](xref-table.md)             |

### Phase 2: State Interface Implementation

| Feature         | Status  | Specification                            |
|-----------------|---------|------------------------------------------|
| State interface | Planned | [state-interface.md](state-interface.md) |

### Phase 3: Binary Reading and Flow Analysis

| Feature                    | Status  | Specification                          |
|----------------------------|---------|----------------------------------------|
| Binary file reading        | Planned | [disassembler.md](disassembler.md)     |
| Flow-following disassembly | Planned | [flow-analysis.md](flow-analysis.md)   |
| Populate state from binary | Planned | [flow-analysis.md](flow-analysis.md)   |
| MOS6502 opcode definitions | Planned | [opcodes.go](opcodes.go)               |

### Phase 4: Output Generation

Output assembly listing from state file. All output files include auto-generated headers.

| Feature                    | Status  | Specification              |
|----------------------------|---------|----------------------------|
| Auto-generated headers     | Planned | [export.md](export.md)     |
| Main disassembly file      | Planned | [export.md](export.md)     |
| Segment files              | Planned | [export.md](export.md)     |
| Data section hex dump      | Planned | [export.md](export.md)     |
| Symbol resolution          | Planned | [export.md](export.md)     |

### Phase 5: Enhanced Analysis

| Feature                | Status  | Specification        |
|------------------------|---------|----------------------|
| C64 ROM symbols        | Planned | [state-file.md](state-file.md) |
| C64 I/O register names | Planned | TBD                  |
| Cross-reference tables | Planned | [state-file.md](state-file.md) |
| Subroutine detection   | Planned | TBD                  |

### Phase 6: Agentic Features

| Feature                     | Status  | Specification |
|-----------------------------|---------|---------------|
| AI-assisted code annotation | Planned | TBD           |
| Pattern recognition         | Planned | TBD           |
| Automatic variable naming   | Planned | TBD           |
| Code structure analysis     | Planned | TBD           |

## Specification Documents

| Document                                   | Description                                |
|--------------------------------------------|--------------------------------------------|
| [overview.md](overview.md)                 | This file - project overview and roadmap   |
| [cli.md](cli.md)                           | Command line interface                     |
| [opcodes.go](opcodes.go)                   | MOS6502 instruction definitions            |
| [architecture.md](architecture.md)         | Core Go type and struct definitions        |
| [state-file.md](state-file.md)                       | JSON state file format for save/load       |
| [state-interface.md](state-interface.md)   | Go interface for state manipulation        |
| [symbol-table.md](symbol-table.md)         | Symbol types, struct, and interface        |
| [annotation-table.md](annotation-table.md) | Annotation struct and interface            |
| [regions-table.md](regions-table.md)       | Region types, struct, and interface        |
| [xref-table.md](xref-table.md)             | Cross-reference types, struct, and interface |
| [entrypoint-table.md](entrypoint-table.md) | Entry point methods                        |
| [disassembler.md](disassembler.md)         | Binary reading and memory mapping          |
| [flow-analysis.md](flow-analysis.md)       | Flow analysis algorithm and state population |
| [export.md](export.md)                     | Assembly output format specification       |

## References

- [MOS6502 Programming Manual](http://archive.6502.org/books/mcs6500_family_programming_manual.pdf)
- [C64 Memory Map](https://sta.c64.org/cbm64mem.html)
- [C64 ROM Disassembly](https://www.pagetable.com/c64ref/c64disasm/)

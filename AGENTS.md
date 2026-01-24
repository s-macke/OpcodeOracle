# OpcodeOracle

An agentic system for reverse engineering legacy computer code, focusing on MOS6502 assembler for the Commodore 64.

## Goals

1. Provide accurate disassembly of MOS6502 binary code
2. Use flow analysis to distinguish code from data
3. Generate readable, reassemblable output
4. Support agentic/AI-assisted reverse engineering workflows

## Specification Index

| File                                                 | Description                                                                  |
|------------------------------------------------------|------------------------------------------------------------------------------|
| [overview.md](spec/overview.md)                      | Project goals, feature roadmap, and links to all specification documents.    |
| [architecture.md](spec/architecture.md)              | Core Go struct definitions and project directory structure.                  |
| [cli.md](spec/cli.md)                                | Command line interface for `new`, `info`, and `export` commands.             |
| [state-file.md](spec/state-file.md)                  | JSON format for persisting reverse engineering sessions (`.orc` files).      |
| [state-interface.md](spec/state-interface.md)        | Go interface for loading, saving, and querying state data.                   |
| [binary.md](spec/binary.md)                          | Binary struct and read methods for accessing raw data.                       |
| [symbol-table.md](spec/symbol-table.md)              | Symbol types, struct, and interface for naming memory addresses.             |
| [annotation-table.md](spec/annotation-table.md)      | Annotation types (inline/headline), struct, and interface for comments.      |
| [regions-table.md](spec/regions-table.md)            | Region types and interface for classifying memory areas as code/data.        |
| [xref-table.md](spec/xref-table.md)                  | Cross-reference types and interface for tracking jumps, calls, and branches. |
| [entrypoint-table.md](spec/entrypoint-table.md)      | Entry point methods for managing code execution starting addresses.          |
| [disassembler.md](spec/disassembler.md)              | Disassembler interface, addressing modes, and output formatting.             |
| [flow-analysis.md](spec/flow-analysis.md)            | Flow-following disassembly algorithm and state population.                   |
| [export.md](spec/export.md)                          | Assembly output file structure and segment generation rules.                 |

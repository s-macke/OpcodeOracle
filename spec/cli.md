# Command Line Interface Specification

This document specifies the command line interface for the OpcodeOracle disassembler.

See [export.md](export.md) for output format specification.

## Usage

```
opcodeoracle <command> [options] [arguments]
```

## Commands

| Command  | Description                                    |
|----------|------------------------------------------------|
| `new`    | Create new project from binary                 |
| `info`   | Display state file information                 |
| `export` | Export state to assembly files                 |

---

## `new` - Create New Project

Creates a new state file from a binary and runs flow-following disassembly.

### Usage

```
opcodeoracle new <binary-file> -e <entry> -o <origin>
```

### Parameters

| Parameter   | Flag             | Required | Description                              |
|-------------|------------------|----------|------------------------------------------|
| Binary file | positional       | Yes      | Path to the binary file                  |
| Entry point | `-e`, `--entry`  | Yes      | Entry point address in hex               |
| Origin      | `-o`, `--origin` | Yes      | Load address/origin in hex               |

### Behavior

1. Reads the binary file
2. Creates state file `<binary-name>.orc`
3. Runs flow-following disassembly from entry point
4. Populates symbols, regions, and code/data classification (see [state.md](state.md))
5. Saves state file

### Examples

```bash
# Create new project from C64 PRG file
opcodeoracle new game.prg -e 0x0810 -o 0x0801
# Creates: game.orc

# Using $ prefix for addresses
opcodeoracle new demo.bin --entry $1000 --origin $0800
# Creates: demo.orc
```

---

## `info` - Show Project Information

Displays information about a state file.

### Usage

```
opcodeoracle info <state-file>
```

### Parameters

| Parameter  | Flag       | Required | Description              |
|------------|------------|----------|--------------------------|
| State file | positional | Yes      | Path to .orc state file  |

### Output

```
Project:     game.orc
Source:      game.prg
Origin:      $0801
Entry:       $0810
Binary size: 4096 bytes
Symbols:     23
  Functions: 8
  Labels:    12
  Variables: 3
Regions:
  Code:      2048 bytes
  Data:      2048 bytes
```

### Examples

```bash
opcodeoracle info game.orc
```

---

## `export` - Export to Assembly Files

Generates assembly output files from state. See [export.md](export.md) for detailed output format.

### Usage

```
opcodeoracle export <state-file>
```

### Parameters

| Parameter  | Flag       | Required | Description             |
|------------|------------|----------|-------------------------|
| State file | positional | Yes      | Path to .orc state file |

Output filename is derived from state file: `game.orc` → `game.asm`

### Behavior

1. Loads state file
2. Creates main assembly file (`<name>.asm`)
3. Creates `segments/` directory with segment files:
   - `0x{addr}_sub.asm` - Subroutines
   - `0x{addr}_code.asm` - Code blocks
   - `0x{addr}_dat.asm` - Data sections

### Examples

```bash
opcodeoracle export game.orc
# Creates: game.asm + segments/
```

---

## Global Options

| Option          | Description                |
|-----------------|----------------------------|
| `-h`, `--help`  | Show help message          |
| `-v`, `--version` | Show version information |

## Exit Codes

| Code | Description              |
|------|--------------------------|
| 0    | Success                  |
| 1    | Invalid command/arguments|
| 2    | File not found           |
| 3    | File read/write error    |
| 4    | Invalid state file       |
| 5    | Disassembly error        |

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
opcodeoracle new <type> <binary-file> [options]
```

### File Types

| Type  | Description                     | Required Options                |
|-------|---------------------------------|---------------------------------|
| `bin` | Raw binary data                 | `--skip`, `--entry`, `--origin` |
| `prg` | C64 PRG file with load address  | `--entry`                       |
| `sid` | SID music file                  | (none)                          |

### Parameters

| Parameter   | Flag             | Required      | Description                              |
|-------------|------------------|---------------|------------------------------------------|
| Type        | positional       | Yes           | File type: `bin`, `prg`, or `sid`        |
| Binary file | positional       | Yes           | Path to the binary file                  |
| Skip bytes  | `-s`, `--skip`   | bin only      | Bytes to skip at start of file           |
| Entry point | `-e`, `--entry`  | bin, prg      | Entry point address                      |
| Origin      | `-o`, `--origin` | bin only      | Load address/origin                      |

### Number Format

Numeric parameters accept decimal or hexadecimal values:

| Format   | Example    | Value  |
|----------|------------|--------|
| Decimal  | `2048`     | 2048   |
| Hex (`$`)| `$0800`    | 2048   |
| Hex (`0x`)| `0x0800`  | 2048   |

### File Type Details

#### Raw Binary (`bin`)

Raw binary files require all addressing information:
- `--skip`: Number of bytes to skip at the beginning (header, etc.)
- `--origin`: Address where the binary is loaded in memory
- `--entry`: Address where code execution begins

#### C64 PRG (`prg`)

PRG files contain a 2-byte load address header:
- Origin is read from the first two bytes (little-endian)
- `--entry`: Required, specifies code start address

#### SID File (`sid`)

SID files contain complete header information:
- Load address, init address, and play address are in the header
- No additional parameters required
- Multiple entry points are extracted (init and play routines)

### Behavior

1. Reads the binary file
2. Extracts or uses provided origin and entry points
3. Creates state file `<binary-name>.orc`
4. Runs flow-following disassembly from entry point(s)
5. Populates symbols, regions, and code/data classification (see [state-file.md](state-file.md))
6. Saves state file

### Examples

```bash
# Raw binary - all parameters required (hex with $)
opcodeoracle new bin code.bin --skip 2 --origin $0800 --entry $0800
# Creates: code.orc

# C64 PRG file - hex with 0x prefix
opcodeoracle new prg game.prg --entry 0x0810
# Creates: game.orc

# Decimal values also work
opcodeoracle new prg game.prg --entry 2064
# Creates: game.orc

# SID file - everything from header
opcodeoracle new sid music.sid
# Creates: music.orc
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
  Subroutines: 8
  Labels:      12
  Bytes:       2
  Words:       1
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

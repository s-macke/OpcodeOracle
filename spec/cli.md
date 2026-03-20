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
| `edit`   | Edit state (annotations, symbols, reinterpret) |
| `mcp`    | Start MCP server (stdio or streamable HTTP)   |

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
| `bin` | Raw binary data                 | `--entry`                       |
| `prg` | C64 PRG file with load address  | `--entry`                       |
| `sid` | SID music file                  | (none)                          |

### Parameters

| Parameter   | Flag             | Required      | Description                              |
|-------------|------------------|---------------|------------------------------------------|
| Type        | positional       | Yes           | File type: `bin`, `prg`, or `sid`        |
| Binary file | positional       | Yes           | Path to the binary file                  |
| Skip bytes  | `-s`, `--skip`   | No (default `0`) | Bytes to skip at start of file        |
| Entry point | `-e`, `--entry`  | bin, prg      | Entry point address(es), comma-separated |
| Origin      | `-o`, `--origin` | No (default `0`) | Load address/origin                   |

### Number Format

Numeric parameters accept decimal or hexadecimal values:

| Format   | Example    | Value  |
|----------|------------|--------|
| Decimal  | `2048`     | 2048   |
| Hex (`$`)| `$0800`    | 2048   |
| Hex (`0x`)| `0x0800`  | 2048   |

### File Type Details

#### Raw Binary (`bin`)

Raw binary files require entry point information:
- `--skip`: Number of bytes to skip at the beginning (header, etc.); defaults to `0`
- `--origin`: Address where the binary is loaded in memory; defaults to `0`
- `--entry`: Address(es) where code execution begins (required, comma-separated)
- If the remaining file data is larger than the available 16-bit address space from `origin`, it is truncated automatically

#### C64 PRG (`prg`)

PRG files contain a 2-byte load address header:
- Origin is read from the first two bytes (little-endian)
- `--entry`: Required, specifies code start address(es), comma-separated

#### SID File (`sid`)

SID files contain complete header information:
- Load address, init address, and play address are in the header
- No additional parameters required
- Multiple entry points are extracted (init and play routines)

### Output

Creates a state file named `<binary-name>.opcodeoracle.json` in the current working directory.

### Behavior

1. Reads the binary file
2. Extracts or uses provided origin and entry points
3. Creates state file `<binary-name>.opcodeoracle.json` in the current working directory
4. Runs flow-following disassembly from entry point(s)
5. Populates symbols, regions, and code/data classification (see [state-file.md](state-file.md))
6. Saves state file

### Examples

```bash
# Raw binary - optional skip/origin (hex with $)
opcodeoracle new bin code.bin --skip 2 --origin $0800 --entry $0800
# Creates: code.opcodeoracle.json

# Raw binary - skip/origin use defaults (0)
opcodeoracle new bin code.bin --entry $0800
# Creates: code.opcodeoracle.json

# Raw binary - multiple entry points
opcodeoracle new bin code.bin --entry "$0800,$0810,2064"
# Creates: code.opcodeoracle.json

# C64 PRG file - hex with 0x prefix
opcodeoracle new prg game.prg --entry 0x0810
# Creates: game.opcodeoracle.json

# Decimal values also work
opcodeoracle new prg game.prg --entry 2064
# Creates: game.opcodeoracle.json

# PRG file - multiple entry points
opcodeoracle new prg game.prg --entry "0x0810,2064"
# Creates: game.opcodeoracle.json

# SID file - everything from header
opcodeoracle new sid music.sid
# Creates: music.opcodeoracle.json
```

---

## `info` - Show Project Information

Displays information about a state file.

### Usage

```
opcodeoracle info <state-file>
```

### Parameters

| Parameter  | Flag       | Required | Description                          |
|------------|------------|----------|--------------------------------------|
| State file | positional | Yes      | Path to .opcodeoracle.json state file |

### Output

```
Project:     game.opcodeoracle.json
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
opcodeoracle info game.opcodeoracle.json
```

---

## `export` - Export to Assembly Files

Generates assembly output files from state. See [export.md](export.md) for detailed output format.

### Usage

```
opcodeoracle export <state-file>
```

### Parameters

| Parameter  | Flag       | Required | Description                          |
|------------|------------|----------|--------------------------------------|
| State file | positional | Yes      | Path to .opcodeoracle.json state file |

Output filename is derived from state file: `game.opcodeoracle.json` → `game.asm`

### Behavior

1. Loads state file
2. Creates main assembly file (`<name>.asm`)
3. Creates `segments/` directory with segment files:
   - `0x{addr}_sub.asm` - Subroutines
   - `0x{addr}_code.asm` - Code blocks
   - `0x{addr}_dat.asm` - Data sections

### Examples

```bash
opcodeoracle export game.opcodeoracle.json
# Creates: game.asm + segments/
```

---

## `mcp` - Start MCP Server

Starts an MCP server exposing OpcodeOracle reverse-engineering tools.

### Usage

```
opcodeoracle mcp [options] <state-file>
```

### Parameters

| Parameter  | Flag       | Required | Description                          |
|------------|------------|----------|--------------------------------------|
| State file | positional | Yes      | Path to .opcodeoracle.json state file |

### Options

| Option        | Required | Default | Description |
|---------------|----------|---------|-------------|
| `--transport` | No       | `stdio` | MCP transport: `stdio` or `http` |
| `--listen`    | http only | (none) | Listen address for HTTP mode (for example `127.0.0.1:8080`) |
| `--path`      | No       | `/mcp`  | HTTP endpoint path in HTTP mode |
| `--dry-run`   | No       | `false` | Show changes without saving state file |
| `--verbose`   | No       | `false` | Enable detailed MCP server logs on stderr |

### Behavior

1. Loads the state file and initializes analysis context
2. Registers MCP tools:
   - `view_disassembly`
   - `search_disassembly`
   - `add_annotation` (supports optional `extend`)
   - `remove_annotation`
   - `add_headline` (supports optional `extend`)
   - `remove_headline`
   - `add_symbol`
   - `remove_symbol`
   - `query_symbols`
   - `query_xrefs`
   - `reinterpret_as_code`
   - `reinterpret_as_data`
   - `list_subroutines_and_data_segments`
   - `get_subroutine_context`
3. Starts transport:
   - `stdio`: newline-delimited JSON-RPC over stdin/stdout
   - `http`: streamable HTTP endpoint at `http://<listen><path>`
4. Emits MCP server logs to stderr:
   - Default: startup, shutdown, and tool call summaries
   - `--verbose`: includes argument/result previews and autosave details

### Examples

```bash
# Stdio transport (default)
opcodeoracle mcp game.opcodeoracle.json

# Explicit stdio transport
opcodeoracle mcp --transport stdio game.opcodeoracle.json

# Streamable HTTP transport
opcodeoracle mcp --transport http --listen 127.0.0.1:8080 --path /mcp game.opcodeoracle.json
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

---

## `edit reinterpret` - Force Code/Data Reinterpretation

Forces reinterpretation and reruns analysis from scratch.

### Usage

```bash
opcodeoracle edit reinterpret <state-file> --code-address <addr>
opcodeoracle edit reinterpret <state-file> --data-start <addr> --data-end <addr>
```

### Options

| Option           | Required | Description |
|------------------|----------|-------------|
| `--code-address` | code mode only | Single address to force as code seed |
| `--data-start`   | data mode only | Start of hard-locked data range |
| `--data-end`     | data mode only | End of hard-locked data range |

`--code-address` is mutually exclusive with `--data-start/--data-end`.

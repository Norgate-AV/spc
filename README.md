# spc
CLI wrapper for the Crestron SIMPL+ Compiler

## Usage

```bash
spc [command] [options] <file>
```

### Commands

- `build` (default): Compile a SIMPL+ program

### Options

- `-t, --target string`: Target series to compile for (e.g., 3, 34, 234)
- `-v, --verbose`: Verbose output
- `--version`: Show version information

### Examples

```bash
# Compile for 3-Series
spc build --target 3 example.usp

# Compile for 3-Series and 4-Series
spc build --target 34 example.usp

# Default command (build) can be omitted
spc --target 3 example.usp
```

## Configuration

Supports hierarchical configuration with YAML, JSON, or TOML formats.

Precedence (highest to lowest):
1. CLI options
2. Local config (`.spc.[yml|json|toml]` in project directory or upwards)
3. Global config (`%APPDATA%\spc\config.[yml|json|toml]`)
4. Defaults

### Config File Example (YAML)

```yaml
compiler_path: "C:\\Program Files (x86)\\Crestron\\Simpl\\SPlusCC.exe"
target: "4"
```

# spc

A better way to compile SIMPL+ files.

## Usage

```bash
spc [command] [options] <file...>
```

### Commands

- `build` (default): Compile one or more SIMPL+ programs

### Options

- `-t, --target string`: Target series to compile for (e.g., 3, 34, 234)
- `-v, --verbose`: Verbose output
- `-o, --out string`: Output file for compilation logs
- `-u, --usersplusfolder stringSlice`: User SIMPL+ folders (can specify multiple)
- `--version`: Show version information

### Examples

```bash
# Compile single file for 3-Series
spc build --target 3 example.usp

# Compile multiple files for 3 and 4-Series
spc --target 34 example.usp another.usl

# Default command (build) can be omitted
spc --target 3 file1.usp file2.usp
```

## Configuration

Supports hierarchical configuration with YAML, JSON, or TOML formats.

Precedence (highest to lowest):

1. CLI options
2. Local config (`.spc.[yml|json|toml]` in project directory or upwards)
3. Global config (`%APPDATA%\spc\config.[yml|json|toml]`)
4. Defaults

### Config File Example

```yaml
compiler_path: "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe"
target: "4"
out: "build.log"
usersplusfolder:
  - "C:/MyCustomSimplPlus"
silent: false
verbose: false
```

```json
{
    "compiler_path": "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe",
    "target": "4",
    "out": "build.log",
    "usersplusfolder": ["C:/MyCustomSimplPlus"],
    "silent": false,
    "verbose": false
}
```

```toml
compiler_path = "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe"
target = "4"
out = "build.log"
usersplusfolder = ["C:/MyCustomSimplPlus"]
silent = false
verbose = false
```

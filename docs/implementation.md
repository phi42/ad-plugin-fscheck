# Implementation Overview

This document explains the code structure, design, and execution flow of the fscheck plugin. It covers the purpose of each file and the concepts behind the major components.

## Architecture

The plugin is a standalone Go binary. The `ade` tool invokes it by serializing a `Spec` protobuf message (which represents one parsed `.rule` file) and writing those bytes to the plugin's stdin. The plugin reads stdin, processes the file rules, and communicates results through stdout (JSON info response) or stderr (progress, warnings, and errors).

The plugin supports a single mode:

| Mode   | Description                                                                                |
| ------ | ------------------------------------------------------------------------------------------ |
| verify | Evaluates `file` rules against the filesystem rooted at the configured directory.          |

There is no compile mode because file checks have no compiled artifact: they are evaluated directly at verify time without generating intermediate test code.

## Package layout

```
ad-plugin-fscheck/
├── main.go              entry point
├── cmd/
│   └── root.go          plugin protocol, mode dispatch, result reporting
└── fscheck/
    ├── types.go         CheckResult type
    ├── runner.go        rule iteration and dispatch
    ├── checks.go        the four supported check kinds
    └── glob.go          glob pattern expansion
```

## Files

### `main.go`

The binary entry point. It delegates immediately to the `cmd` package and contains no logic of its own.

### `cmd`

#### `root.go`

Implements the plugin protocol and top-level flow:

- When invoked with `--info`, it prints a JSON descriptor listing the supported modes and config prefix, then exits. The `ade` host calls this before each invocation to verify that the plugin supports the requested mode.
- When invoked interactively (stdin is a terminal), it prints a help message and exits.
- Otherwise, it reads the serialized `Spec` protobuf from stdin, resolves the root directory from `plugin_config["root-dir"]` (falling back to the current working directory), and calls into the `fscheck` package.

After the runner returns, it prints one line per rule result to stderr (`passed`, `warn`, or `error`) and exits with a non-zero code if any rule with error severity failed.

### `fscheck`

#### `types.go`

Defines the `CheckResult` type that aggregates the outcome of all checks within one rule. It carries the rule name plus separate slices for failures (rules with error severity that failed) and warnings (rules with warning severity that failed).

#### `runner.go`

The orchestrator. It iterates over the rules in the `Spec`, skips non-file rules, dispatches each `Check` entry to the appropriate kind-specific function in `checks.go`, and accumulates the resulting messages into either the failures or warnings slice depending on the rule's severity.

The runner also handles selector resolution: if a check's path string matches the name of a selector defined in the spec, the selector's pattern value is used in place of the literal path. This lets file rules reuse `path` selectors declared at the top of the rule file.

The DSL may embed a `glob:` or `regex:` scheme prefix on path and pattern values to disambiguate matching strategies. Both prefixes are stripped before evaluation since fscheck always treats paths as globs and content patterns as regular expressions.

#### `checks.go`

Implements the four supported check kinds:

- `must exist`: passes if the glob expands to at least one existing path.
- `must not exist`: fails for each existing path the glob expands to.
- `must contain`: compiles the pattern as a Go regex and fails for each matching file whose contents do not match. An empty match set is itself a failure, since a contain check on a nonexistent file is meaningless.
- `must not contain`: same as above, inverted; an empty match set is not a failure.

Content checks use Go's standard `regexp` package on the entire file contents (no line splitting). Files are filtered to regular files only before reading.

#### `glob.go`

Implements the glob expansion. Patterns are split on `/` into segments and walked recursively. The wildcard segments supported are:

- `*` and `?`: standard `filepath.Match` semantics within a single segment.
- `**`: matches zero or more directory levels. The expander tries the remaining segments at the current level, then recurses into each subdirectory while keeping the `**` segment in the parts list.
- `.`: skipped, treated as a no-op.

Missing intermediate directories are not an error: they simply produce an empty match set. This keeps `must not exist` rules from failing with a filesystem error when the parent directory is also absent.

## Execution flow

```
    ┌────────────────────┐
    │  ade verify        │
    │    -i adr.rule     │
    │    -p fscheck      │
    └─────────┬──────────┘
              │
  (ade serializes .rule file
   writes it to plugin's stdin)
              │
              │
              ▼
┌─────────────────────────────┐
│         [cmd/root.go]       │
│                             │
│  Read Spec from stdin       │
│  Resolve root dir           │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────┐
│     [fscheck/runner.go]     │
│                             │
│  For each file rule:        │
│    For each check:          │
│      Resolve selector       │
│      Dispatch by kind       │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────┐
│  [fscheck/checks.go +       │
│       fscheck/glob.go]      │
│                             │
│  Expand glob, stat paths,   │
│  read files, match regex    │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────┐
│         [cmd/root.go]       │
│                             │
│  Print pass/warn/error      │
│  per rule to stderr         │
└─────────────────────────────┘
```

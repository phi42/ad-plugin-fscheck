# Filesystem Check Plugin for Architectural Decision Enforcement

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

An [ad-guidance-tool](https://github.com/adr/ad-guidance-tool) enforcement plugin that verifies `file` rules from the ADE DSL against the filesystem. Rules express glob-based existence and content assertions that are evaluated at runtime against a target directory.

## Installation

Install from a GitHub release:

```sh
ade plugin install fscheck --repo github.com/phi42/ad-plugin-fscheck
```

Or build from source and register locally:

```sh
go build -o fscheck
ade plugin install fscheck --path ./fscheck
```

## Usage

### Verify

```sh
ade verify -i path/to/adr.rule -p fscheck
```

The plugin evaluates each `file` rule in the rule file against the configured root directory and prints one `passed`, `warn`, or `error` line per rule to stderr. A non-zero exit code is returned if any rule with `severity error` fails.

### Configuration

Plugin-specific options are stored under the `plugin_configs.fscheck` namespace and forwarded to the plugin at runtime. Set them with `ade config set` from the project root:

```sh
ade config set plugin_configs.fscheck.root-dir .
```

Pass `--global` to write the value to the user-level config instead of the project-level `.ade.yaml`.

| Config key                         | Required for | Description                                                                              |
| ---------------------------------- | ------------ | ---------------------------------------------------------------------------------------- |
| `plugin_configs.fscheck.root-dir`  | verify       | Directory against which glob patterns are resolved. Defaults to the working directory.   |

## Supported rules

Only `file` blocks are processed. `code` and `custom` blocks are skipped with a warning.

| ADE DSL assertion                   | Behaviour                                       |
| ----------------------------------- | ----------------------------------------------- |
| `path "..." must exist`             | at least one file matches the glob pattern      |
| `path "..." must not exist`         | no file matches the glob pattern                |
| `path "..." must contain "..."`     | at least one matching file contains the regex   |
| `path "..." must not contain "..."` | no matching file contains the regex             |

Rules with `severity warning` produce a `warn` line on failure but do not affect the exit code. Rules with `severity error` produce an `error` line and cause a non-zero exit.

## Glob syntax

Path patterns are slash-separated and support:

- `*`: matches any sequence of characters within a single path segment.
- `?`: matches a single character.
- `**`: matches zero or more directory levels.

## Documentation

See [docs/implementation.md](docs/implementation.md) for a high-level explanation of the code structure and implementation design.

## License

Licensed under the [Apache License, Version 2.0](./LICENSE).

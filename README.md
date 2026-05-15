# Filesystem Check Plugin for Architectural Decision Enforcement

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

An [ad-guidance-tool](https://github.com/adr/ad-guidance-tool) enforcement plugin that verifies `file` rules from the ADE DSL against the filesystem. Rules express glob-based existence and content assertions that are evaluated at runtime against a target directory.

## Installation

Install from a GitHub release:

```sh
adg enforce plugin install fscheck --repo github.com/phi42/adplugin-fscheck
```

Or build from source and register locally:

```sh
go build -o fscheck
adg enforce plugin install fscheck --path ./fscheck
```

## Usage

```sh
adg enforce verify -i path/to/adr.rule -p fscheck -d ./target-directory
```

## Supported rules

Only `file` blocks are processed. `code` blocks and `custom` blocks are skipped with a warning.

| ADE DSL assertion                       | Behaviour                                                       |
| --------------------------------------- | --------------------------------------------------------------- |
| `path "..." must exist`                 | at least one file matches the glob pattern                      |
| `path "..." must not exist`             | no file matches the glob pattern                                |
| `path "..." must contain "..."`         | at least one matching file contains the string                  |
| `path "..." must not contain "..."`     | no matching file contains the string                            |

Rule results are printed as `LEVEL  [rule-name] message`. Rules with `severity warning` produce a `WARN` line on failure but do not affect the exit code. Rules with `severity error` produce an `ERROR` line and cause a non-zero exit.

## License

Licensed under the [Apache License, Version 2.0](./LICENSE).

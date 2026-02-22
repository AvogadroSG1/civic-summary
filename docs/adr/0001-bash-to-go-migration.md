# Migrate Civic Summary Pipeline from Bash to Go

## Status

Accepted

## Context and Problem Statement

The original Hagerstown City Council summarization pipeline was implemented as a collection of bash scripts (`hagerstown-council-agent.sh` and supporting scripts). When we needed to add Washington County BOCC as a second government body, we faced a choice: extend the bash scripts or rewrite in a compiled language.

The bash pipeline had served well for a single body but presented challenges:

- Hardcoded paths throughout (output directories, playlist IDs, prompt files)
- Error handling limited to basic exit code checks and simple retry loops
- No unit testing — validation was manual
- Adding BOCC would require duplicating scripts with different parameters
- Date parsing was fragile string manipulation with `sed` and `grep`
- No structured metadata for quarantined (failed) meetings

## Decision Drivers

- Multi-body support without code duplication
- Testability with mock external tools (yt-dlp, whisper, claude)
- Config-driven extensibility — new bodies via YAML, not code
- Type safety for domain objects (Meeting, Transcript, Summary)
- Structured quarantine with JSON metadata for retry
- Single binary distribution via goreleaser

## Considered Options

1. Extend bash with functions and parameterization
2. Python CLI with Click/argparse
3. Go CLI with Cobra/Viper

## Decision Outcome

Chosen option: **Go CLI with Cobra/Viper**, because it provides type safety, single binary distribution, a mature CLI ecosystem (Cobra for commands, Viper for config), and the `CommandExecutor` interface pattern enables full unit testing of pipeline stages without invoking real binaries.

### Option 1: Extend Bash

Parameterize existing scripts to accept body-specific config. Pros: minimal rework, keeps working system. Cons: no type safety, no unit testing, date parsing remains fragile, error handling stays primitive, each new body still needs script-level wiring.

### Option 2: Python CLI

Use Click or argparse with subprocess calls. Pros: rapid development, Jinja2 for prompt templates, pytest for testing. Cons: runtime dependency on Python + venv, no single binary, subprocess mocking less ergonomic than Go interfaces, deployment complexity on macOS launchd.

### Option 3: Go CLI (chosen)

Cobra/Viper CLI with DDD domain types and a `CommandExecutor` interface for all external tool invocations. Pros: single binary, strong typing, `CommandExecutor` interface for mock-based testing, config-driven bodies, goreleaser for builds. Cons: Go template syntax less familiar than Jinja2, no REPL for prompt iteration, heavier initial investment than bash parameterization.

## Consequences

### Good

- **Config-driven bodies**: Adding a new government body requires only a YAML config block and a prompt template file — no code changes
- **87% service coverage**: 119 unit tests with mock executors validate all pipeline stages
- **Quarantine with metadata**: Failed meetings store structured JSON (error, stage, timestamp) enabling automated retry with `civic-summary quarantine retry`
- **Wikilink cross-referencing**: Automated Obsidian `[[wikilink]]` injection between chronologically adjacent meeting summaries
- **Single binary**: `go install` or goreleaser produces a self-contained binary for macOS (arm64 + amd64)
- **ISO date bug fix**: Structured date parsing with explicit format list caught and fixed a bug that bash `date` silently mishandled

### Bad

- **Go template syntax**: `text/template` is less intuitive than Jinja2 for prompt authors; prompt iteration requires recompiling or using template files
- **No REPL for prompt development**: Unlike Python where prompts can be iterated interactively, Go requires template files and CLI invocation
- **Higher initial investment**: The Go rewrite took substantially more effort than parameterizing existing bash scripts would have
- **External tool dependency remains**: The CLI still shells out to yt-dlp, whisper, and claude — these are not Go-native and require PATH configuration in launchd

## Related Decisions

- [ADR-0000: Use MADR](0000-use-madr.md)

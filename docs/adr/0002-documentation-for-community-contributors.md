# Documentation for Community Contributors

## Status

Accepted

## Context and Problem Statement

civic-summary is a well-tested Go CLI (~5,600 lines, 119 tests, 87% coverage) with strong architectural patterns (DDD, mockable executors, config-driven bodies, externalized prompts). However, there is no onboarding documentation — no README, no CONTRIBUTING guide, no example config, no prerequisite checker, no CI pipeline. The gap between "excellent code" and "a community member can fork this for their own city council" is entirely a documentation and tooling gap.

We want hobbyist programmers to be able to clone the repository, configure it for their own local government body, and contribute back — without needing to read source code to understand how things work.

## Decision Drivers

- Contributors MUST be able to add a new government body without reading source code
- Setup MUST validate that required tools are installed before a confusing build failure
- The project MUST have CI to maintain quality as community contributions arrive
- Documentation SHOULD be layered: quick start for users, deep dive for contributors
- The project SHOULD support Linux builds to reach the widest audience

## Considered Options

1. README-only documentation
2. Comprehensive documentation + tooling (chosen)
3. Interactive `civic-summary init` wizard

## Decision Outcome

Chosen option: **Comprehensive documentation + tooling**, because a README alone doesn't solve the prerequisite validation problem, doesn't provide CI, and doesn't give contributors enough guidance to understand the codebase patterns. An interactive wizard would require code changes and testing effort that could be deferred.

### Deliverables

1. **`config.example.yaml`** — Fully commented configuration reference
2. **`README.md`** — Pipeline diagram, prerequisites, quick start, CLI reference, config reference
3. **`CONTRIBUTING.md`** — Dev setup, project structure walkthrough, testing guide, code standards
4. **`scripts/check-prerequisites.sh`** — Validates Go, yt-dlp, claude, whisper, golangci-lint
5. **Makefile `check` and `setup` targets** — `make check` runs prerequisite validation, `make setup` creates config directory and copies example config
6. **`.github/workflows/ci.yaml`** — Test, lint, and cross-platform build matrix
7. **`.github/workflows/release.yaml`** — Goreleaser-based release on tag push
8. **`docs/architecture.md`** — System context, pipeline stages, domain model, testability design
9. **`docs/prompt-template-guide.md`** — Template variables, Go template syntax primer, customization guide
10. **`LICENSE`** — MIT license
11. **`.goreleaser.yaml` update** — Add Linux to build targets
12. **`internal/output/output.go` update** — Platform-safe notifications (no-op on Linux)

## Consequences

### Good

- A new contributor can go from `git clone` to a working build in under 5 minutes
- Adding a new government body is clearly documented as a config-only operation
- CI prevents regressions as community PRs arrive
- Linux support broadens the potential contributor and user base
- Architecture documentation makes the Commander pattern and pipeline stages discoverable
- The prompt template guide enables non-programmers to customize for their government body

### Bad

- Documentation must be maintained alongside code changes
- The prerequisite script is a best-effort check (e.g., it can't verify Claude CLI authentication)
- Linux builds compile but macOS-specific features (osascript notifications) are no-ops

## Related Decisions

- [ADR-0000: Use MADR](0000-use-madr.md)
- [ADR-0001: Bash to Go Migration](0001-bash-to-go-migration.md)

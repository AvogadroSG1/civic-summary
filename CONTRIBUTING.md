# Contributing to civic-summary

Thanks for your interest in contributing! This guide will help you get set up and explain how the codebase is organized.

## Development Setup

### Prerequisites

- Go 1.25+
- golangci-lint (for `make lint`)
- yt-dlp (for integration tests or manual testing)
- Claude CLI (for integration tests or manual testing)

### Getting Started

```bash
git clone https://github.com/AvogadroSG1/civic-summary.git
cd civic-summary

# Verify your environment
make check

# Build the binary
make build

# Run all tests
make test

# Run linter
make lint
```

## Project Structure

```
civic-summary/
├── main.go                     # Entry point — just calls cmd.Execute()
├── cmd/                        # CLI commands (one file per command)
│   ├── root.go                 # Root command, global flags
│   ├── process.go              # Full pipeline execution
│   ├── discover.go             # Phase 1: find new videos
│   ├── transcribe.go           # Phase 2: get transcripts
│   ├── analyze.go              # Phase 3: generate summaries
│   ├── crossref.go             # Phase 4: add wikilinks
│   ├── validate.go             # Phase 5: quality checks
│   ├── bodies.go               # Body management (list, show)
│   ├── quarantine.go           # Failed meeting management
│   ├── status.go               # Processing status overview
│   ├── version.go              # Version info
│   ├── completion.go           # Shell completions
│   └── helpers.go              # Shared helpers (config loading, pipeline wiring)
│
├── internal/
│   ├── config/                 # Configuration loading and validation
│   │   └── config.go           # Viper-based config with env var binding
│   │
│   ├── domain/                 # Core data types — the "nouns" of the system
│   │   ├── body.go             # Government body (city council, BOCC, etc.)
│   │   ├── meeting.go          # A single meeting with date and video ID
│   │   ├── transcript.go       # SRT transcript content
│   │   ├── summary.go          # Generated markdown summary
│   │   ├── quarantine.go       # Failed meeting metadata
│   │   ├── validation.go       # Validation issues (errors + warnings)
│   │   └── pipeline.go         # Pipeline stages and processing stats
│   │
│   ├── executor/               # Shell command abstraction — the key to testability
│   │   ├── executor.go         # Commander interface (Execute, ExecuteWithStdin)
│   │   ├── mock.go             # MockCommander for tests
│   │   ├── ytdlp.go            # yt-dlp wrapper
│   │   ├── whisper.go          # Whisper wrapper
│   │   └── claude.go           # Claude CLI wrapper
│   │
│   ├── markdown/               # Markdown processing utilities
│   │   ├── frontmatter.go      # YAML frontmatter parse/inject
│   │   ├── sanitize.go         # Strip Claude meta-commentary
│   │   └── wikilinks.go        # Obsidian [[wikilink]] generation
│   │
│   ├── output/                 # Terminal output and notifications
│   │   └── output.go           # Logging setup, formatting, macOS notifications
│   │
│   ├── retry/                  # Retry with exponential backoff
│   │   └── retry.go            # Generic retry logic
│   │
│   └── service/                # Pipeline services — the "verbs" of the system
│       ├── pipeline.go         # PipelineOrchestrator (wires all stages)
│       ├── discovery.go        # Find new videos from YouTube
│       ├── transcription.go    # Download/generate transcripts
│       ├── analysis.go         # Send transcript to Claude, get summary
│       ├── crossref.go         # Inject Obsidian wikilinks
│       ├── validation.go       # Validate summary quality
│       ├── quarantine.go       # Manage failed meetings
│       └── index.go            # Generate meeting index
│
├── templates/                  # Prompt templates (Go text/template format)
├── testdata/fixtures/          # Golden test data (config, SRT, summaries)
├── docs/                       # Architecture docs and ADRs
├── scripts/                    # Development helper scripts
└── support/                    # Deployment helpers (launchd)
```

## Testing Guide

### Running Tests

```bash
# All tests
make test

# Verbose output
go test ./... -v

# Single package
go test ./internal/service/ -v

# With coverage
make coverage
```

### How MockCommander Works

All external tool calls (yt-dlp, whisper, claude) go through the `Commander` interface defined in `internal/executor/executor.go`:

```go
type Commander interface {
    Execute(ctx context.Context, name string, args ...string) (*CommandResult, error)
    ExecuteWithStdin(ctx context.Context, stdin string, name string, args ...string) (*CommandResult, error)
}
```

In tests, we use `MockCommander` (`internal/executor/mock.go`) which lets you pre-configure responses for specific commands:

```go
mock := executor.NewMockCommander()

// Set up what yt-dlp should "return"
mock.OnCommand("yt-dlp --flat-playlist ...", &executor.CommandResult{
    Stdout: "video1\nTitle One\n",
}, nil)

// Pass mock into the service being tested
ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
discovery := service.NewDiscoveryService(ytdlp, cfg)

// Test the service — it calls yt-dlp through the mock
meetings, err := discovery.DiscoverNewMeetings(ctx, body)
```

This pattern means **no real binaries are needed for unit tests**. Tests run fast and are deterministic.

### Golden Fixtures

Test data lives in `testdata/fixtures/`. These files represent known-good inputs and outputs:

- `config.yaml` — Reference configuration
- `sample.srt` — Example SRT transcript
- `valid-summary.md` — Example valid summary output
- `playlist-output.txt` — Example yt-dlp playlist output

When adding new features, add corresponding test fixtures if the feature processes structured input/output.

### Writing a New Test

1. Create a test file next to the source file (e.g., `myservice_test.go`)
2. Set up a `MockCommander` with the expected command responses
3. Create the service with the mock
4. Call the method under test
5. Assert results with `testify`

```go
func TestMyFeature(t *testing.T) {
    mock := executor.NewMockCommander()
    mock.OnCommand("tool --flag", &executor.CommandResult{
        Stdout: "expected output",
    }, nil)

    svc := service.NewMyService(mock)
    result, err := svc.DoThing(context.Background())

    require.NoError(t, err)
    assert.Equal(t, "expected", result)
}
```

## Adding a New Pipeline Feature

Follow this pattern when adding functionality to the processing pipeline:

1. **Domain type** — If your feature introduces a new concept, add a type in `internal/domain/`
2. **Service** — Create a service in `internal/service/` with a constructor that accepts the required executors
3. **Tests** — Write tests using `MockCommander` in a `*_test.go` file alongside the service
4. **Wire into pipeline** — If it's a pipeline stage, add it to `PipelineOrchestrator` in `internal/service/pipeline.go`
5. **CLI command** — Add a Cobra command in `cmd/` that creates the service and calls it

## Adding a New Government Body

**No code changes are needed.** See the [README](README.md#adding-your-own-government-body) for the step-by-step process. It requires only:

1. A config block in `config.yaml`
2. A prompt template file in the templates directory

## Code Standards

- Run `make lint` before submitting — this runs golangci-lint with the project's configuration
- Code is formatted with `gofmt` (enforced by the linter)
- Follow existing patterns in the codebase — if you're unsure, look at how similar features are implemented
- Keep services focused — one service per pipeline concern
- All external tool calls MUST go through the `Commander` interface

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for agenda URL parsing
fix: handle empty playlist response gracefully
docs: add architecture decision record for retry strategy
test: add golden fixture for multi-section summary
chore: update golangci-lint to v2.x
```

All commits MUST include co-authorship:

```
feat: add PDF export support

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

## ADR Process

We use [MADR](https://adr.github.io/madr/) (Markdown Architectural Decision Records) for significant decisions. ADRs live in `docs/adr/`.

Write an ADR when:
- Choosing between multiple valid approaches
- Making a decision that would be hard to reverse
- Introducing a new external dependency
- Changing the pipeline architecture

See [ADR-0000](docs/adr/0000-use-madr.md) for the format and [ADR-0001](docs/adr/0001-bash-to-go-migration.md) for a real example.

To create a new ADR:

```bash
cp docs/adr/0000-use-madr.md docs/adr/NNNN-your-decision.md
# Edit with your decision context, options, and outcome
```

## Questions?

Open an issue on GitHub and we'll help you get started.

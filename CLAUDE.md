# CLAUDE.md - civic-summary

Go CLI that transforms YouTube recordings of government meetings into citizen-friendly Obsidian markdown summaries.

Module: `github.com/AvogadroSG1/civic-summary` · Go 1.25 · entrypoint `main.go` → `cmd.Execute()`

## Architecture

```
main.go                 # Thin entrypoint
cmd/                    # Cobra command tree
  root.go               #   persistent --config / --verbose flags, slog setup
  helpers.go            #   loadConfig, getBody, buildExecutors, buildPipeline (DI wiring)
  process.go            #   full pipeline (--body / --all / --dry-run)
  discover|transcribe|analyze|crossref|validate.go   # single-stage commands
  bodies.go             #   bodies list|show
  quarantine.go         #   quarantine list|retry|remove
  status.go completion.go version.go
internal/
  config/               # Viper loading, defaults, env binding, path helpers, validation
  domain/               # DDD types: Body, Meeting, Transcript, Summary,
                        #   QuarantineEntry/Manifest, ValidationResult, PipelineStage/Stats
  executor/             # Commander interface + OsCommander, MockCommander,
                        #   YtDlp / Whisper wrappers
  llm/                  # Client interface + New() factory; Anthropic-compatible
                        #   (/v1/messages) and OpenAI-compatible
                        #   (/v1/chat/completions) clients; error classification
  markdown/             # Frontmatter parse/inject/validate, model output sanitize, wikilinks
  output/               # slog setup, Banner/Success/Failure/Warning/Info, macOS notifications
  retry/                # Generic retry with configurable backoff delays
  service/              # Discovery, Transcription, Analysis, CrossReference, Validation,
                        #   Quarantine, Index, PipelineOrchestrator
templates/              # Go text/template prompt files (read at runtime, NOT embedded)
testdata/fixtures/      # Golden data: config.yaml, sample.srt, valid-summary.md, playlist-output.txt
docs/                   # architecture.md, prompt-template-guide.md, adr/ (MADR)
scripts/                # check-prerequisites.sh (make check)
support/launchd/        # macOS scheduling plist + install.sh
```

Dependency direction is strictly one-way: `cmd → service → {executor, llm, markdown, config, domain}`. `domain` imports nothing from the project. `config` and `llm` do not import each other — `domain.LLMConfig` is the shared type, and `cmd` does the wiring.

## Pipeline Stages

```
Discovery -> Transcription -> Analysis -> CrossReference -> Validation -> [write + index]
  (yt-dlp)    (captions,       (LLM HTTP    (date ->        (frontmatter,
               whisper           API)        wikilinks)       sections,
               fallback)                                      word count,
                                                              meta-commentary)
```

`PipelineOrchestrator.ProcessBody` (internal/service/pipeline.go) runs discovery once, then wraps each meeting's stages 2–5 in `retry.Do`. Failures are quarantined; previously quarantined items are retried at the end of every run; `IndexService.UpdateIndex` regenerates `index.md` last.

`retry.Do` short-circuits on errors that report `Permanent() bool == true` — because a retry re-downloads and re-transcribes the video, an unusable API key or model name must not cost three transcriptions. `internal/llm` errors satisfy that interface for authentication, unknown-model, context-window, and invalid-request failures.

Each stage is also reachable standalone via its own command, which is the fastest way to debug one stage without re-running the whole pipeline.

## Key Commands

```bash
make check              # Verify prerequisites (go, yt-dlp, an LLM API key, optional whisper/lint/goreleaser)
make setup              # check + scaffold ~/.civic-summary/{config.yaml,templates}
make build              # Build ./civic-summary with version ldflags
make install            # go install with same ldflags
make test               # go test ./... -v
make lint               # golangci-lint run
make coverage           # coverage.out + coverage.html + total
make clean              # remove binary and coverage artifacts
make release            # goreleaser snapshot build

go test ./internal/service -run TestPipeline -v    # single package / single test
```

## Configuration

- Search order: `--config` flag → `~/.civic-summary/config.yaml` → `./config.yaml`.
- Env overrides use the `CIVIC_SUMMARY_` prefix (`CIVIC_SUMMARY_OUTPUT_DIR`, `CIVIC_SUMMARY_YTDLP`, `CIVIC_SUMMARY_WHISPER`, `CIVIC_SUMMARY_WHISPER_MODEL`, `CIVIC_SUMMARY_LLM_PROVIDER`, `CIVIC_SUMMARY_LLM_MODEL`, `CIVIC_SUMMARY_LLM_BASE_URL`, `CIVIC_SUMMARY_LLM_API_KEY_ENV`, `CIVIC_SUMMARY_LLM_MAX_TOKENS`, `CIVIC_SUMMARY_LLM_MAX_TOKENS_FIELD`). 12-Factor compliant.
- `config.example.yaml` is the annotated reference and what `make setup` copies.
- Defaults: `log_retention_days: 90`, `max_retries: 3`, `backoff_delays: [5, 20, 60]`, `tools.ytdlp: yt-dlp`, and for the `llm` block `provider: anthropic`, `model: claude-opus-5`, `max_tokens: 16000`, `max_tokens_field: max_completion_tokens`, `timeout_seconds: 900`, `max_retries: 2`, `stream: true`. Whisper is optional — when unset, `TranscriptionService` has no fallback and caption-less videos fail.
- **The API key never lives in config.** `llm.api_key_env` names the environment variable to read (default `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` by provider); `llm.New` reads it and errors if empty. `Config.Validate()` deliberately does *not* require it, so credential-free commands (`bodies list`, `quarantine list`) keep working.
- **`llm.temperature` is unset by default and must stay that way** unless a user opts in: current Claude models (Opus 5, Sonnet 5, Opus 4.8/4.7) reject sampling parameters with HTTP 400.
- Bodies may override the global `llm` block with a `llm:` sub-block. `domain.LLMOverride` uses pointer fields so an omitted key inherits while an explicit zero (`stream: false`, empty `base_url`) is honoured; `Config.ResolveLLM(body)` performs the merge and is what `Validate()` checks.
- `Config.Validate()` requires `output_dir`, ≥1 body, and per body: `playlist_id` **or** `video_source_url`, plus `output_subdir`, `filename_pattern`, `title_date_regex`, `prompt_template`, and ≥1 tag.
- Body slugs come from the YAML map keys and are injected into `Body.Slug` at load time — never duplicate the slug inside the block.
- Derived paths (all in `internal/config/config.go`, use these rather than joining paths by hand):
  `BodyOutputDir` = `<output_dir>/<output_subdir>`, `FinalizedDir` = `…/Finalized Meeting Summaries`,
  `QuarantineDir` = `…/Automation/quarantine`, `LogDir` = `…/Automation/logs`,
  `TemplateDir()` = `~/.civic-summary/templates` if it exists, else `./templates`.

Adding a government body requires **only** a YAML block plus a prompt template — no code changes. Preserve that property.

## Design Decisions

- **Commander interface** — every shell-out goes through `executor.Commander`. `OsCommander` in production, `MockCommander` in tests. Never call `os/exec` outside `internal/executor`.
- **LLM over HTTP, two protocols only** — `internal/llm` has one client per wire protocol, not per vendor. Because both SDKs support `option.WithBaseURL`, those two cover the first-party APIs plus gateways (OpenRouter, Groq, Together, Azure) and self-hosted servers (Ollama, vLLM, LM Studio). Add a provider only for a genuinely different protocol.
- **Per-body LLM resolution** — `AnalysisService` takes a `service.LLMClientFor` resolver rather than a client, because the model can differ per body. `cmd/helpers.go` supplies the real one; tests pass a stub.
- **Streaming by default** — summaries exceed 1000 words, so `max_tokens` is large and a non-streaming request that size risks an SDK HTTP timeout. `stream: false` exists for compatible servers with unreliable SSE.
- **Externalized prompts** — `text/template` files loaded from `TemplateDir()` at runtime, not compiled into the binary, so contributors can iterate on prompts without rebuilding. Template variables are documented in `docs/prompt-template-guide.md` and defined by `service.PromptData`.
- **Config-driven bodies** — behavior differences between bodies live in YAML, not `switch` statements.
- **Same-date disambiguation** — `Meeting.Sequence` is 0 for a solo meeting on a date and 1..N (ordered by VideoID for determinism) when several share a date, producing `-1`/`-2` filename suffixes. `markdown.resolveTarget` resolves wikilinks to sequenced files.
- **Quarantine** — failures write `<QuarantineDir>/<videoID>/metadata.json` (plus transcript/partial output when available) and an entry in `manifest.json`, so runs are resumable.
- **Cross-references** — `markdown.AddWikilinks` rewrites human-readable dates to `[[target|original text]]` only when the target summary file exists on disk; self-links are skipped.
- **Output sanitization** — models sometimes prefix meta-commentary; `markdown.Sanitize` strips everything before the first frontmatter delimiter and validation rejects what slips through.
- **Structured logging** — `log/slog` text handler to stdout; `--verbose` switches to debug level. User-facing terminal output goes through `internal/output`, not raw `fmt` in services.

## Validation Rules

`ValidationService.Validate` (internal/service/validation.go) separates hard errors from advisory warnings:

- Errors: missing/invalid frontmatter, missing keys from `markdown.RequiredFrontmatterKeys` (`date`, `author`, `tags`, `source`, `meeting_date`), tags containing spaces, missing `## 1.`–`## 5.` sections, missing `#` title, `< 500` words, model meta-commentary in the body.
- Warnings: missing `## Conclusion`, missing attribution footer, `< 1000` words, no `[HH:MM:SS` timestamps.

`TranscriptionService.ValidateTranscript` separately rejects empty transcripts and anything under 500 words.

## Testing

- `go test ./...` needs no external binaries, no network, and no API key.
- `internal/llm` is tested against `httptest.Server` via `option.WithBaseURL`, which covers request shape, streaming accumulation, and error classification. `internal/service` drives `AnalysisService` through a stub `llm.Client` that records prompts — this is what makes the rendered prompt assertable (`MockCommander` used to discard stdin, so it was not).
- Unit tests drive services through `executor.NewMockCommander()`; configure with `OnCommand("<binary> <args...>", result, err)` and assert against `mock.Calls`.
- Golden fixtures live in `testdata/fixtures/`; `testdata` is excluded from lint.
- `cmd/` and `internal/output` have no tests — keep those packages thin and push logic into `internal/service` where it is testable.
- There are currently **no** build-tagged integration tests. If you add tests that shell out to real yt-dlp/whisper or call a real model endpoint, gate them behind `//go:build integration`.

## CI/CD

- `.github/workflows/ci.yaml` on push/PR to `main`: tests with `-race` + coverage, `golangci-lint-action@v7`, and a build matrix over `{darwin,linux} × {amd64,arm64}` with `CGO_ENABLED=0`.
- `.github/workflows/release.yaml` on `v*` tags: goreleaser (`.goreleaser.yaml`) publishes tar.gz archives + checksums.
- Version metadata is injected via ldflags into `cmd.version`/`commit`/`date`.

## Conventions

- Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `chore:`); include co-authorship trailers per `CONTRIBUTING.md`.
- Lint set: errcheck, govet, staticcheck, unused, ineffassign, gocritic, misspell, gofmt. Run `make lint` before pushing.
- Wrap errors with context: `fmt.Errorf("reading config: %w", err)`.
- Every exported identifier has a doc comment; every package has a package comment.
- Significant decisions get a MADR record in `docs/adr/` (see `docs/adr/0000-use-madr.md`).

## Dependencies

| Library | Purpose |
|---------|---------|
| spf13/cobra | CLI framework |
| spf13/viper | Config file + env loading |
| stretchr/testify | Test assertions |
| gopkg.in/yaml.v3 | YAML frontmatter parse/marshal |
| anthropics/anthropic-sdk-go | Anthropic-compatible Messages API client |
| openai/openai-go | OpenAI-compatible Chat Completions client |
| stdlib | text/template, os/exec, regexp, log/slog, encoding/json |

External binaries at runtime: `yt-dlp` (required), OpenAI `whisper` (optional fallback). Analysis needs network access to the configured endpoint plus an API key in the environment; `civic-summary status` probes both.

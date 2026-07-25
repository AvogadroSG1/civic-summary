#!/usr/bin/env bash
#
# check-prerequisites.sh — Validate that required and optional tools are available.
# Exit 0 if all required tools are found, 1 otherwise.

set -euo pipefail

REQUIRED_GO_VERSION="1.25"
PASS=0
FAIL=0
WARN=0

# Counters use arithmetic assignment rather than ((VAR++)): the latter evaluates
# to the pre-increment value, so the first call returns 1 and set -e aborts.
pass() { PASS=$((PASS + 1)); printf "  \033[32m✓\033[0m %s\n" "$1"; }
fail() { FAIL=$((FAIL + 1)); printf "  \033[31m✗\033[0m %s\n" "$1"; }
warn() { WARN=$((WARN + 1)); printf "  \033[33m⚠\033[0m %s\n" "$1"; }
info() { printf "  \033[34mℹ\033[0m %s\n" "$1"; }

echo "civic-summary prerequisite check"
echo "================================"
echo ""

# ── Required Tools ───────────────────────────────────────────────────────────

echo "Required:"

# Go
if command -v go &>/dev/null; then
    go_version=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
    if printf '%s\n%s\n' "$REQUIRED_GO_VERSION" "$go_version" | sort -V -C; then
        pass "Go ${go_version} (>= ${REQUIRED_GO_VERSION})"
    else
        fail "Go ${go_version} found, but >= ${REQUIRED_GO_VERSION} required"
    fi
else
    fail "Go not found — install from https://go.dev/dl/"
fi

# yt-dlp
if command -v yt-dlp &>/dev/null; then
    pass "yt-dlp ($(yt-dlp --version 2>/dev/null || echo 'unknown version'))"
else
    fail "yt-dlp not found — install: brew install yt-dlp (macOS) or pip install yt-dlp"
fi

# Language model API key
#
# Summaries are generated over HTTP against whichever provider the config's llm
# block names, so the definitive check is 'civic-summary status'. All this script
# can do without parsing YAML is confirm that some credential is exported.
if [ -n "${CIVIC_SUMMARY_LLM_API_KEY_ENV:-}" ] && [ -n "${!CIVIC_SUMMARY_LLM_API_KEY_ENV:-}" ]; then
    pass "LLM API key set in \$${CIVIC_SUMMARY_LLM_API_KEY_ENV}"
elif [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    pass "LLM API key set in \$ANTHROPIC_API_KEY"
elif [ -n "${OPENAI_API_KEY:-}" ]; then
    pass "LLM API key set in \$OPENAI_API_KEY"
else
    fail "No LLM API key found — export the variable named by llm.api_key_env (default \$ANTHROPIC_API_KEY or \$OPENAI_API_KEY)"
fi

echo ""

# ── Optional Tools ───────────────────────────────────────────────────────────

echo "Optional:"

# Whisper
if command -v whisper &>/dev/null || command -v whisper-cli &>/dev/null; then
    pass "whisper (fallback transcription)"
else
    info "whisper not found — only needed if videos lack captions"
fi

# golangci-lint
if command -v golangci-lint &>/dev/null; then
    pass "golangci-lint ($(golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo 'unknown'))"
else
    info "golangci-lint not found — only needed for development (make lint)"
fi

# goreleaser
if command -v goreleaser &>/dev/null; then
    pass "goreleaser"
else
    info "goreleaser not found — only needed for release builds (make release)"
fi

echo ""

# ── Configuration ────────────────────────────────────────────────────────────

echo "Configuration:"

config_dir="$HOME/.civic-summary"
config_file="$config_dir/config.yaml"
templates_dir="$config_dir/templates"

if [ -d "$config_dir" ]; then
    pass "Config directory exists: $config_dir"
else
    warn "Config directory missing: $config_dir — run 'make setup' to create it"
fi

if [ -f "$config_file" ]; then
    pass "Config file exists: $config_file"
else
    warn "Config file missing: $config_file — run 'make setup' to create from example"
fi

if [ -d "$templates_dir" ]; then
    pass "Templates directory exists: $templates_dir"
else
    warn "Templates directory missing: $templates_dir — run 'make setup' to create it"
fi

info "Run 'civic-summary status' to verify the configured model is reachable"

echo ""

# ── Summary ──────────────────────────────────────────────────────────────────

echo "────────────────────────────────"
printf "  Passed: %d  |  Failed: %d  |  Warnings: %d\n" "$PASS" "$FAIL" "$WARN"
echo "────────────────────────────────"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "Fix the failures above before proceeding."
    exit 1
fi

echo ""
echo "All required prerequisites met."
exit 0

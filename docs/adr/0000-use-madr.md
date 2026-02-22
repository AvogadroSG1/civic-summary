# Use MADR for Architectural Decision Records

## Status

Accepted

## Context and Problem Statement

We need a consistent format for documenting architectural decisions in civic-summary. Without a standard format, decisions are scattered across commit messages, comments, and tribal knowledge.

## Decision Drivers

- Decisions MUST be discoverable and reviewable by future contributors
- Format SHOULD be lightweight enough to encourage regular use
- Records SHOULD capture rationale, alternatives, and trade-offs

## Considered Options

1. MADR (Markdown Architectural Decision Records)
2. Nygard-style ADRs (original Michael Nygard format)
3. No formal ADR process

## Decision Outcome

Chosen option: **MADR**, because it provides structured sections (Context, Decision Drivers, Options, Consequences) in a markdown format that lives alongside the code. The template encourages documenting alternatives and trade-offs without excessive ceremony.

Reference: <https://adr.github.io/madr/>

## Consequences

### Good

- Decisions are version-controlled alongside code
- Structured format ensures consistent documentation of alternatives and trade-offs
- Markdown renders natively on GitHub and in Obsidian

### Bad

- Requires discipline to create records for significant decisions
- Adds a small overhead to the decision-making process

# Prompt Template Guide

This guide explains how to create and customize prompt templates for civic-summary. Each government body uses a prompt template to instruct Claude on how to generate meeting summaries.

## Template Variables

Templates use Go's `text/template` syntax. The following variables are available:

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{.BodyName}}` | string | Display name of the government body | `Hagerstown City Council` |
| `{{.MeetingDateHuman}}` | string | Human-readable meeting date | `February 04, 2025` |
| `{{.MeetingDateISO}}` | string | ISO 8601 date | `2025-02-04` |
| `{{.MeetingType}}` | string | Type of meeting | `Regular Session` |
| `{{.VideoID}}` | string | YouTube video ID | `dQw4w9WgXcQ` |
| `{{.VideoURL}}` | string | Full YouTube watch URL | `https://www.youtube.com/watch?v=dQw4w9WgXcQ` |
| `{{.AgendaURL}}` | string | Agenda URL (may be empty) | `https://example.com/agenda.pdf` |
| `{{.Transcript}}` | string | Full SRT transcript content | *(multi-line SRT text)* |
| `{{.TodayDate}}` | string | Today's date (ISO format) | `2025-02-05` |
| `{{.Author}}` | string | Author name from body config | `Peter O'Connor` |
| `{{.Tags}}` | []string | Tag list from body config | `[City-Council, Hagerstown]` |
| `{{.FooterText}}` | string | Footer text from body config | `This citizen summary was created...` |

## Go Template Syntax Primer

If you're new to Go templates, here are the five constructs you'll use:

### 1. Variable Insertion

```
The meeting was on {{.MeetingDateHuman}}.
```

### 2. Conditionals

```
{{if .AgendaURL}}Agenda: {{.AgendaURL}}{{else}}No agenda available.{{end}}
```

### 3. Range (Loops)

```
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}
```

The `{{.}}` inside a range refers to the current element.

### 4. Whitespace Control

A hyphen trims whitespace on that side:

```
{{- .BodyName -}}       ← trims whitespace on both sides
{{- range .Tags}}       ← trims whitespace before the range
{{- end}}               ← trims whitespace before end
```

### 5. Default Values

Use `if` to provide fallbacks:

```
{{if .FooterText}}{{.FooterText}}{{else}}Default footer text here.{{end}}
```

## Creating a Template

### Step 1: Copy an Existing Template

Start from one of the included templates:

```bash
cp templates/hagerstown.prompt.tmpl ~/.civic-summary/templates/my-council.prompt.tmpl
```

### Step 2: Customize the Sections

The template is a prompt that instructs Claude on what to produce. The key sections to customize are:

**Meeting context** — Adjust the role and body-specific language:
```
You are an expert civic engagement analyst specializing in local government meeting analysis.

**Task**: Generate a comprehensive "Citizen Summary" for a {{.BodyName}} meeting.
```

**Section names** — Rename sections to match your body's terminology:
- "Citizen Comments" vs "Public Comments" vs "Public Hearing"
- "Actions Taken" vs "Consent Agenda" vs "Roll Call Votes"
- "Input Requested from Council" vs "Input Requested from Commissioners"

**Section guidance** — Adjust the instructions within each section to reflect the types of business your body handles:
```
## 3. Actions Taken

[Document all votes, approvals, ordinances, resolutions, financial decisions]
- Categorize by type (e.g., "Consent Agenda", "Contract Awards", "Budget Items")
```

**Audience framing** — Change the audience description:
```
**Audience**: Write for general Washington County residents, not government insiders.
```

### Step 3: Reference in Config

Point your body's config to the new template:

```yaml
bodies:
  my-council:
    prompt_template: my-council.prompt.tmpl
```

### Step 4: Test

Run analysis on a single video to verify your template produces good results:

```bash
civic-summary analyze VIDEO_ID --body=my-council --date=2025-02-04
```

Review the output and iterate on the template as needed.

## Tips for Better Summaries

### Section Structure

Keep the numbered section structure (1-5) consistent. Claude follows it reliably:

1. **Updates** — Administrative/informational items
2. **Citizen/Public Comments** — Public testimony
3. **Actions Taken** — Votes, approvals, financial decisions
4. **Input Requested** — Items needing elected official guidance
5. **Critical Discussions** — Deep-dive analysis with "why this matters"

### Audience Targeting

Be explicit about who the audience is. Include phrases like:
- "Write for general citizens, not government insiders"
- "Explain jargon and acronyms"
- "Use exact dollar amounts, dates, and proper names"

### Timestamp Instructions

Always include timestamp format requirements. This helps citizens find specific moments in the video:
```
Include timestamps in format **[HH:MM:SS-HH:MM:SS]** or **[HH:MM:SS]** for ALL major topics.
```

### Output Format Control

The `CRITICAL INSTRUCTIONS` section at the end of the template is important — it prevents Claude from adding preamble text before the markdown. Always include:
```
**OUTPUT FORMAT**: Start your response with exactly "---" (the YAML frontmatter opener).
Do not include ANY text before this.
```

### Template Variables in Frontmatter

The frontmatter section in your template defines the metadata for the generated markdown. Make sure it includes:

```
---
date: {{.TodayDate}}
author: {{.Author}}
tags:
{{- range .Tags}}
  - {{.}}
{{- end}}
source: {{.VideoURL}}
meeting_date: {{.MeetingDateISO}}
---
```

The validation stage checks for these fields, so don't remove them.

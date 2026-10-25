# Scrute — Local AI PR Reviewer

> Review your diff before your teammates do—using your own API key, on your machine.

**Home:** [scrute.rejisterjack.com](https://scrute.rejisterjack.com)

## One-line idea

A Go CLI that reads your git diff, sends it to your chosen AI provider with your own API key, and returns a structured code review before you push—without sending your repo to a third-party review service.

## The problem

Many AI code review tools are subscription-based and run after you push, when fixing issues is most expensive. Developers want a review before push: local, fast, and keyed with their own credentials so source never sits on someone else’s servers. Scrute runs in the repo, reads the diff plus enough context to understand it, and returns a structured review with issues ranked by severity.

## What Scrute does

- Reads your staged git diff and surrounding file context
- Calls Claude or GPT using your API key from the environment
- Returns a structured review: summary, issues with severity (critical, warning, suggestion), and line references
- Configurable via `.scrute.yaml`: model, excluded paths, review strictness
- Output to the terminal or a markdown file for PR attachment

## Who it's for

- Solo developers who want a second opinion before push
- Teams without heavy review culture who still want automated pre-flight checks
- Anyone who wants CI-adjacent signal before CI runs, without another SaaS subscription

## How you use it

1. Install: `go install github.com/rejisterjack/scrute@latest`
2. Set your API key: `export ANTHROPIC_API_KEY=...` (or OpenAI equivalent)
3. Stage changes with `git add`
4. Run `scrute` at the repo root
5. Fix what matters, then push

## How it runs

- Local CLI only—nothing is hosted by Scrute
- Reads git state from the local repository
- Calls the AI API with the developer’s key; no Scrute backend receives code
- Requires network only for the model API call

## Tech stack

- Go (CLI)
- Git diff parsing (go-git)
- Anthropic and OpenAI APIs (caller-supplied keys)
- YAML configuration per repository

## Success criteria

- Reliable `go install` path documented in the README
- Review output: summary, severities, line references
- Working `.scrute.yaml` for model and exclusions
- Anthropic and OpenAI key support
- Demo in README showing diff → review in the terminal
- Open source at [github.com/rejisterjack/scrute](https://github.com/rejisterjack/scrute)

## Limitations & non-goals

- Review quality depends on model and prompts; Scrute is not a substitute for human review
- Large diffs are slow and costly; enforce limits and warn users
- Very large files may need chunking due to context windows
- Not a hosted PR bot, policy engine, or security scanner replacement

## Distribution

| Artifact | Channel |
|----------|---------|
| CLI | `go install`, GitHub releases (Linux, macOS, Windows) |

No server, database, or hosted component is required.

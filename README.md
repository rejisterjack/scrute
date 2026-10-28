# Scrute

Review your diff before your teammates do — using your own API key, on your machine.

Scrute is a local CLI. It reads the staged git diff and nearby file context, sends that to Claude or GPT with your key, and prints a structured review: a summary, issues ranked `critical`, `warning`, or `suggestion`, and file:line references. Nothing is hosted by Scrute. The only network call is the one to the model API you chose.

Scrute is not a substitute for human review, a hosted PR bot, or a security scanner.

## Install

Go:

```sh
go install github.com/rejisterjack/scrute@latest
```

Prebuilt binaries for Linux, macOS, and Windows (amd64 and arm64) are attached to [GitHub releases](https://github.com/rejisterjack/scrute/releases).

## Review a change

1. Set a key: `export ANTHROPIC_API_KEY=...` or `export OPENAI_API_KEY=...`
2. Stage changes with `git add`
3. Run `scrute` at the repository root
4. Fix what matters, then push

```sh
git add .
scrute
scrute --provider openai
scrute --format markdown --out REVIEW.md
```

`scrute init` writes a `.scrute.yaml`. `scrute hook install` adds an opt-in pre-push hook that aborts the push when scrute exits non-zero. `scrute hook uninstall` removes that hook and leaves any other hook alone. On Windows the hook is a POSIX shell script and runs under Git for Windows. It does not run in cmd.exe. The Windows zip from the release is the scrute binary the hook runs.

## Example

```text
$ scrute
scrute  anthropic/claude-sonnet-5

Summary
  The change drops a nil check before the map lookup.

Critical
  internal/pay.go:42  Nil map lookup
    m[k] panics when m is nil.

Warning
  main.go:10  Ignored error
    WriteTo error is discarded.

1 critical, 1 warning
```

That transcript is the real terminal layout. The findings are an example of the shape, not a recorded model call.

## Configuration

`.scrute.yaml` in the repository root. A missing file uses these defaults:

```yaml
provider: anthropic          # anthropic | openai
model: ""                    # empty uses the built-in default for the provider
strictness: balanced         # lenient | balanced | strict
context_lines: 40
max_diff_bytes: 200000
fail_on: warning             # none | suggestion | warning | critical
exclude:
  - ".env"
  - ".env.*"
  - "**/*.pem"
  - "vendor/**"
  - "node_modules/**"
  - "dist/**"
```

Flags `--provider`, `--model`, and `--strictness` override the file for one run. An `exclude` list in the file replaces the defaults, so copy the entries you still want. Binaries and gitlinks are never sent.

Strictness only changes the prompt. `lenient` asks for likely bugs and security mistakes. `balanced` adds correctness and API misuse. `strict` also asks for maintainability notes.

Default models are `claude-sonnet-5` and `gpt-5.4`. Set `model` when you want a different id. Keys are read only from the environment and are stripped from error text.

## Exit codes

- `0` — review finished, and nothing is at or above `fail_on`
- `1` — review finished, with issues at or above `fail_on`
- `2` — usage, git, or config error, or a text file was omitted because it exceeded `max_diff_bytes`
- `3` — the model request failed, or the response was still not a valid review after one retry

`NO_COLOR` or `TERM=dumb` turns off ANSI color. `--verbose` prints the files sent and the token counts. It does not print the key.

## Limits

Large diffs are split by file and reviewed in chunks. A single file bigger than `max_diff_bytes` is omitted and the command exits `2` when that was the only reason to fail, so a pre-push hook does not treat an unread file as clean. Raise the limit or review that file yourself.

Review quality depends on the model and the prompt.

## Development

```sh
go test -race ./...
```

GitHub Actions runs the race tests, `golangci-lint`, and `govulncheck` on `dev` and `main`. Provider tests use a local HTTP server. Tagging `v*` is what publishes a release. The tagged commit must already be on `main`.

## License

MIT. See [LICENSE](LICENSE).

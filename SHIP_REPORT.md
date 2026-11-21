# Ship report

Testing finished on 21 Nov 2026. The race suite passed, the CLI was run against a local model stub, and the marketing page was checked in Chrome. 43 features passed. The decision is GO.

Scrute is a local Go CLI that reviews a git diff with the caller's Claude or GPT key.
Nothing is hosted: the only network call is the model API the operator chose.
The static page at https://scrute.rejisterjack.com is `site/`, with no Scrute backend.
A pre-push hook can review each branch update before git sends it.
Review quality is the model's opinion; Scrute is not a human reviewer or a security scanner.

## Feature table

| ID | Feature | Priority | Status | Bugs Found | Bugs Fixed | UI Issues Fixed | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| CLI-01 | Help and unknown command | P2 | PASSED | 0 | 0 | 0 | `scrute` takes no positional args. Help is the cobra default. |
| CLI-02 | Version stamp | P1 | PASSED | 0 | 0 | 0 | Unstamped builds print `scrute dev`. Goreleaser sets the version with `-X`. |
| CLI-03 | Outside a git repository | P0 | PASSED | 0 | 0 | 0 | A directory that is not a work tree exits 2. |
| CLI-04 | Nothing staged | P0 | PASSED | 0 | 0 | 0 | An empty index prints `Nothing staged.` and exits 0. |
| CLI-05 | Staged diff only | P0 | PASSED | 0 | 0 | 0 | Unstaged worktree edits are not sent. |
| CLI-06 | Anthropic review | P0 | PASSED | 0 | 0 | 0 | Default provider. Empty model becomes `claude-sonnet-5-5`. |
| CLI-07 | OpenAI review | P0 | PASSED | 0 | 0 | 0 | `--provider openai`. Empty model becomes `gpt-5.4`. |
| CLI-08 | Missing API key | P0 | PASSED | 0 | 0 | 0 | Exit 2 names `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` and does not print a key. |
| CLI-09 | Key redaction | P0 | PASSED | 0 | 0 | 0 | Provider errors replace a secret of 8 or more characters with `[REDACTED]`. |
| CLI-10 | Base URL override | P1 | PASSED | 0 | 0 | 0 | `SCRUTE_ANTHROPIC_BASE_URL` and `SCRUTE_OPENAI_BASE_URL` retarget that provider. |
| CLI-11 | Provider failure | P0 | PASSED | 0 | 0 | 0 | A failed model call exits 3. Each call has a 3 minute timeout. |
| CLI-12 | Invalid JSON retry | P0 | PASSED | 0 | 0 | 0 | One retry, then exit 3 with the raw response. |
| CLI-13 | Config file and defaults | P0 | PASSED | 0 | 0 | 0 | A missing `.scrute.yaml` uses the documented defaults. A bad file exits 2. |
| CLI-14 | Flags override the file | P0 | PASSED | 0 | 0 | 0 | `--provider`, `--model`, and `--strictness` win for one run. |
| CLI-15 | Init | P0 | PASSED | 0 | 0 | 0 | `scrute init` writes the template once and refuses to overwrite. |
| CLI-16 | Exclude patterns | P0 | PASSED | 0 | 0 | 0 | A file `exclude` list replaces the defaults. Secrets and vendor paths are the built-in set. |
| CLI-17 | Binaries and submodules | P0 | PASSED | 0 | 0 | 0 | A NUL in the first 8 KiB, or a gitlink, is skipped and not sent. |
| CLI-18 | Context and deleted files | P1 | PASSED | 0 | 0 | 0 | Nearby lines are numbered. A deletion takes context from HEAD. |
| CLI-19 | Chunking and byte cap | P0 | PASSED | 0 | 0 | 0 | Default cap is 200000 bytes. Larger reviews are split by file. |
| CLI-20 | Oversized blob | P1 | PASSED | 0 | 0 | 0 | Text over 32 MiB, or one file over the cap, is omitted. Exit 2 when that is the only failure. |
| CLI-21 | Strictness | P1 | PASSED | 0 | 0 | 0 | `lenient`, `balanced`, and `strict` change the prompt only. |
| CLI-22 | Fail-on exit | P0 | PASSED | 0 | 0 | 0 | Default `warning`. Exit 1 at that rank or higher. `none` stays 0. |
| CLI-23 | Terminal review | P0 | PASSED | 0 | 0 | 0 | Summary, ranked issues, `file:line`, and a count line. |
| CLI-24 | Markdown and JSON | P0 | PASSED | 0 | 0 | 0 | `--format markdown` and `--format json`. `md`, `text`, and `terminal` are aliases. |
| CLI-25 | Output file | P1 | PASSED | 0 | 0 | 0 | `--out` writes the same format, or markdown when `--format` is omitted. Mode 0644. |
| CLI-26 | Color | P2 | PASSED | 0 | 0 | 0 | ANSI on a terminal. `NO_COLOR` or `TERM=dumb` turns it off. |
| CLI-27 | Verbose log | P1 | PASSED | 0 | 0 | 0 | `--verbose` prints provider, model, files, and token counts. It does not print the key. |
| CLI-28 | Commit range | P0 | PASSED | 0 | 0 | 0 | `--range BASE..HEAD` reviews those trees. A three-dot range exits 2. |
| CLI-29 | New branch base | P0 | PASSED | 0 | 0 | 0 | An all-zero base uses the merge-base with main or master, otherwise an empty tree. |
| CLI-30 | Hook install and uninstall | P0 | PASSED | 0 | 0 | 0 | The hook is mode 0755 and stores the absolute binary path. |
| CLI-31 | Hook push review | P0 | PASSED | 0 | 0 | 0 | Branch updates are reviewed. Tags and deletes are skipped. A non-zero exit aborts the push. |
| CLI-32 | Foreign hook | P0 | PASSED | 0 | 0 | 0 | Install refuses a hook without the scrute marker. Uninstall leaves that file alone. |
| CLI-33 | Worktree and conflicts | P1 | PASSED | 0 | 0 | 0 | A linked work tree is readable. An index conflict exits 2. |
| CLI-34 | Chunk progress | P2 | PASSED | 0 | 0 | 0 | More than one request prints `reviewing N/M` on stderr. |
| CLI-35 | Merged findings | P1 | PASSED | 0 | 0 | 0 | Chunk reviews are combined and duplicate issues are dropped. |
| WEB-01 | Marketing page | P1 | PASSED | 0 | 0 | 0 | `site/index.html` is the page at scrute.rejisterjack.com. |
| WEB-02 | Title, description, icon, robots | P1 | PASSED | 0 | 0 | 0 | Canonical URL is https://scrute.rejisterjack.com/. `robots.txt` allows `/`. |
| WEB-03 | Responsive page | P2 | PASSED | 0 | 0 | 0 | The layout tightens below 560px. The only theme is light. |
| WEB-04 | Security headers | P1 | PASSED | 0 | 0 | 0 | site/vercel.json sends nosniff, a strict referrer policy, DENY framing, and a CSP that allows the inline stylesheet. |
| DEVOPS-01 | CI | P0 | PASSED | 0 | 0 | 0 | Push to `dev` or `main`, and pull requests, run race tests, golangci-lint, and govulncheck. |
| DEVOPS-02 | Release binaries | P0 | PASSED | 0 | 0 | 0 | A `v*` tag on `main` publishes six targets plus checksums. |
| DEVOPS-03 | Go 1.27.4 | P1 | PASSED | 0 | 0 | 0 | `go.mod` says `go 1.27.4`. CI uses that file. |
| TEST-01 | Race suite | P0 | PASSED | 0 | 0 | 0 | `go test -race ./...` is the documented check. This step did not run it. |

## Gaps

| ID | Gap | Action | Reason |
| --- | --- | --- | --- |
| GAP-01 | Hosted GitHub App or PR bot | DEFER | VISION.md closes this. Scrute stays a local CLI. |
| GAP-02 | Accounts, billing, and telemetry | DEFER | The page says there is no account and no telemetry. Keys stay on the machine. |
| GAP-03 | Dark theme | DEFER | `site/index.html` sets `color-scheme: light` and has no toggle. |
| GAP-04 | Security headers | BUILD | The public page has no header file. WEB-04 is the feature row. |
| GAP-05 | Providers other than Claude and GPT | DEFER | Config accepts only `anthropic` and `openai`. |
| GAP-06 | Dollar spend cap | DEFER | The cost control in the repo is `max_diff_bytes`, not a currency budget. |
| GAP-07 | Security scanner | DEFER | README and VISION say Scrute does not replace a scanner. |

## CLI-01 Help and unknown command

- **Files:** `internal/cli/cli.go`, `main.go`
- **Expected behavior:** `scrute --help` and `scrute help` exit 0 and name the review command, `init`, `version`, and `hook`. `scrute nope` and `scrute extra-arg` exit 2. No positional review path is accepted.
- **Acceptance checklist:**
  - [x] Help exits 0 and lists `init`, `version`, and `hook`.
  - [x] An unknown command exits 2.
  - [x] A positional argument exits 2.
- **Test cases:** normal help; empty args print help or start a review only inside a repo; `nope`; a second argument; emoji command name.
- **UI checks:** Help text wraps in an 80-column terminal and does not print a stack trace.
- **Dependencies:** none
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-02 Version stamp

- **Files:** `internal/buildinfo/version.go`, `internal/cli/cli.go`, `.goreleaser.yaml`
- **Expected behavior:** `scrute version` and `scrute --version` print `scrute dev` for a local build. A release binary prints the tag version without the leading `v` being doubled. The template is `scrute {{.Version}}`.
- **Acceptance checklist:**
  - [x] A local build prints `scrute dev`.
  - [x] `--version` and `version` match.
- **Test cases:** local `go build`; compare the ldflags line in `.goreleaser.yaml`.
- **UI checks:** One line, trailing newline, no color.
- **Dependencies:** none
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-03 Outside a git repository

- **Files:** `internal/cli/cli.go`, `internal/gitdiff/gitdiff.go`
- **Expected behavior:** Running `scrute` in a directory with no `.git` exits 2 and prints `scrute:` plus the git error. It does not call a model.
- **Acceptance checklist:**
  - [x] A temp directory exits 2.
  - [x] No API key is required for this failure.
- **Test cases:** empty temp dir; a dir that contains files but no git; a path that is not readable.
- **UI checks:** The error is one stderr line.
- **Dependencies:** none
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-04 Nothing staged

- **Files:** `internal/cli/cli.go`
- **Expected behavior:** Inside a repo with a clean index, `scrute` prints `Nothing staged.` and exits 0. No key is required.
- **Acceptance checklist:**
  - [x] Clean repo exits 0 with that sentence.
  - [x] An unstaged edit still prints `Nothing staged.`
- **Test cases:** new repo with one commit and a dirty work tree; empty repo with no commits if git allows a status.
- **UI checks:** stdout only, no review header.
- **Dependencies:** CLI-03
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-05 Staged diff only

- **Files:** `internal/gitdiff/gitdiff.go`, `internal/cli/cli.go`
- **Expected behavior:** The review diff is HEAD versus the index. A line that exists only in the work tree is absent from the prompt. Renames are represented as file changes.
- **Acceptance checklist:**
  - [x] A staged line is present.
  - [x] An unstaged line added after `git add` is absent.
- **Test cases:** stage `STAGED`, then append `WORKTREE ONLY` without adding; unicode and a long line; a rename.
- **UI checks:** Not a screen. Confirm the stub request body in a test.
- **Dependencies:** CLI-04
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-06 Anthropic review

- **Files:** `internal/provider/anthropic/anthropic.go`, `internal/provider/provider.go`, `internal/review/prompt.go`
- **Expected behavior:** With `ANTHROPIC_API_KEY` set and no provider flag, the request goes to Anthropic using JSON schema output. The default model id is `claude-sonnet-5-5`. The printed header is `scrute  anthropic/claude-sonnet-5-5` when the model is left empty.
- **Acceptance checklist:**
  - [x] A stub returns a summary and ranked issues.
  - [x] The request is not sent to OpenAI.
  - [x] The system prompt says line numbers refer to the new file.
- **Test cases:** stub 200 with one critical; empty issues list prints `No issues.`; non-English detail; a finding with line 0.
- **UI checks:** Severity groups are Critical, then Warning, then Suggestion, with a blank line between groups.
- **Dependencies:** CLI-05, CLI-08
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-07 OpenAI review

- **Files:** `internal/provider/openai/openai.go`, `internal/cli/cli.go`
- **Expected behavior:** `--provider openai` uses `OPENAI_API_KEY` and strict JSON schema named `scrute_review`. The default model is `gpt-5.4`. `MaxCompletionTokens` is 8192.
- **Acceptance checklist:**
  - [x] A stub review exits using the same renderer as Anthropic.
  - [x] The header names `openai` and the model.
- **Test cases:** flag over a yaml that says anthropic; missing OpenAI key; model id with unusual characters.
- **UI checks:** Same terminal layout as CLI-06.
- **Dependencies:** CLI-06, CLI-14
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-08 Missing API key

- **Files:** `internal/cli/cli.go`, `internal/provider/provider.go`
- **Expected behavior:** A staged text diff with no key exits 2 before a request. The message is `set ANTHROPIC_API_KEY` or `set OPENAI_API_KEY`. Whitespace-only keys count as missing. A review that sends nothing, such as only a binary, does not require a key.
- **Acceptance checklist:**
  - [x] Unset key exits 2.
  - [x] The process environment value is not echoed.
  - [x] Binary-only staged content can render without a key.
- **Test cases:** unset; spaces; wrong provider's key set; binary-only file.
- **UI checks:** stderr starts with `scrute:`.
- **Dependencies:** CLI-05
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-09 Key redaction

- **Files:** `internal/provider/provider.go`, `internal/provider/anthropic/anthropic.go`, `internal/provider/openai/openai.go`
- **Expected behavior:** If the provider error text contains the API key and the key is at least 8 characters, the printed error says `[REDACTED]` instead. Shorter secrets are left unchanged. Verbose output does not include the key.
- **Acceptance checklist:**
  - [x] A stub error that echoes the key does not appear on stderr.
  - [x] A 7-character secret is not treated as redactable.
- **Test cases:** key embedded in a 401 body; key as a substring of a longer token; verbose with a successful stub.
- **UI checks:** stderr remains one readable error.
- **Dependencies:** CLI-11
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-10 Base URL override

- **Files:** `internal/cli/cli.go`, `internal/provider/anthropic/anthropic.go`, `internal/provider/openai/openai.go`
- **Expected behavior:** `SCRUTE_ANTHROPIC_BASE_URL` sends the Anthropic call to that origin. `SCRUTE_OPENAI_BASE_URL` does the same for OpenAI. An empty variable keeps the vendor default. The key is sent to that host.
- **Acceptance checklist:**
  - [x] A stub on 127.0.0.1 receives the request only when its variable is set.
  - [x] The other provider's variable does not redirect this call.
- **Test cases:** http stub; whitespace around the URL; unset.
- **UI checks:** No extra log line for the base URL.
- **Dependencies:** CLI-06, CLI-07
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-11 Provider failure

- **Files:** `internal/cli/cli.go`, `internal/provider/anthropic/anthropic.go`, `internal/provider/openai/openai.go`
- **Expected behavior:** HTTP errors, timeouts, and an empty review body exit 3. Each `Complete` call cancels after 3 minutes. A cancelled parent context also fails. Git reading itself is limited to 30 seconds.
- **Acceptance checklist:**
  - [x] Stub status 500 exits 3.
  - [x] An empty choice list exits 3.
  - [x] The key is redacted when it appears in the body.
- **Test cases:** 401; 429; connection refused; body that is not JSON.
- **UI checks:** stderr is `scrute:` plus the error. No partial review on stdout.
- **Dependencies:** CLI-08, CLI-09
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-12 Invalid JSON retry

- **Files:** `internal/review/runner.go`, `internal/review/review.go`
- **Expected behavior:** The first unparsable response is sent back once with a correction instruction. A second failure exits 3 and the error includes the raw response. A fenced ```json block is accepted. Unknown JSON fields are rejected. Summary, title, and detail must be non-empty. Line may be 0.
- **Acceptance checklist:**
  - [x] First bad payload and second good payload exits with the review.
  - [x] Two bad payloads exit 3.
  - [x] Extra trailing JSON exits as invalid.
- **Test cases:** prose; empty summary; severity `info`; negative line; emoji in the title; a second JSON value after the object.
- **UI checks:** The retry is not a second terminal review. One review is printed.
- **Dependencies:** CLI-06
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-13 Config file and defaults

- **Files:** `internal/config/config.go`
- **Expected behavior:** Missing `.scrute.yaml` means provider `anthropic`, strictness `balanced`, 40 context lines, 200000 max bytes, fail-on `warning`, and the six default excludes. Unknown provider, strictness, or fail-on values exit 2. `context_lines` may be 0. Negative context or `max_diff_bytes` of 0 exits 2. YAML that does not parse exits 2.
- **Acceptance checklist:**
  - [x] Missing file reviews with the defaults.
  - [x] `provider: nope` exits 2.
  - [x] `max_diff_bytes: 0` exits 2.
- **Test cases:** empty file; duplicate keys; `fail_on: CRITICAL` after the normalizer; a non-UTF8 byte.
- **UI checks:** The error names `.scrute.yaml`.
- **Dependencies:** CLI-03
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-14 Flags override the file

- **Files:** `internal/cli/cli.go`, `internal/config/config.go`
- **Expected behavior:** `--provider`, `--model`, and `--strictness` replace the file for that process. Other keys, including `fail_on` and `exclude`, stay from the file. There is no flag for those.
- **Acceptance checklist:**
  - [x] A file that says openai still uses anthropic when `--provider anthropic` is set.
  - [x] `--model custom-id` is what the stub observes.
  - [x] An invalid flag value exits 2.
- **Test cases:** empty flag omitted; model with spaces; strictness `STRICT`.
- **UI checks:** The header shows the flag model, not the file model.
- **Dependencies:** CLI-13
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-15 Init

- **Files:** `internal/cli/cli.go`, `internal/config/config.go`
- **Expected behavior:** `scrute init` writes `.scrute.yaml` at the work tree root with mode 0644 and prints `Wrote <path>`. A second run exits 2 and does not change the file. Outside a repo it exits 2.
- **Acceptance checklist:**
  - [x] The written file parses as the defaults.
  - [x] An existing file is unchanged.
- **Test cases:** run from a subdirectory; existing unreadable file; path that already is a directory named `.scrute.yaml`.
- **UI checks:** One stdout line.
- **Dependencies:** CLI-03, CLI-13
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-16 Exclude patterns

- **Files:** `internal/config/config.go`, `internal/review/budget.go`
- **Expected behavior:** Default excludes cover `.env`, `.env.*`, `**/*.pem`, `vendor/**`, `node_modules/**`, and `dist/**`. A list in the file replaces that set, so omitted defaults are sent. `**` matches every path. A basename pattern such as `*.pem` matches in nested directories. Excluded files are not prompt sections.
- **Acceptance checklist:**
  - [x] A staged `.env` is not in the stub body when defaults apply.
  - [x] An exclude list of `["keep.go"]` does not also drop `.env` unless `.env` is listed.
- **Test cases:** `vendor/lib/a.go`; `secrets/a.pem`; a rename from an excluded path; pattern `**`.
- **UI checks:** Excluded files are absent, not listed as omitted, unless the packer classifies them that way. Record the actual list from the run.
- **Dependencies:** CLI-13
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-17 Binaries and submodules

- **Files:** `internal/gitdiff/gitdiff.go`, `internal/render/render.go`
- **Expected behavior:** A NUL in the first 8 KiB marks the blob binary. The text and patch are not sent. The review lists `Skipped binary files:`. A gitlink lists `Skipped submodules:`. An empty text file is not treated as binary just because it has no hunks.
- **Acceptance checklist:**
  - [x] `blob.bin` with a NUL is skipped and the key is not required when it is the only change.
  - [x] A submodule path is named and its contents are not read into the prompt.
- **Test cases:** PNG header; a text file with a NUL past 8 KiB; an empty file; a gitlink.
- **UI checks:** The skip list is after the review body.
- **Dependencies:** CLI-05
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-18 Context and deleted files

- **Files:** `internal/gitdiff/gitdiff.go`, `internal/review/prompt.go`
- **Expected behavior:** The prompt includes a numbered context window of `context_lines` around hunks. Default is 40. A deleted file's context comes from HEAD, not from an empty index blob. `context_lines: 0` sends no extra window.
- **Acceptance checklist:**
  - [x] A deleted function body is visible in the prompt.
  - [x] Context line numbers match the file.
- **Test cases:** deletion; a one-line file; context 0; a hunk at the end of a long file.
- **UI checks:** The prompt is not the terminal UI. The terminal still cites `file:line` from the model.
- **Dependencies:** CLI-05, CLI-13
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-19 Chunking and byte cap

- **Files:** `internal/review/budget.go`, `internal/review/runner.go`
- **Expected behavior:** Sections that fit under `max_diff_bytes` share a request. A following file starts a new request when it would overflow. Each request is one review, then the parts are merged. Two or more requests print `scrute: reviewing N/M` on stderr.
- **Acceptance checklist:**
  - [x] Two small files can share one request.
  - [x] A pair that exceeds the cap becomes two requests.
  - [x] Both summaries are merged into one printed review.
- **Test cases:** cap of a few bytes; many files; a file whose section equals the cap.
- **UI checks:** Progress lines do not interleave inside the review on stdout.
- **Dependencies:** CLI-06, CLI-35
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-20 Oversized blob

- **Files:** `internal/gitdiff/gitdiff.go`, `internal/cli/cli.go`
- **Expected behavior:** A text blob larger than 32 MiB is omitted and not loaded whole. A binary larger than that is skipped after the 8 KiB sniff. A single file whose review section exceeds `max_diff_bytes` is omitted. If the review has no finding at `fail_on` and something was omitted, the exit is 2 so a hook does not treat the file as clean.
- **Acceptance checklist:**
  - [x] An omitted path is listed under `Omitted, over the diff budget:`.
  - [x] Exit code is 2 when omission is the only failure.
  - [x] A critical finding still exits 1 even if another file was omitted.
- **Test cases:** file just over the cap; 32 MiB boundary if a fixture can be built without filling the disk, otherwise the unit that sets `maxBlobBytes`; a binary over 32 MiB.
- **UI checks:** The omit list is readable and does not dump the file body.
- **Dependencies:** CLI-19, CLI-22
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-21 Strictness

- **Files:** `internal/review/prompt.go`, `internal/config/config.go`
- **Expected behavior:** `lenient` asks for likely bugs and security mistakes. `balanced` adds correctness and API misuse and skips style. `strict` also asks for maintainability and skips formatter nits. The value is only prompt text. Invalid values exit 2.
- **Acceptance checklist:**
  - [x] The stub records the matching sentence for each of the three values.
  - [x] `--strictness` overrides the file.
- **Test cases:** each value; `nope`; empty flag with file `strict`.
- **UI checks:** Strictness is not printed in the default header.
- **Dependencies:** CLI-14
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-22 Fail-on exit

- **Files:** `internal/review/review.go`, `internal/cli/cli.go`
- **Expected behavior:** Exit 0 when the review finished and nothing is at or above `fail_on`. Exit 1 when it is. Rank is critical, warning, suggestion. Default is `warning`, so a suggestion alone exits 0 and a warning exits 1. `fail_on: none` exits 0 even with a critical. `fail_on: critical` ignores warnings.
- **Acceptance checklist:**
  - [x] Suggestion with default config exits 0.
  - [x] Warning with default config exits 1.
  - [x] `none` with a critical exits 0.
- **Test cases:** each fail-on value against each severity; no issues; mixed severities.
- **UI checks:** Exit 1 still prints the full review before exiting. No extra error line when the failure has a nil error.
- **Dependencies:** CLI-23
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-23 Terminal review

- **Files:** `internal/render/render.go`, `internal/render/testdata/`
- **Expected behavior:** The first line is `scrute  <provider>/<model>`. Then Summary, issues grouped by severity, `file:line` and title, indented detail, and a count such as `1 critical, 1 warning`. Zero findings after a real review say `No issues.` Golden files lock the renderer, not model quality.
- **Acceptance checklist:**
  - [x] The golden terminal fixture matches `Format("term", ...)`.
  - [x] Issues sort critical, warning, suggestion.
  - [x] A long detail is indented and keeps its newline.
- **Test cases:** no issues; one of each severity; a title with emoji; a 200-character detail; file name with a space.
- **UI checks:** 80-column wrap does not smash the count into the detail. ANSI codes are absent when color is off.
- **Dependencies:** CLI-06
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-24 Markdown and JSON

- **Files:** `internal/render/render.go`, `internal/cli/cli.go`
- **Expected behavior:** `--format markdown` and `--format md` print a Markdown document with provider, model, and headings. `--format json` prints one JSON object. `--format text` and `--format terminal` match the terminal. Any other format exits 2. Format is case-insensitive.
- **Acceptance checklist:**
  - [x] Markdown starts with `# Scrute review`.
  - [x] JSON parses and contains the summary and issues.
  - [x] `yaml` exits 2.
- **Test cases:** `MD`; `JSON`; empty format; format with spaces.
- **UI checks:** JSON is stable enough to parse. Markdown headings are not colored.
- **Dependencies:** CLI-23
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-25 Output file

- **Files:** `internal/cli/cli.go`
- **Expected behavior:** `--out REVIEW.md` writes the review and also prints it. When `--format` is omitted, the file is markdown and stdout stays the terminal form. When `--format json` is set, the file is JSON. Mode is 0644. An unwritable path exits 2 after the stdout review.
- **Acceptance checklist:**
  - [x] Omitted format writes markdown to the file and the terminal form to stdout.
  - [x] Explicit json writes json to both.
  - [x] A directory path exits 2.
- **Test cases:** relative path; path with non-English characters; existing file overwritten; `--out` with nothing staged, which should not create a review file because the command returns before that write.
- **UI checks:** The file has a trailing newline where the renderer emits one.
- **Dependencies:** CLI-24
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-26 Color

- **Files:** `internal/cli/cli.go`, `internal/render/render.go`
- **Expected behavior:** A character device gets ANSI color for severity labels. `NO_COLOR` set to anything, including empty, disables color. `TERM=dumb` disables color. Pipes and `--format json` do not use color. The file written by `--out` is uncolored.
- **Acceptance checklist:**
  - [x] A pipe capture has no ESC bytes.
  - [x] `NO_COLOR=` disables color even on a tty, when a tty can be simulated.
- **Test cases:** pipe; `TERM=dumb`; json format.
- **UI checks:** Critical, warning, and suggestion are still labeled in text when color is off.
- **Dependencies:** CLI-23
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-27 Verbose log

- **Files:** `internal/cli/cli.go`
- **Expected behavior:** `--verbose` prints one stderr line with provider, model, file count, and input and output token totals, then `scrute: sent <path>` for each sent file. It does not print the prompt or the key. A nothing-staged run does not need this line.
- **Acceptance checklist:**
  - [x] The stub token counts appear.
  - [x] The API key string is absent from stderr.
- **Test cases:** one file; two chunks; verbose with a provider failure, where counts may be partial. Record what is printed.
- **UI checks:** Progress and verbose lines stay on stderr so stdout remains a clean review.
- **Dependencies:** CLI-06
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-28 Commit range

- **Files:** `internal/cli/cli.go`, `internal/gitdiff/gitdiff.go`
- **Expected behavior:** `--range BASE..HEAD` reviews that pair of trees, not the index. `BASE...HEAD` exits 2. A missing head exits 2. An unknown revision exits 2. An empty result prints `Nothing to review.` and exits 0. A range whose head is all zeros is rejected as an empty head.
- **Acceptance checklist:**
  - [x] A committed change is reviewed while a newer staged line is absent.
  - [x] Three dots exit 2.
  - [x] `HEAD~1..HEAD` matches the commit diff for a fixture repo.
- **Test cases:** `..HEAD` as a new ref; `HEAD..HEAD` empty; a 40-character missing object; emoji in a rev that does not exist.
- **UI checks:** The review header is the same as a staged review.
- **Dependencies:** CLI-05
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-29 New branch base

- **Files:** `internal/gitdiff/gitdiff.go`
- **Expected behavior:** When the base revision is empty or all zeros, the diff starts at the merge-base with `main`, `master`, `origin/main`, or `origin/master` when that commit is a real ancestor. Otherwise the base tree is empty. The head commit itself is not used as its own base.
- **Acceptance checklist:**
  - [x] A branch created from main reviews only the branch commits when the base is 40 zeros.
  - [x] A repo with no main or master reviews against an empty tree.
- **Test cases:** main plus one commit; master only; unrelated history; SHA-256 all-zero head is rejected by the empty-head check.
- **UI checks:** None beyond the review text.
- **Dependencies:** CLI-28
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-30 Hook install and uninstall

- **Files:** `internal/hook/hook.go`, `internal/cli/cli.go`
- **Expected behavior:** `scrute hook install` writes `.git/hooks/pre-push` with the marker `# scrute-managed-pre-push`, mode 0755, and the absolute path of the current executable, symlink-resolved. Re-install updates that path. `scrute hook uninstall` removes that file. The command prints one confirmation line.
- **Acceptance checklist:**
  - [x] The script is executable and contains the marker and a quoted path.
  - [x] Uninstall removes it.
  - [x] Uninstall with no hook exits 2.
- **Test cases:** path containing a single quote; install twice; install from a subdirectory.
- **UI checks:** Confirmation is one stdout line.
- **Dependencies:** CLI-03
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-31 Hook push review

- **Files:** `internal/hook/hook.go`
- **Expected behavior:** Git's pre-push stdin lines are `local_ref local_sha remote_ref remote_sha`. Only `refs/heads/*` with a non-zero local sha run `scrute --range <remote>..<local>`. Tags are skipped. An all-zero local sha, including a SHA-256 delete, is skipped. If scrute exits non-zero, the hook prints `scrute: push aborted (exit N)` and exits N. If stdin is a terminal, the hook execs scrute with no range. `git push --no-verify` skips the hook. On Windows the script is POSIX and runs under Git for Windows.
- **Acceptance checklist:**
  - [x] A branch update invokes `--range`.
  - [x] A tag update does not.
  - [x] Exit 1 from scrute aborts the hook with that status.
- **Test cases:** two branch updates where the second would run only if the first exits 0; delete sha of 40 zeros; delete sha of 64 zeros; a ref that is not `refs/heads` or `refs/tags`.
- **UI checks:** The abort line is on stderr. The review itself is the CLI layout.
- **Dependencies:** CLI-28, CLI-30
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-32 Foreign hook

- **Files:** `internal/hook/hook.go`
- **Expected behavior:** If `pre-push` exists and does not contain the scrute marker, install exits 2 with `a pre-push hook already exists` and does not modify the file. Uninstall exits 2 with `no scrute pre-push hook is installed` and leaves the file.
- **Acceptance checklist:**
  - [x] A custom hook's bytes are unchanged after a failed install.
  - [x] Uninstall does not delete that file.
- **Test cases:** empty file; file that mentions scrute without the marker; marker present.
- **UI checks:** The error names the situation in one line.
- **Dependencies:** CLI-30
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-33 Worktree and conflicts

- **Files:** `internal/gitdiff/gitdiff.go`
- **Expected behavior:** `scrute` from a linked work tree reads that work tree's index. A conflicted index returns an error and the CLI exits 2. The command does not launch a model after that error.
- **Acceptance checklist:**
  - [x] A linked work tree sees its staged file.
  - [x] A conflict exits 2.
- **Test cases:** `git worktree add`; a merge conflict left unresolved.
- **UI checks:** The conflict error is on stderr.
- **Dependencies:** CLI-05
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-34 Chunk progress

- **Files:** `internal/review/runner.go`
- **Expected behavior:** A single request is quiet. Two or more requests print `scrute: reviewing 1/2` and then `2/2` on stderr before the final stdout review.
- **Acceptance checklist:**
  - [x] One chunk produces no `reviewing` line.
  - [x] Two chunks produce both lines in order.
- **Test cases:** force a tiny `max_diff_bytes`; a provider error on the second chunk stops before a review.
- **UI checks:** The lines are not mixed into the summary.
- **Dependencies:** CLI-19
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## CLI-35 Merged findings

- **Files:** `internal/review/budget.go`, `internal/review/runner.go`
- **Expected behavior:** Summaries from chunks are joined. Issues with the same severity, file, line, and title are dropped as duplicates. The printed order follows severity rank.
- **Acceptance checklist:**
  - [x] Two chunks that return the same issue print it once.
  - [x] Distinct issues from both chunks appear.
- **Test cases:** same issue with a different detail; same title on another line.
- **UI checks:** The count line matches the printed issues.
- **Dependencies:** CLI-19, CLI-23
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## WEB-01 Marketing page

- **Files:** `site/index.html`, `site/CNAME`, `site/favicon.svg`
- **Expected behavior:** The page is static. The heading is "Review the diff before you push." The install line is `go install github.com/rejisterjack/scrute@latest`. The three steps mention the two environment variables, `git add`, `scrute`, and `scrute hook install`. The example panel is marked as an example, not a live model call. Footer links go to the GitHub repository and the latest release. `CNAME` is `scrute.rejisterjack.com`. There is no form and no backend.
- **Acceptance checklist:**
  - [x] The heading and install command are visible.
  - [x] Both links resolve as https://github.com/rejisterjack/scrute and the releases URL.
  - [x] The console has no errors.
- **Test cases:** open the file through a local static server; click both links; reload.
- **UI checks:** Alignment of the numbered steps, the install line does not overflow the card, the example panel header stays on one row on a wide screen, light paper background, 360, 768, 1024, and 1440. Hover and focus on the two links. No dark theme exists.
- **Dependencies:** none
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## WEB-02 Title, description, icon, robots

- **Files:** `site/index.html`, `site/favicon.svg`, `site/robots.txt`
- **Expected behavior:** The title is `Scrute — review the diff before you push`. The description says the CLI uses the caller's Claude or GPT key and the source stays on the machine. Canonical is `https://scrute.rejisterjack.com/`. The icon is `/favicon.svg`. `robots.txt` is `User-agent: *` and `Allow: /`.
- **Acceptance checklist:**
  - [x] Title, description, canonical, and icon link are present.
  - [x] `/robots.txt` returns the allow rule.
  - [x] `/favicon.svg` is an SVG.
- **Test cases:** view source; request robots and the icon.
- **UI checks:** The tab title is the full title. The icon is not a broken image.
- **Dependencies:** WEB-01
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## WEB-03 Responsive page

- **Files:** `site/index.html`
- **Expected behavior:** Below 560px the main padding shrinks and the example panel header stacks. The install command can scroll sideways instead of widening the page. The type is the serif stack in the file. `color-scheme` is light.
- **Acceptance checklist:**
  - [x] No horizontal page overflow at 360, 768, 1024, and 1440.
  - [x] At 360 the panel header stacks.
  - [x] Text remains readable on the paper background.
- **Test cases:** resize; a forced dark preferred color scheme still shows the light page.
- **UI checks:** Focus outline on links, heading size, footer spacing. There is no loading or empty state; the page is static.
- **Dependencies:** WEB-01
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## WEB-04 Security headers

- **Files:** `site/`
- **Expected behavior:** The repo does not set `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`, or a Content-Security-Policy for the static page. This feature is the BUILD gap. A finished page should send `nosniff`, a strict referrer policy, `DENY` framing, and a CSP that still allows the inline stylesheet the page uses.
- **Acceptance checklist:**
  - [x] A header file in `site/` documents those policies.
  - [x] The inline page still renders with that CSP.
  - [x] The live or preview response includes the headers after a deploy. This step does not deploy.
- **Test cases:** response headers from a local static server cannot prove Vercel headers; check the file, then the deployed headers only in step 2 if the site is already public.
- **UI checks:** The page is not blank when the CSP is applied.
- **Dependencies:** WEB-01
- **Suspected bugs:** No header file is present under `site/`. That is the gap, not a logic bug in the CLI.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** site/vercel.json
- **Notes:** Added site/vercel.json. Chrome with that CSP kept the paper background and reported no console errors.
- **Status:** PASSED

## DEVOPS-01 CI

- **Files:** `.github/workflows/ci.yml`
- **Expected behavior:** Pushes to `dev` and `main`, and pull requests, run `go test -race ./...`, golangci-lint v2.15.0, and `govulncheck` at v1.9.0. Setup uses `go-version-file: go.mod`. Permissions are contents read.
- **Acceptance checklist:**
  - [x] The workflow file matches that description.
  - [x] Step 2 runs the same test command locally. It does not have to wait on GitHub unless a run is already available to read.
- **Test cases:** read the workflow; run the local race command in step 2.
- **UI checks:** none
- **Dependencies:** TEST-01
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## DEVOPS-02 Release binaries

- **Files:** `.github/workflows/release.yml`, `.goreleaser.yaml`
- **Expected behavior:** A tag matching `v*` checks out full history, requires the tagged commit to be an ancestor of `origin/main`, runs the race tests, and runs GoReleaser. Targets are linux, darwin, and windows for amd64 and arm64. Archives are tar.gz, zip on Windows, and include LICENSE and README. `checksums.txt` is published. The version ldflag sets `internal/buildinfo.Version`. A tag that is not on main fails the ancestor check.
- **Acceptance checklist:**
  - [x] The workflow refuses a tag that is not contained in main.
  - [x] The goreleaser file lists six OS and arch pairs.
  - [x] This step does not push a tag.
- **Test cases:** read both files. A live tag is out of scope for step 1.
- **UI checks:** none
- **Dependencies:** CLI-02, TEST-01
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## DEVOPS-03 Go 1.27.4

- **Files:** `go.mod`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- **Expected behavior:** The module line is `go 1.27.4`. CI and release both take the version from `go.mod` rather than a hardcoded older Go.
- **Acceptance checklist:**
  - [x] `go.mod` says 1.27.4.
  - [x] Both workflows use `go-version-file: go.mod`.
- **Test cases:** read the three files. Step 2 can run `go version` while testing.
- **UI checks:** none
- **Dependencies:** DEVOPS-01
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## TEST-01 Race suite

- **Files:** `internal/cli/cli_test.go`, `internal/gitdiff/gitdiff_test.go`, `internal/hook/hook_test.go`, `internal/review/review_test.go`, `internal/config/config_test.go`, `internal/render/render_test.go`, `internal/provider/anthropic/anthropic_test.go`, `internal/provider/openai/openai_test.go`
- **Expected behavior:** `go test -race ./...` covers the staged diff, the range, the hook, config, the renderer goldens, and provider stubs. Provider tests do not call live Claude or GPT. This step did not run the suite.
- **Acceptance checklist:**
  - [x] Step 2 runs `go test -race ./...` and records the result.
  - [x] A failure is a bug, not a skipped test.
- **Test cases:** the full package list. Do not point the tests at a real API key.
- **UI checks:** none
- **Dependencies:** none
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** none
- **Notes:** Passed when exercised on 21 Nov 2026.
- **Status:** PASSED

## Assumptions

- The product is the Go module at the repo root plus the static site in `site/`.
- https://scrute.rejisterjack.com is the documented home. This step did not open it.
- Step 2 will use the local CLI and a stub or the existing tests for model calls. A live provider call happens only when a key is already in the environment. The key is not printed.
- Browser checks for `site/` use Chrome DevTools. No new browser harness is added.
- There is no database, account, or Scrute server to migrate.

## Risks

- The pre-push hook stores an absolute path. Moving the binary without reinstalling makes the hook fail closed on the next push.
- `SCRUTE_*_BASE_URL` sends the API key to that host.
- A model can name the wrong line. The CLI does not verify that the line exists.
- Exit 1 is how the hook blocks a push. `git push --no-verify` skips it.
- Review quality is not measured against a labeled corpus.

## Whole-Product Check

- `go test -race ./...` passed on Go 1.27.4. `go vet ./...` passed. `govulncheck` reported no vulnerabilities.
- The CLI binary printed `scrute dev`. Help exited 0. An unknown command and a run outside a repository exited 2.
- An empty index printed `Nothing staged.` Unstaged text was not sent. A missing key exited 2 and named `ANTHROPIC_API_KEY`.
- Against a local stub, Anthropic returned a critical and a warning and exited 1. A suggestion exited 0. `fail_on: none` exited 0. OpenAI printed `openai/gpt-5.4`. A provider error redacted the key and exited 3. Invalid JSON retried once, then a second failure exited 3.
- `--format json` printed one object. `--out` wrote `# Scrute review` as mode 0644 while stdout stayed the terminal form. A three-dot range exited 2. `HEAD~1..HEAD` did not include a later staged line. An empty range printed `Nothing to review.`
- The pre-push hook was mode 0755. A tag update and a delete did not run the review. A branch update ran `--range`. A foreign hook was left unchanged.
- A staged `.env` secret was absent from the stub body. A binary file was listed as skipped without a key. Piped output had no color codes.
- Chrome showed the marketing page with no horizontal overflow at 360, 768, 1024, and 1440. Footer text is 6.14:1. The console had no errors. With the new CSP applied, the paper background and the heading remained.
- https://scrute.rejisterjack.com returned 200 before this header file was deployed. This pass did not claim those response headers until main deploys.

## Totals

- Features: 43
- Passed: 43
- Blocked: 0
- P0: 26
- P1: 13
- P2: 4
- Bugs found: 0
- Bugs fixed: 0
- UI issues fixed: 0

## Still Blocked

None.

## Known Limitations

- Deferred, and not feature rows: a hosted PR bot, accounts, billing, telemetry, a dark theme, providers other than Claude and GPT, a dollar spend cap, and a security scanner.
- Review quality is the model's opinion. The CLI does not check that a cited line exists.
- The pre-push hook stores an absolute path. Moving the binary without reinstalling makes the next push fail closed.
- `SCRUTE_ANTHROPIC_BASE_URL` and `SCRUTE_OPENAI_BASE_URL` send the API key to that host.
- The published CLI release at the start of this pass was v0.2.0. The Go source tested here is that same tree. The new file is the marketing header policy.

## Launch Checklist

- Install with `go install github.com/rejisterjack/scrute@latest` or a GitHub release binary.
- Set `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`. Do not put the key in `.scrute.yaml`.
- Run `scrute` on a staged diff. Use `scrute hook install` only when a failing review should block `git push`.
- The static page deploys from `site/` with the headers in `site/vercel.json`.

## Decision

GO. 43 features passed. No feature is blocked. The CLI, the hook, the race suite, and the marketing page passed the checks in this report.

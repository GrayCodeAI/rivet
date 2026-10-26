# Rivet

Rivet combines two agent tool ideas in one local CLI:

- **Reusable blocks:** Keep a script and its check in a project's `.rivet/blocks/` directory. Verify the block before running it again. A change to its manifest, script, or check invalidates the local receipt.
- **Compact command output:** Run ordinary commands or blocks through Rivet. It prints a bounded summary and saves the complete output in `.rivet/logs/`. A command's exit code is preserved.

The design was informed by [Caveman Blocks](https://github.com/caveman-ai/blocks) and [RTK](https://github.com/rtk-ai/rtk). Rivet is independent Go code, with no external Go dependencies. It supports POSIX shell blocks on Linux and macOS.

## Try it

```sh
go build -o rivet ./cmd/rivet
./rivet init --project /path/to/project
./rivet add --project /path/to/project \
  --description 'Count lines in a file' \
  --script examples/line-count/run.sh \
  --check examples/line-count/verify.sh line-count
./rivet verify --project /path/to/project line-count
./rivet run --project /path/to/project line-count /path/to/file
./rivet exec --project /path/to/project -- go test ./...
./rivet list --project /path/to/project
```

`add` copies the script and check. The check receives `RIVET_SCRIPT` and `RIVET_PROJECT_ROOT` and must exit zero. `run` passes arguments as literal argument values and runs from the project root. `exec` also runs from the project root. Both write combined stdout and stderr to a private local log and return its path; `--max-lines` controls the visible line count. Use `--log-dir` to keep logs outside a clean worktree. `--json` returns a structured result for Reflect and other tools. Long lines are truncated in the summary, but retained in the log. A successful `go test` run with long output gets a package count; other long output keeps its first and last lines. `list` and `show` return JSON for other GrayCode tools.

Commit `.rivet/blocks/` and `.rivet/.gitignore` with a project. Verification receipts and logs stay local. Reverify after changes or when external dependencies need a fresh check.

Rivet does not auto-rewrite shell commands or install hooks. It only runs when called explicitly. Block verification is a quality gate, not a sandbox or security guarantee. Review scripts before importing or running them.

## GrayCode boundary

Reflect's `work` flow can call Rivet for verification and compact results with `--rivet-bin`, or verify and run a committed block with `--verify-block`. Ridge can display Rivet's JSON block status and raw logs. Relay continues to own model routing and token accounting. Ridge integration is not wired yet.

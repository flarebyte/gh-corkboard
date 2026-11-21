# Contributing

## Local setup

Use Go 1.26 or later, Git, and GitHub CLI. For the standard checks, install the [gh-flarebyte extension](https://github.com/flarebyte/gh-flarebyte) with `lint`, `test`, `cc`, and `build` support, and make `jscpd` available on your PATH for duplicate-code detection.

From the repository root, use the [source installation steps](README.md#install-from-source) to run Corkboard through GitHub CLI. For a quick check of the entry point, run `go run ./cmd/gh-corkboard --help`. Exercise commands that write board data in a disposable Git worktree so your own board stays available.

## Verification

Run the checks from the repository root.

```sh
gh flarebyte config validate
gh flarebyte lint
gh flarebyte test
gh flarebyte cc
gh flarebyte build
```

Use `gh flarebyte format` to apply the configured formatter. Build targets and checksum paths are in [`.gh-flarebyte.cue`](.gh-flarebyte.cue). The standard build packages macOS arm64, Linux amd64, and Windows amd64 artifacts and writes their checksums to `build/checksums.txt`.

For a focused iteration, run the package that owns the behavior.

```sh
go test ./internal/cli -run TestPatchDocumentFormats
go test ./internal/board ./internal/docformat ./internal/schema
```

Tests use temporary Git repositories and local fixtures. Add CLI cases for public flags, output streams, and exit codes. Put parser, persistence, and filesystem cases near the packages they exercise. Use the [implementation overview](TECHNICAL_DESIGN.md) to locate those boundaries.

## Updating docs and examples

Update the [README](README.md) when setup or user workflows change. Keep the [design sources](doc/design-meta/AGENTS.md) aligned with public command and output changes. Edit the inputs under `doc/design-meta/` and regenerate the Markdown under `doc/design/` with `flyb`.

```sh
flyb validate --config doc/design-meta
flyb generate markdown --config doc/design-meta
```

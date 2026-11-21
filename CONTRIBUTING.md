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

## Refreshing Go metadata

The two root-level Go pipelines are managed by [the Arrow sync manifest](sync.gh-arrow.cue). Use Arrow 0.3.0 or newer and run these commands from the repository root.

```sh
gh arrow sync diff
gh arrow sync
gh arrow sync check
gh flarebyte meta
```

Arrow uses the configured GitHub mapping for `flarebyte:baldrick-reserve` when one exists. Point that mapping at a Reserve checkout containing `shared/go/all/` using the [Arrow configuration format](https://github.com/flarebyte/gh-arrow/blob/main/doc/design-meta/examples/config.cue). Arrow otherwise reads the repository's default branch. Review the diff before syncing because sync replaces changed pipeline files.

Metadata generation requires `thoth`, `maat`, Node.js 24 or later, and `npx`. The shared pipelines invoke `npx maat-ostraca`, which may need npm registry access. YAML goes under `thoth-meta/` and debug JSON under `temp/`. Since the pipelines keep going after analysis errors, inspect the debug reports when checking a run.

## GitHub Actions

The [build workflow](.github/workflows/build.yml) runs Go vet, tests, and a configured Linux amd64 build using the standalone Flarebyte builder. Its job reports the `CI passed` check.

To publish a release, set a new stable version in [main.project.yaml](main.project.yaml), then run **Manual release** in GitHub Actions on the default branch. The [release workflow](.github/workflows/release.yml) tests the selected commit, builds the configured extension binaries and archives with Flarebyte, and verifies their SHA-256 manifest before uploading. The executable embeds the version from `main.project.yaml`, which the release workflow reads to name the tag. It creates the `v`-prefixed tag at the tested commit and publishes the release with generated notes. If publishing is interrupted, check GitHub Releases for the draft and verify the tag before retrying.

# gh-corkboard

Corkboard is a GitHub CLI extension for keeping essential notes in a Git worktree. Humans and agents sharing a checkout can publish handoffs, record progress and reviews, and find related Markdown, JSON, and CSV documents by milestone or concept. Each document has a stable UUID you can use in a handoff or review.

## Install with GitHub CLI

With GitHub CLI (`gh`) installed, run these commands from any directory. GitHub CLI can install Corkboard without a source checkout or Go when the latest published release includes a binary for your platform. The release workflow builds binaries for macOS arm64, Linux amd64, and Windows amd64. See the [release assets](https://github.com/flarebyte/gh-corkboard/releases).

```sh
gh extension install flarebyte/gh-corkboard
gh corkboard --version
```

If GitHub CLI asks you to authenticate, run `gh auth login` and retry. If a release binary is unavailable for your platform, use the source installation below.

## Install from source

You need Git, GitHub CLI (`gh`), and Go 1.26.11 or later. From the root of the Corkboard source checkout named `gh-corkboard`, build the executable and register it as a local extension. Go names the executable `gh-corkboard.exe` on Windows.

```sh
go build ./cmd/gh-corkboard
gh extension install .
gh corkboard --help
```

GitHub CLI uses the executable in that checkout. Rebuild it there after changing versions. See the [GitHub CLI extension install reference](https://cli.github.com/manual/gh_extension_install).

## Quick start

In a fresh Git worktree, initialize a board with `human:owner` as its administrator. Create a topic, save a Markdown note, and read it back.

```sh
gh corkboard --as human:owner init
gh corkboard --as human:owner access topic create specs
printf '%s\n' '# Pagination' 'Return a cursor when more results are available.' |
  gh corkboard --as human:owner doc create --in topic:specs --name pagination \
    --format md --milestone v1/Specs --concept Cursor/Pagination

doc_id=$(gh corkboard --as human:owner doc locate --in topic:specs \
  --name pagination --milestone v1/Specs)
gh corkboard --as human:owner doc get "$doc_id"
```

The final command prints the stored Markdown. To find related documents, search by milestone and concept.

```sh
gh corkboard --as human:owner doc search --milestone 'v1/*' \
  --concept 'Cursor/*' --format table
```

## Working with a board

- Board data lives in `.gh-corkboard/` at the worktree root. `init` adds a root ignore rule. Each checkout has its own board, and board commands work offline without GitHub authentication. The board filesystem must support hard links for atomic document and schema creation.
- Use `access` to register principals, create topics, and grant permissions. Agent and skill principals can write to their own channels. `--as` selects a registered principal for local permission checks. Anyone with unrestricted access to the same OS account can select another principal or edit the files directly.
- Use `doc get --with-revision` before a guarded `doc update` or `doc patch`. Use `schema` to register validation rules and `doc validate` to check drafts. Well-formed content can be saved while it still has schema findings.
- Administrators can inspect drift with `index audit`, reconcile it with `index resync`, and preview archived documents with `index prune`. Save a `snapshot create` ZIP outside the board before applying a reviewed prune plan. Snapshots support offline inspection.

Run `gh corkboard help` or `gh corkboard help doc update` for command syntax. Commands that support `--json` return structured results. Plain `doc get` and `schema get` write stored bytes. `snapshot create` writes a binary ZIP. Exit status `1` means a finding or malformed input, and `2` means a usage, permission, or operational error.

## Documentation

- [Command reference](doc/design/corkboard.md#command-interface) and [workflow examples](doc/design/corkboard.md#use-cases)
- [CLI output examples](doc/design/command-outputs.md)
- [Contributor setup and checks](CONTRIBUTING.md)
- [Implementation overview](TECHNICAL_DESIGN.md)
- [License](LICENSE)

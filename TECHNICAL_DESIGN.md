# Implementation overview

Corkboard is a Go GitHub CLI extension with a local filesystem backend. A board belongs to one Git worktree. See the [v1 specification](doc/design/corkboard.md) for command contracts and the [validation decision](doc/decision/validation.md) for the schema language choice.

## Code map

| Location | Responsibility |
| --- | --- |
| [`cmd/gh-corkboard`](cmd/gh-corkboard) | Process entry point, standard streams, and exit status |
| [`internal/cli`](internal/cli) | Cobra commands, flags, text and JSON output, diagnostics, and workflow tests |
| [`internal/board`](internal/board) | Worktree discovery, locks, access policy, catalog, filesystem operations, and retention |
| [`internal/docformat`](internal/docformat) | Markdown, JSON, and CSV parsing and formatting, patching, and validation projections |
| [`internal/schema`](internal/schema) and [`internal/strictjson`](internal/strictjson) | Offline Draft 2020-12 compilation, validation diagnostics, and strict JSON decoding |
| [`internal/testrepo`](internal/testrepo) | Disposable Git test fixtures |
| [`doc/design-meta`](doc/design-meta) | Source inputs for the v1 specification and CLI examples |

## Command and storage flow

For a document mutation, the CLI gathers flags and passes the operation to the board. The board resolves the worktree, opens confined directory handles, and takes the board lock. It validates the stored policy and catalog, then checks the caller's permissions and any expected revision before changing data. Document formatting and patching operate on bytes before publication. The CLI renders the result after the operation.

The access policy and catalog are versioned JSON. Both are checked against [embedded schemas](internal/board/spec) and Go invariants. User schemas live in the board's schema registry and apply only when validating document content. All schema reference resolution is local.

The catalog maps UUIDs to derived paths and stores digests and metadata. The logical name key combines scope, case-folded milestone, and name. Updates keep both UUID and filename stable. A guarded write compares against the digest of the current file bytes.

## Consistency and recovery

- Reads use a shared board lock and mutations use an exclusive lock. Authorization and board I/O happen under the same lock. The local `--as` selector relies on cooperating callers.
- Catalog and file replacement are atomic at the individual-file level. Document bytes are written before the matching catalog change. An interrupted operation can leave drift that `index audit` reports and `index resync` reconciles.
- Resync keeps UUIDs found in canonical filenames. On index rebuild, metadata stored only in the catalog may be lost. Conflicting UUIDs or logical names stop reconciliation before writes.
- Prune preview, guarded snapshots, and prune apply share the same candidate calculation. Apply rechecks each candidate before unlinking. Snapshots archive stored files for offline inspection, including files with catalog drift.

Add command syntax and rendering in `internal/cli`. Reuse the board's authorization, document selection, and publication paths for operations that access stored data. Keep pure transformations in `internal/docformat` so create, update, patch, and validate can share parsers. See [CONTRIBUTING.md](CONTRIBUTING.md#verification) for the verification commands.

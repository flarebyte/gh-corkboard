package flyb

_proseNotes: [
	{
		name:  "board.board"
		title: "Workspace and handoffs"
		markdown: #"""
			The Go binary is `gh-corkboard`, invoked as `gh corkboard`. The CLI uses the current Git worktree root, including when invoked from a subdirectory or from a linked worktree. Commands fail outside a Git worktree. Each board belongs to one checkout and works offline without GitHub authentication. Handoffs require agents to share that board. Folder names route information and carry no access-control semantics. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)

			```text
			.gh-corkboard/
			├── index/
			│   └── documents.json
			├── schemas/
			│   ├── csv/
			│   ├── json/
			│   │   └── spec.schema.json
			│   └── md/
			│       └── features.schema.json
			├── agents/
			│   ├── implementation/
			│   │   └── pagination-handoff.md
			│   └── specs/
			│       └── follow-up.md
			├── skills/
			│   └── review/
			└── topics/
			    └── specs/
			        ├── pagination.spec.json
			        └── pagination.features.md
			```

			`init` creates the five top-level directories, the three schema format directories and an empty version-1 index on first initialization. Repeated runs preserve existing files, including a damaged index. It adds `/.gh-corkboard/` as the final root `.gitignore` rule if needed, verifies the rule is effective, and reports tracked corkboard paths for the user to resolve. It never changes Git tracking.

			Document paths are relative to `.gh-corkboard/` and use forward slashes. They start with `agents/<agent-id>/`, `skills/<skill-id>/` or `topics/<topic-id>/` and may have additional directories. Directory segments, document names and schema IDs use lowercase ASCII letters, digits, hyphens and underscores, starting with a letter or digit. Reject reserved platform names, absolute paths, traversal, symlinks, and non-regular files. Resolve filesystem operations relative to the managed directories so path checks remain effective during IO.

			The basename is `<name>.<format>` or `<name>.<schema-id>.<format>`. The two spellings are unambiguous because dots are reserved. A named schema always resolves by both format and ID, such as `schemas/md/features.schema.json` for `pagination.features.md`. A missing named schema is a validation error. Agent inboxes use the same document commands as other folders. A handoff can be a short Markdown file under the recipient's folder that names the sender, requested action and board-relative path of a shared document. The recipient can set it to `resolved` after acting. A self-note goes under the writer's own agent ID.
			"""#
	},
	{
		name:  "board.interface"
		title: "Command conventions"
		markdown: #"""
			The command table uses `-` for standard input and paths relative to the board unless specified otherwise. Snapshot destinations and `--from` files are resolved relative to the caller's working directory. Create refuses to replace an existing resource and update/delete require an existing target. Updates change only the fields supplied. `doc validate --all` discovers documents on disk, including unindexed files.

			Use `--json` on commands that return metadata, mutation summaries, reports or diagnostics. Findings and result lists use lexical path order. The two `get` commands emit the original bytes. Send usage and operational errors to stderr and keep machine-readable reports on stdout. Exit 0 for success including a clean audit and successful validation of schemaless documents. Exit 1 for validation failures or audit findings. Exit 2 for invalid invocation or operational failure, including failed cleanup preconditions and partial writes.

			```sh
			gh corkboard init
			gh corkboard doc create topics/specs/pagination.md --from pagination.md --description "Pagination requirements" --status active
			gh corkboard doc create agents/implementation/pagination-handoff.md --from handoff.md --description "Implement pagination from topics/specs/pagination.md" --status active
			gh corkboard index search --prefix agents/implementation --status active --json
			gh corkboard doc get agents/implementation/pagination-handoff.md
			gh corkboard doc update agents/implementation/pagination-handoff.md --status resolved
			gh corkboard index search --status review --json
			gh corkboard index audit --json
			gh corkboard index prune --json
			# Substitute the plan ID from the preview.
			gh corkboard index prune --apply --expect "$plan_id" --snapshot ../corkboard-before-prune.zip
			```

			The implementation should exercise initialization from a subdirectory and linked worktree, agent handoffs, invalid draft writes, schema changes and reference checks, CSV and Markdown projection boundaries, external file edits, concurrent CLI updates, stale prune plans, and ZIP failure before deletion. The packaged binary follows the GitHub CLI naming conventions for precompiled extensions. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
			"""#
	},
	{
		name:  "board.validation"
		title: "Validation rules"
		markdown: #"""
			Use JSON Schema Draft 2020-12 directly. A registry file is `schemas/<format>/<id>.schema.json`. Schema writes compile the definition and reject unsupported dialects and unresolved references. V1 accepts same-resource fragment references and uses bundled Draft 2020-12 meta-schema resources. Fetching schema references from the filesystem or network is disabled. Apply standard annotation semantics for `format`. A schema update changes subsequent validation for documents bound to that ID. Authors can register a new ID such as `spec-v2` when both contracts must coexist. Schema deletion checks bound filenames on disk, including files absent from the index. [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)

			| Format | Validation input |
			| --- | --- |
			| JSON | Parse exactly one UTF-8 JSON value and reject duplicate object keys. Preserve numeric precision through validation. |
			| CSV | Parse UTF-8 comma-separated records with standard quoting and a mandatory header. Use the header verbatim as ordered `columns` and each data record as an object in `rows`. Header names must be nonempty and unique. Reject records whose field count differs from the header. Preserve data cell strings, including empty strings and whitespace. Skip physically empty lines outside quoted fields. |
			| Markdown | Parse UTF-8 as CommonMark 0.31.2. Collect document-level headings in source order into `headings`, with parsed level and visible text from inline nodes. Strip markup and link destinations, include image alt text and inline code, turn inline breaks into spaces, trim edges and collapse whitespace runs. Headings inside containers such as quotes and lists stay outside this projection. |

			The CSV projection supports required columns, header order, row count and string constraints. For example, a source file with the header `id,priority` and the row `pagination,01` projects to `{"columns":["id","priority"],"rows":[{"id":"pagination","priority":"01"}]}`. Typed numeric, date and relational constraints would need a future profile. CSV on the Web provides a standardized metadata vocabulary if that interoperability becomes a requirement. [W3C CSV on the Web](https://www.w3.org/TR/tabular-metadata/)

			The Markdown projection supports required headings, order and multiplicity. For example, `## Acceptance **criteria**` yields the entry `{"level":2,"text":"Acceptance criteria"}`. A schema can check for that entry with `contains` or enforce a sequence with `prefixItems`. Markdown prose and section body content remain outside the v1 schema contract. [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/)

			Every document remains writable even when validation would fail. On explicit validation, parse the native format first, report syntax or projection errors, then evaluate the declared schema when present. A well-formed plain document returns `schemaless`. Bound documents return `valid` or `invalid`. Report the instance path and source coordinates where available. CommonMark accepts free-form text. Schemaless Markdown therefore only requires decodable UTF-8. The index uses a built-in schema with additional Go checks for board-relative path validity and unique paths. User schema operations cannot alter the index contract.

			CUE supports composition and constraints across fields, and exposes validation and JSON Schema conversion through its Go API. Both engines would consume the same CSV and Markdown projections. Keeping one native JSON Schema engine gives authors reusable JSON artifacts for documents and the index. Revisit CUE if real board schemas need shared constraints or cross-field relationships that become hard to express in JSON Schema. [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)
			"""#
	},
	{
		name:  "board.index"
		title: "Catalog and reconciliation"
		markdown: #"""
			`index/documents.json` is one catalog for the whole board. Its top-level object has `version: 1` and a `documents` array ordered lexically by canonical path. Each entry has a unique board-relative `path`, a human-authored `description` which may be empty, a `status`, and a lowercase hexadecimal SHA-256 digest of the indexed bytes. The built-in version-1 JSON Schema requires these fields, constrains the status enum and digest, and disallows unknown properties. Paths and uniqueness are checked in Go.

			`index search` combines supplied filters with AND. The prefix matches a full path or a directory boundary. The query is a case-insensitive substring of the path or description. Search returns indexed data, with `index audit` available to check its freshness. `doc get` reads the file on disk. Use `doc update` to edit metadata without supplying content. New entries default to an empty description and `draft`. Any status can be set explicitly. When CLI or external content changes, a prior `obsolete` status reverts to `draft` unless the CLI write explicitly sets a status for the new content.

			`index audit` reports missing files, unindexed documents, digest changes, malformed or missing index, and unsupported entries under the document roots. Drift and an unusable catalog are distinguishable diagnostics. `index resync` adds discovered managed files as drafts, preserves surviving descriptions and status except for changed obsolete files, updates digests and drops records for missing files. Report every dropped record and reset. External renames appear as a missing path and a new file. Build an empty or missing catalog from files on disk. Refuse to overwrite a malformed or unsupported index so its metadata remains recoverable. Audit and resync operate on file presence and bytes. Use `doc validate --all` to check document and schema rules.

			All CLI operations that mutate the board take a board-level exclusive lock. Use an OS-backed lock released on process exit, with lock bookkeeping under `index/`. Reads spanning multiple files use a compatible lock. Write replacements atomically at the individual-file level. Write document bytes before the corresponding index entry, and report partial success if the index write fails. Audit and resync recover file/catalog consistency after interrupted updates. Use a reliable local filesystem with working file locks. Direct editors can create drift and must finish writes before a snapshot or destructive command.
			"""#
	},
	{
		name:  "board.retention"
		title: "Prune and archive"
		markdown: #"""
			`index prune` previews every explicitly obsolete document whose recorded digest still matches the file on disk. It reports skipped obsolete entries with missing or changed files and emits a deterministic plan ID derived from the sorted candidate paths and digests. Apply requires `--apply --expect <plan-id>`. Recompute the plan under the board lock and fail before any deletion if it differs or the catalog is unusable. Recheck each candidate immediately before unlinking. If any unlink fails, stop, reconcile entries for files successfully removed, and report completed and remaining paths. If reconciliation fails, report that too so `index audit` can identify the drift. Resolved, invalid, schemaless and unindexed documents have no implicit deletion eligibility.

			`index prune --apply --expect <plan-id> --snapshot <zip>` first writes the complete pre-deletion board under the same lock. A failed archive aborts the prune. `snapshot create --output <zip>` archives independently, including a board with index drift or invalid documents. ZIP entries start at `.gh-corkboard/` and include the five directories and their stored files. Exclude CLI lock bookkeeping and temporary files. Fail on unsafe entries and publish a completed archive atomically without replacing an existing destination. The destination is outside the board tree. Snapshots are local and can be unpacked for review or recovery.
			"""#
	},
]

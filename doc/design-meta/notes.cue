package flyb

_proseNotes: [
	{
		name:  "board.access"
		title: "Identity and access policy"
		markdown: #"""
			Every board command requires a global `--as <principal>` selector before the command. Principals are `human:<id>`, `agent:<id>`, or `skill:<id>`. The CLI checks registration and resolves the principal's role through `index/access.json`. Help and version output are available without a principal. On first initialization, the supplied human becomes the first administrator. On an existing board, `init` requires an administrator. A missing or damaged policy on an existing board blocks access and must be recovered by the operator. A repeated `init` cannot claim that board.

			The policy is plaintext JSON validated against a built-in version-1 schema. Each principal entry has an ID and an optional role. After schema validation, the CLI resolves an omitted `role` to `member`. An explicit role must be valid and non-null. Check unique principal and topic IDs, one grant per scope and principal with a nonempty set of distinct permissions, registered grant subjects and scopes, and at least one administrator after applying defaults. Only a human can hold `admin`.

			`read` allows discovery, retrieval, and validation. `write` allows creating documents and changing their content, description, milestone, or concept. A topic `delete` grant allows deleting individual documents in that topic. `status` allows setting any lifecycle status and setting or clearing the caller's own review. Each of `write`, `delete`, and `status` implies `read` and can be granted independently where the scope permits it. Board admins have all four permissions in every scope and run board-wide prune. Only a board admin can delete a document from an agent or skill channel.

			Folder removal requires the board `admin` role. `doc delete` retains the folder even when deleting its last document. `access topic delete` removes the topic folder and its grants only when the folder is empty and the scope has no remaining index entries.

			Only a principal with the `admin` role can add or revoke grants in any scope. An admin can also manage principals, create topics, remove empty topics, read and write any document, mutate schemas, and run board-wide audit, resync, prune, and snapshots. With no scope filter, `access list` shows all grants for an admin. A private scope owner can inspect grants by selecting that scope.

			All registered principals can read the shared schema registry. `access whoami` returns the selected principal's role and effective permissions. `access grant` adds one permission and `access revoke` removes one. Both are idempotent. Remove the grant record when its permission set becomes empty. Revoking an explicit `read` grant leaves the read access implied by `write`, `delete`, or `status` in place. Owners and admins retain their implicit rights. Agent and skill owners get read and write in their own channels. Private scopes allow explicit `read` and `status` grants, so an owner needs a separate `status` grant to review or change status. Reject write and delete grants for private scopes in both CLI commands and policy validation. Reject changes that would leave the board without an admin.

			Check the current ACL under the board lock before file access or exposing document metadata. Filter index searches by effective scope permissions and require read access for bulk validation results. A syntactically valid UUID yields the same unavailable error for missing and inaccessible documents. Policy errors fail closed. Reuse the same checks for humans, agents, and skills.

			In local CLI mode, `--as` declares the caller's identity. It provides routing and guardrails for agents using their assigned identity. An agent with unrestricted shell access under the same OS account can impersonate a principal or edit the board directly. For enforcement against a rogue agent, run the CLI through a trusted launcher or broker that binds each agent session to its principal, rejects identity overrides, and keeps board files and administration outside the agent's direct filesystem access. The same JSON policy can be used in that setup.
			"""#
	},
	{
		name:  "board.board"
		title: "Workspace and handoffs"
		markdown: #"""
			The Go binary is `gh-corkboard`, invoked as `gh corkboard`. Board commands require a Git worktree and resolve its root from the current directory, including in a linked worktree. Each board belongs to one checkout and works offline without GitHub authentication. Handoffs require agents to share that board. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)

			A scope is one container for documents and permissions. It can be an agent channel such as `agent:test`, a skill channel such as `skill:review`, or a shared topic such as `topic:specs`. Each scope maps to one directory under `agents/`, `skills/`, or `topics/`.

			```text
			.gh-corkboard/
			├── index/
			│   ├── access.json
			│   └── documents.json
			├── schemas/
			│   ├── csv/
			│   ├── json/
			│   │   └── spec.schema.json
			│   └── md/
			│       └── features.schema.json
			├── agents/
			│   ├── implementation/
			│   ├── specs/
			│   └── test/
			│       └── b69ac8e7-6755-4249-83ef-211daa24ab6f--pagination-review.md
			├── skills/
			│   └── review/
			└── topics/
			    └── specs/
			        ├── 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb--pagination.md
			        └── cf89eab1-3451-4e41-a6bb-0c6110a9d934--pagination.md
			```

			`init` creates the five top-level directories, the three schema format directories, a version-1 access policy and an empty version-1 index on first initialization. It adds `/.gh-corkboard/` as the final root `.gitignore` rule if needed, verifies the rule is effective, and reports tracked corkboard paths. Repeated initialization preserves existing board data.

			For create, the caller supplies a registered scope, a name, a format and optionally a schema ID. Accept slugs matching `[a-z0-9][a-z0-9_-]{0,63}` and reject platform-reserved names. Allocate a UUIDv4 and build the path inside the scope directory. The basename is `<uuid>--<name>.<format>` or `<uuid>--<name>.<schema-id>.<format>`. The UUID makes the filename unique while the name keeps it readable. Validate the derived path every time the index is loaded and use root-confined filesystem operations that reject symlinks and special files. Anchor scope access so a replaced directory cannot redirect IO to a different scope. [Go traversal-resistant file APIs](https://go.dev/blog/osroot)

			A named schema resolves by format and ID. For example, a filename ending in `--pagination.features.md` uses `schemas/md/features.schema.json`. A missing named schema is a validation error.

			Use a document's UUIDv4 to get, modify, delete, validate, or reference it. Create and index search return the UUID as the document's ID. `doc locate` resolves an exact name within a scope, optionally qualified by milestone. The CLI keeps the name, UUID, and path stable through content and metadata updates. For a handoff, include the document's UUID in a shared topic. The recipient needs read access to both the handoff and the referenced document. Self-notes go in the writer's own scope. [UUIDv4 specification](https://www.rfc-editor.org/rfc/rfc9562.html#name-uuid-version-4)
			"""#
	},
	{
		name:  "board.interface"
		title: "Command conventions"
		markdown: #"""
			Board commands in the table follow `gh corkboard --as <principal>`. Run help directly with `gh corkboard help`. Board commands address resources by scope, slug, format, schema ID, document UUID, or prune plan ID. Use stdin for document and schema bytes, and shell redirection for local input or snapshot output. The CLI accepts no filesystem path parameters. Create refuses to replace an existing resource. Update and delete require an existing target. `doc update` requires at least one change flag. Check permission for every requested field before changing document bytes or metadata. `doc validate --all` checks readable indexed documents only.

			Use `--json` on commands that return metadata, mutation summaries, reports or diagnostics. Document metadata results expose UUID, scope, name, format, description, status, and review comments keyed by author. They also expose schema ID, milestone, and concept when present. They omit the internal path and digest. Admin maintenance diagnostics may include paths for manual recovery. Exit 0 for success. Exit 1 for validation failures, audit findings, or malformed document or patch content, with diagnostics on stdout. Exit 2 for usage, authorization, or operational failure, with errors on stderr. Sort search and validation lists lexically by scope, name, case-folded milestone, and UUID. Sort an unset milestone as an empty string.

			`doc get` and `schema get` write only the stored bytes to stdout, with no envelope, status message, or extra newline. Pipe JSON directly to `jq`, and Markdown or CSV to tools suited to those formats. Errors go to stderr. In Bash, use `set -o pipefail` when the pipeline should fail if the get command fails.

			Find default text, JSON, raw streams, and representative diagnostics in the [CLI output examples](command-outputs.md).

			Exercise scoped create and denied cross-scope mutations. Verify that an admin can grant topic access and issue private read and status grants. Verify that a member cannot change grants, including in its own scope. Verify that a writer cannot explicitly set a status without a status grant, a reviewer with only status can set any status and update only its own review, and a combined update fails before any changes if the caller lacks a required permission. Check that locate returns the same ID as create and search, that linked reviews can span scopes, and that inaccessible targets are rejected when linking. Verify that agent and skill owners can edit their own documents and are denied deletion, even if they hold delete access in a topic. Verify that topic delete leaves the folder intact and that a board admin can remove an empty topic with no index entries. Verify formatting on both content write commands, CSV order preservation when requested, and unchanged file and index state when syntax is invalid. Verify that milestone and concept filters combine with other filters after read authorization and that clearing either value leaves the other intact. Also exercise malformed policy and index data, symlink rejection, well-formed drafts that fail schema validation, schema changes, concurrent updates, archived documents edited after preview, and snapshot failures. Build the packaged binary using GitHub CLI naming conventions. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
			"""#
	},
	{
		name:  "board.cli.help"
		title: "Help for the CLI and its subcommands"
		markdown: #"""
			`gh corkboard help` shows the root usage, global flags, command groups, and examples for initializing a board and discovering commands. Running `gh corkboard` with no arguments, `gh corkboard --help`, or `gh corkboard -h` shows the same help.

			Use `gh corkboard help <command> ...` or append `--help` or `-h` to a command path. Group help lists immediate subcommands with short descriptions. Leaf help includes its usage syntax, required arguments and permissions, available flags and defaults, stdin and stdout behavior, and a representative invocation. Commands with format-specific behavior explain which formats and options apply.

			Render help from the same command definitions used for argument parsing. Help works outside a Git worktree without a board, principal, or stdin. It writes plain text to stdout and exits 0. An unknown command path exits 2, sends the error and the closest valid help invocation to stderr, and leaves stdout empty.
			"""#
	},
	{
		name:  "board.dependencies"
		title: "Dependency choices and review scope"
		markdown: #"""
			Start with the five direct modules in the dependency table. Cobra follows the command framework used by `gh-flarebyte`, including explicit stdout and stderr streams. The reviewed revisions use Go 1.26.0 or newer because of `golang.org/x/sys`. Keep `golang.org/x/text` at v0.41.0 or later as an indirect requirement. [x/text advisory](https://pkg.go.dev/vuln/GO-2026-6629)

			The review on 2026-11-21 checked exact versions and used the 100-star floor from the neighboring project. A temporary Go module exercised Cobra help, the Markdown heading parser, offline JSON Schema loading, and strict unified diff application. The macOS, Linux, and Windows package import lists contained eight third-party modules in total. OSV returned no matching advisories for their selected versions. The broader module metadata included tooling requirements with advisories outside those imports. Scan the final application's imports when they exist. [OSV](https://osv.dev/) · [Go Vulnerability Database](https://pkg.go.dev/vuln/)
			"""#
	},
	{
		name:  "board.formatting"
		title: "Automatic formatting on write"
		markdown: #"""
			Every CLI write of document content runs the format-specific formatter before saving. `doc create` formats stdin using the requested format. `doc update --stdin` formats replacement bytes using the indexed document format. `doc patch` formats the patched result. Metadata-only updates and index resync operate on the index. Use deterministic and idempotent rules so the same content and formatting options always produce the same bytes.

			For CSV, sort data records by the first column when its header is `id` or `name`, matched case-insensitively. Compare cell text lexicographically and case-sensitively, keeping the input order for equal keys. For example, IDs `b2` and `a1` appear as `a1` and `b2` after formatting. For other first headers, preserve row order. Use `--preserve-row-order` on a CSV content write when the order carries meaning, including when a schema constrains row positions. It keeps the supplied row order and applies the other CSV formatting rules. Using it for another format or without content to write is a usage error.

			Reject malformed JSON or CSV, including duplicate JSON keys or malformed CSV headers, and invalid UTF-8 in any format before writing the file or index. Report diagnostics against the supplied input. A syntactically valid draft can be saved even when it would fail its declared schema. `doc validate` checks schema conformance separately.

			Compute the index digest from the formatted bytes. During an update, compare those bytes to the current file and rewrite the file only if they differ. Apply any requested metadata changes, and use the archived-to-draft rule when the stored content changes.

			For a successful `doc update --stdin`, set `formattingChanged` in the JSON result to whether the formatter changed the supplied bytes. Include it even when the formatted output already matches the file. The default text output adds `Formatting changed the supplied bytes.` when the value is true. Metadata-only updates omit the field.
			"""#
	},
	{
		name:  "board.patching"
		title: "Patch an existing document"
		markdown: #"""
			`doc patch <doc-id> --expect <sha256>` reads a patch from stdin and requires write access. The indexed document format selects the patch syntax. Keep `doc update --stdin` for full replacements. To create a patch, save the exact bytes from `doc get`, derive the patch from that copy, and pass its lowercase hexadecimal SHA-256 digest to `--expect`. Use the hash of the saved base, including its newlines.

			For Markdown and CSV, accept one textual unified diff with `---` and `+++` headers and at least one hunk. Use `diff -u` on the saved base and an edited copy to generate the hunk ranges. Header names are opaque labels, and `document` works for both. The UUID selects the target. Match every hunk's position and context exactly, including line endings. Reject multi-file patches, binary data, file creation or deletion markers, and Git rename or mode metadata. For CSV, hunks refer to physical lines, including lines inside quoted cells. Parse the full resulting CSV and apply the normal formatting and sorting rules. `--preserve-row-order` works for CSV patches.

			For JSON, use RFC 7396. Merge objects recursively, delete an object member whose patch value is `null`, and replace arrays as whole values. A non-object root patch replaces the root value. Parse the base and patch using the native JSON rules for document content and preserve numeric tokens. The merge operates on values and serializes the result in the canonical JSON format. Use full replacement to give an object member the literal value `null`. [RFC 7396](https://datatracker.ietf.org/doc/html/rfc7396#section-2)

			Under the board lock, check authorization and hash the current file bytes before applying the patch. A changed base or mismatched hunk exits 2 and writes nothing. In JSON mode, stderr carries a diagnostic object with `code` and `message`. Use `patch_base_changed` for a digest mismatch and `patch_context_mismatch` for a hunk mismatch. Read the document again and regenerate the patch from that base. Malformed patch input or a result with invalid native syntax exits 1 with diagnostics and performs no write. Schema validation remains a separate command.

			Apply a patch to a temporary result and use the established document write and index update flow after all checks pass. The document keeps its ID and existing metadata, subject to the usual archived-to-draft rule when bytes change. Successful patch output has the same document metadata shape as a content update. For text patches, `formattingChanged` compares the patched text with the formatted result. When true, the default text output adds `Formatting changed the patched content.` JSON patches produce canonical JSON directly, so their results omit this field.

			Use `go-gitdiff` for parsing unified text and applying it in strict mode to a buffer. Validate the single-document patch shape before applying it. For JSON, use the existing strict decoder and implement the RFC merge over the parsed values, preserving numeric tokens with `json.Number`. The upstream `json-patch/v5` source contains paths that prune nulls from replacement arrays. Check the merge implementation against RFC 7396 Appendix A, nulls inside arrays, null roots, and large numbers. Test the unified diff adapter with strict context, insertions, and the missing-final-newline marker. [Go unified diff parser](https://github.com/bluekeyes/go-gitdiff/blob/master/gitdiff/parser.go) · [Strict text application](https://github.com/bluekeyes/go-gitdiff/blob/master/gitdiff/apply_text.go) · [Go JSON number decoding](https://pkg.go.dev/encoding/json#Decoder.UseNumber) · [Merge Patch implementation](https://github.com/evanphx/json-patch/blob/master/v5/merge.go)
			"""#
	},
	{
		name:  "board.validation"
		title: "Validation rules"
		markdown: #"""
			Use JSON Schema Draft 2020-12 directly. A registry file is `schemas/<format>/<id>.schema.json`. Schema writes compile the definition and reject unsupported dialects and unresolved references. V1 accepts same-resource fragment references and uses bundled Draft 2020-12 meta-schema resources. Fetching schema references from the filesystem or network is disabled. Apply standard annotation semantics for `format`. A schema update changes subsequent validation for documents bound to that ID. Authors can register a new ID such as `spec-v2` when both contracts must coexist. Schema deletion checks bound filenames on disk, including files absent from the index. [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)

			The CSV projection supports required columns, header order, row count and string constraints. For example, a source file with the header `id,priority` and the row `pagination,01` projects to the JSON shown in the CSV projection example. Typed numeric, date and relational constraints would need a future profile. CSV on the Web provides a standardized metadata vocabulary if that interoperability becomes a requirement. [W3C CSV on the Web](https://www.w3.org/TR/tabular-metadata/)

			The Markdown projection supports required headings, order and multiplicity. For example, `## Acceptance **criteria**` yields the heading shown in the Markdown projection example. A schema can check for that entry with `contains` or enforce a sequence with `prefixItems`. Markdown prose and section body content remain outside the v1 schema contract. [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/)

			Well-formed documents remain writable even when schema validation would fail. `doc validate` parses the native format and evaluates any declared schema. It returns `invalid` for syntax or projection errors, a missing declared schema, or schema violations. It returns `schemaless` for well-formed documents without a declared schema and `valid` for documents that satisfy their declared schema. Report the instance path and source coordinates where available. CommonMark accepts free-form text. Schemaless Markdown therefore only requires decodable UTF-8. The index and access policy use built-in schemas and additional Go checks for uniqueness and references. User schema operations cannot alter those contracts.

			CUE supports composition and constraints across fields, and exposes validation and JSON Schema conversion through its Go API. Both engines would consume the same CSV and Markdown projections. Keeping one native JSON Schema engine gives authors reusable JSON artifacts for documents and the index. Revisit CUE if real board schemas need shared constraints or cross-field relationships that become hard to express in JSON Schema. [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)
			"""#
	},
	{
		name:  "board.validation.internal"
		title: "Validating the index and access policy"
		markdown: #"""
			Apply the field rules to the board-owned JSON files `index/documents.json` and `index/access.json`. Enforce the encoded size and nesting limits during parsing, reject duplicate member names before decoding into Go structs or maps, and validate the resulting objects with built-in JSON Schema Draft 2020-12. Run the Go checks for references, derived paths, unique IDs, logical name keys, grant pairs, and the required human administrator. [JSON Schema validation keywords](https://json-schema.org/draft/2020-12/json-schema-validation.html#name-validation-keywords-for-str) · [Go JSON decoding behavior](https://pkg.go.dev/encoding/json#hdr-Security_Considerations)

			Validate the persisted access policy before trusting its grants and the persisted index before using its paths. Validate the prospective internal JSON before changing a document or replacing its metadata. Invalid metadata supplied via CLI flags exits 2 with a field-specific diagnostic and leaves files and index entries unchanged. Treat descriptive text and comments as data, escape control and layout characters when rendering them in terminal summaries, and keep stored paths confined to the board during filesystem access.
			"""#
	},
	{
		name:  "board.index"
		title: "Catalog and reconciliation"
		markdown: #"""
			`index/documents.json` is one catalog for the whole board. Its top-level object has `version: 1` and a `documents` array ordered by canonical path. Each entry stores logical document metadata, its derived board-relative `path`, and the lowercase hexadecimal SHA-256 digest of the indexed bytes. Validate structure with the built-in schema and check that IDs, paths, and logical name keys are unique. Go also checks scope and review author registration and exact path derivation. A malformed index blocks document selections and writes that would depend on it.

			The logical name key is scope, case-folded milestone, and name. An unset milestone is its own namespace. Format and schema do not distinguish names in the same namespace. For example, `pagination` can coexist in `topic:specs` at milestones `v1 launch/Specs` and `v2/Specs`. Check uniqueness under the board lock before create or a milestone update, including clearing the milestone. A collision exits 2 before changing the file or index. A milestone update changes the logical lookup key and retains the filename and UUID.

			`doc locate --in <scope> --name <name> [--milestone <text>]` finds the exact logical name after checking read access to the scope. A supplied milestone matches the entire value case-insensitively after trimming. Supply `--milestone ""` to select an unassigned milestone explicitly. With the flag omitted, look across milestones in that scope and return the UUID for a single indexed match. Multiple matches exit 2 with an ambiguity error that asks for a milestone. Check file presence before returning the UUID and use the same unavailable error for a missing or inaccessible match. Output the UUID on one line or an object with an `id` field in JSON mode.

			`milestone` and `concept` are optional strings on each index entry. Writers choose their own values subject to the field rules. Use `/` to separate hierarchy segments. A concept has 1–48 characters total and each segment uses ASCII letters, as in `Cursor/Pagination`. A milestone has 1–64 characters total and allows digits, spaces, and version punctuation under the field rules, as in `v1 launch/Specs`. The length includes separators. Every segment must be nonempty. These labels are stored as metadata and have no effect on the derived document path.

			Set these fields with `--milestone` and `--concept` on `doc create` or `doc update` with write access. Trim leading and trailing whitespace from the whole flag value before checking it and preserve its case. Omitted flags leave existing values intact on update. An explicitly empty or whitespace-only value clears the field. The index and metadata output omit unset values.

			`index search` starts with all indexed documents for an admin. For other principals, keep only documents in scopes where the caller has at least one effective permission, including the implicit rights on its own channel. Apply this check before matching any search terms or preparing output, in every output format. A search for an inaccessible scope returns an empty successful result.

			Combine supplied filters with AND. `--in` matches one exact logical scope, such as `topic:specs`, `agent:test`, or `skill:review`. `--milestone` and `--concept` match only entries with that field set. Trim outer whitespace from each filter. A term without an asterisk must be a valid field value and matches the entire value case-insensitively. A term with one trailing `*` matches values that begin with the text before it, also case-insensitively. The prefix must be a possible beginning of a valid label and may end at a slash. For example, `--concept 'Cursor/*'` matches `Cursor/Pagination`, and `--milestone 'v1 launch/*'` matches `v1 launch/Specs`. Quote the filter so the shell passes the asterisk literally. A lone `*` matches any set value. Blank terms and asterisks anywhere other than the single trailing position are usage errors.

			`--query` matches a case-insensitive substring in the name, description, milestone, or concept. Search returns indexed metadata, including comments. `doc get` reads the authorized file on disk.

			New entries default to an empty description, `draft` status, and an empty `reviewComments` object. A principal with `status` may set any lifecycle value. An `obsolete` document remains readable by authorized principals and is discouraged for new references. Mark it `archived` when it is ready for admin cleanup. A CLI content write or resync that records changed bytes resets `archived` to `draft`, unless an authorized CLI update supplies a status for the new bytes. Until resync records an external edit, prune checks the digest and skips the changed file.

			`reviewComments` sits alongside the document's lifecycle `status` in each index entry. It maps canonical principal IDs such as `agent:specs` or `agent:test` to one current review per author. Each review contains a nonblank `comment`, a review `status`, and an optional document UUID in `locator`. Review statuses are `pass` for criteria met, `warning` for concerns to call out, and `fail` for unmet criteria. These outcomes are independent of the document lifecycle status.

			`doc update --review-comment <text> --review-status <status>` inserts or replaces the caller's review. Supply `--review-locator <doc-id>` to include a link in that review. `--clear-review-comment` removes the caller's entire review and can be repeated safely. A review replacement and a clear are mutually exclusive. Both require the `status` permission and can be used independently of a lifecycle change. Author attribution comes from the resolved caller and is retained after grant revocation.

			A review can link to any existing indexed document on the board that the reviewer can read. When returning review metadata, the CLI includes the locator only if the target is available to the reader. If the target is later deleted or the reader loses access, the index retains the stored locator and the CLI still returns the review text and outcome.

			Administrators use `index audit` to report missing files, unindexed documents, digest changes, malformed or missing index, and unsupported or conflicting entries. `index resync` preserves metadata at surviving paths subject to the archived-to-draft rule. It drops records for missing files and reports removals and status resets. Retain the UUID in an unindexed canonical filename. For a manually added `<name>.<format>` or `<name>.<schema-id>.<format>` file in a registered scope, allocate a UUID and rename it to the canonical filename. Adopted entries default to `draft` with empty reviews and no milestone or concept.

			Before resync writes anything, reject duplicate UUIDs, a known UUID at a different path, or a logical name collision. Report the conflicting entries so an admin can resolve them. Rename newly adopted files atomically before publishing the reconciled index. If an index write fails, the UUIDs in any renamed files let a retry reuse them. A missing index can be rebuilt with the filename UUIDs, but milestones and other metadata come from the index. If same-name files would collide in the unassigned namespace, recover the index from a snapshot before reconciling. Resync preserves malformed or unsupported indexes for operator recovery.

			All CLI operations affecting the board use an OS-backed lock released on process exit. Mutations take an exclusive lock. Reads hold a compatible lock from authorization through IO so an ACL change cannot interleave with a CLI read. Keep lock bookkeeping under `index/` and write replacements atomically at the individual-file level. Write document bytes before the corresponding index entry, and report partial success if the index write fails. Audit and resync recover file/catalog consistency after interrupted updates. Direct editors can create drift and must finish writes before a snapshot or destructive command.
			"""#
	},
	{
		name:  "board.index.search-output"
		title: "Search output formats"
		markdown: #"""
			Use `index search --format table` or `--format csv` for a compact summary. `--format json` is equivalent to `--json` and returns full document metadata. Specify one output selector per invocation. With neither selector, search prints the detailed text summary.

			Both compact formats start with `id`, `name`, `scope`, and `description`, followed in order by `status`, `milestone`, and `concept` where applicable. Keep `scope` even when filtering with `--in`. Omit `status`, `milestone`, or `concept` when its own filter is an exact match. Keep the corresponding column for an unfiltered field or a trailing-asterisk filter. An unset field appears as an empty cell when its column is present. Choose the columns from the filters so the header stays the same even when the result is empty.

			Cap each compact description at 60 Unicode code points including the suffix. For longer descriptions, keep the first 57 and append `...`. Use one row per document and preserve search result order. CSV writes a lowercase header and standard quoted records. Table output uses uppercase headings and aligned columns. Both write just the header when no documents match. JSON retains full descriptions and metadata regardless of applied filters.
			"""#
	},
	{
		name:  "board.index.path-resolution"
		title: "Internal locator and path resolution"
		markdown: #"""
			The validated document index maps UUIDs to canonical board-relative paths in both directions. A UUID is stable and contains no location information. The filename carries the UUID so it can be recovered during resync. Perform lookups against the current index under the board lock.

			**Locator to path.** Look up the UUID in the index, check the caller's permission for the requested operation on the entry's scope, then use the entry's validated path for root-confined filesystem access. Index validation checks that the path exactly matches the path derived from the entry's scope, UUID, name, format, and optional schema ID. Return the standard unavailable error for a missing or inaccessible document.

			**Path to locator.** Validate the canonical path under a registered scope and look up the exact path in the index. Audit reports an absent entry as unindexed. Resync can read the UUID from the filename and reuse it when adopting that document.

			In the index example, the lookup pairs UUID `b69ac8e7-6755-4249-83ef-211daa24ab6f` with `agents/test/b69ac8e7-6755-4249-83ef-211daa24ab6f--pagination-review.md`. Both conversions are internal storage functions. The public `doc locate` command looks up the logical name in the index and returns the UUID.
			"""#
	},
	{
		name:  "board.retention"
		title: "Prune and archive"
		markdown: #"""
			An admin uses `index prune` to preview archived documents whose recorded digest still matches the disk file. The preview reports skipped archived entries with missing or changed files and emits a deterministic plan ID derived from the sorted candidate IDs, paths and digests. Apply requires `--apply --expect <plan-id>`. Recompute the plan under the board lock and reject stale input before deleting. Recheck each candidate immediately before unlinking. If any unlink fails, stop, reconcile entries for files successfully removed, and report completed and remaining IDs. Report reconciliation failures so `index audit` can identify the drift. Prune eligibility depends on archived status and matching bytes.

			`snapshot create` writes the ZIP to stdout and can archive a board with index drift or invalid documents. ZIP entries start at `.gh-corkboard/` and include the five directories and their stored files, including the access policy. Exclude CLI lock bookkeeping and temporary files. Fail on unsafe entries or output errors. `--expect <plan-id>` compares the current prune plan before the CLI emits bytes. For a pre-prune archive, save the ZIP outside the board and apply the same plan only after the snapshot command succeeds. Prune checks the plan again, so changes to candidates between commands invalidate it. Discard incomplete output after an archive failure.
			"""#
	},
]

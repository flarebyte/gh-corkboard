package flyb

_proseNotes: [
	{
		name:  "board.access"
		title: "Identity and access policy"
		markdown: #"""
			Every board command requires a global `--as <principal>` selector before the command. Principals are `human:<id>`, `agent:<id>`, or `skill:<id>`. The CLI checks registration and resolves the principal's role through `index/access.json`. Help and version output are available without a principal. On first initialization, the supplied human becomes the first administrator. On an existing board, `init` requires an administrator. A missing or damaged policy on an existing board blocks access and must be recovered by the operator. A repeated `init` cannot claim that board.

			The policy is plaintext JSON validated against a built-in version-1 schema. Each principal entry has an ID and an optional role. After schema validation, the CLI resolves an omitted `role` to `member`. An explicit role must be valid and non-null. Check unique principal and topic IDs, one grant per scope and principal with a nonempty set of distinct permissions, registered grant subjects and scopes, and at least one administrator after applying defaults. Only a human can hold `admin`.

			`read` allows discovery, retrieval, and validation. `write` allows creating documents and changing their content or description. A topic `delete` grant allows deleting individual documents in that topic. `status` allows setting any lifecycle status and setting or clearing the caller's own review comment. Each of `write`, `delete`, and `status` implies `read` and can be granted independently where the scope permits it. Board admins have all four permissions in every scope and run board-wide prune. Only a board admin can delete a document from an agent or skill channel.

			Folder removal requires the board `admin` role. `doc delete` retains the folder even when deleting its last document. `access topic delete` removes the topic folder and its grants only when the folder is empty and the scope has no remaining index entries.

			Only a principal with the `admin` role can add or revoke grants in any scope. An admin can also manage principals, create topics, remove empty topics, read and write any document, mutate schemas, and run board-wide audit, resync, prune, and snapshots. With no scope filter, `access list` shows all grants for an admin. A private scope owner can inspect grants by selecting that scope.

			All registered principals can read the shared schema registry. `access whoami` returns the selected principal's role and effective permissions. `access grant` adds one permission and `access revoke` removes one. Both are idempotent. Remove the grant record when its permission set becomes empty. Revoking an explicit `read` grant leaves the read access implied by `write`, `delete`, or `status` in place. Owners and admins retain their implicit rights. Agent and skill owners get read and write in their own channels. Private scopes allow explicit `read` and `status` grants, so an owner needs a separate `status` grant to review or change status. Reject write and delete grants for private scopes in both CLI commands and policy validation. Reject changes that would leave the board without an admin.

			Check the current ACL under the board lock before file access. Apply the read check before returning search or bulk-validation results, descriptions, or other metadata. A syntactically valid UUID yields the same unavailable error for missing and inaccessible documents. Policy errors fail closed. Reuse the same checks for humans, agents, and skills.

			In local CLI mode, `--as` declares the caller's identity. It provides routing and guardrails for agents using their assigned identity. An agent with unrestricted shell access under the same OS account can impersonate a principal or edit the board directly. For enforcement against a rogue agent, run the CLI through a trusted launcher or broker that binds each agent session to its principal, rejects identity overrides, and keeps board files and administration outside the agent's direct filesystem access. The same JSON policy can be used in that setup.
			"""#
	},
	{
		name:  "board.board"
		title: "Workspace and handoffs"
		markdown: #"""
			The Go binary is `gh-corkboard`, invoked as `gh corkboard`. The CLI uses the current Git worktree root, including from a subdirectory or linked worktree. Commands fail outside a Git worktree. Each board belongs to one checkout and works offline without GitHub authentication. Handoffs require agents to share that board. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)

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
			│   └── specs/
			│       └── follow-up.md
			├── skills/
			│   └── review/
			└── topics/
			    └── specs/
			        ├── pagination.spec.json
			        └── pagination.features.md
			```

			`init` creates the five top-level directories, the three schema format directories, a version-1 access policy and an empty version-1 index on first initialization. It adds `/.gh-corkboard/` as the final root `.gitignore` rule if needed, verifies the rule is effective, and reports tracked corkboard paths. Repeated initialization preserves existing board data.

			For create, the caller supplies a registered scope, a name, a format and optionally a schema ID. Accept slugs matching `[a-z0-9][a-z0-9_-]{0,63}` and reject platform-reserved names. The parser accepts only the declared types, enum values and slug form. Build the board-relative path internally from the scope and basename. For example, `topic:specs`, `pagination`, `json` and `spec` map to `topics/specs/pagination.spec.json`. Each managed document is an immediate child of its scope directory. Validate the stored path against the derived path every time the index is loaded, and use root-confined filesystem operations that reject symlinks and special files. Anchor scope access so a replaced directory cannot redirect IO to a different scope. [Go traversal-resistant file APIs](https://go.dev/blog/osroot)

			The basename is `<name>.<format>` or `<name>.<schema-id>.<format>`. A named schema resolves by both format and ID, such as `schemas/md/features.schema.json` for `pagination.features.md`. A missing named schema is a validation error.

			Create returns a random UUIDv4. Use that ID to get, modify, delete, validate, or reference a document. The UUID is an opaque locator. Authorization comes from the current principal and scope policy. IDs stay stable through content and metadata updates. For a handoff, post in a shared topic and include the referenced document's UUID. The recipient needs read access to both the handoff and any referenced document. Self-notes go in the writer's own scope. [UUIDv4 specification](https://www.rfc-editor.org/rfc/rfc9562.html#name-uuid-version-4)
			"""#
	},
	{
		name:  "board.interface"
		title: "Command conventions"
		markdown: #"""
			Commands in the table follow `gh corkboard --as <principal>`. They address board resources by scope, slug, format, schema ID, document UUID, or prune plan ID. Use stdin for document and schema bytes, and shell redirection for local input or snapshot output. The CLI accepts no filesystem path parameters. Create refuses to replace an existing resource. Update and delete require an existing target. `doc update` requires at least one change flag. Check permission for every requested field before changing document bytes or metadata. `doc validate --all` checks readable indexed documents only.

			Use `--json` on commands that return metadata, mutation summaries, reports or diagnostics. Document metadata results expose UUID, scope, name, format, optional schema ID, description, status, and review comments keyed by author. They omit the internal path and digest. Admin maintenance diagnostics may include paths for manual recovery. The two `get` commands and `snapshot create` emit original bytes or ZIP bytes as appropriate. Send usage, authorization and operational errors to stderr. Exit 0 for success, including a clean audit and successful validation of schemaless documents. Exit 1 for validation failures or audit findings. Exit 2 for usage, authorization or operational failure. Search and validation list results use a stable lexical scope and filename order.

			```sh
			gh corkboard --as human:owner init
			gh corkboard --as human:owner access principal add agent:specs
			gh corkboard --as human:owner access principal add agent:implementation
			gh corkboard --as human:owner access principal add agent:test
			gh corkboard --as human:owner access topic create specs
			gh corkboard --as human:owner access grant topic:specs --to agent:specs --permission write
			gh corkboard --as human:owner access grant topic:specs --to agent:specs --permission status
			gh corkboard --as human:owner access grant topic:specs --to agent:implementation --permission read
			gh corkboard --as human:owner access grant topic:specs --to agent:test --permission status
			gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --description "Requirements for cursor pagination" --status active --json < pagination.md
			gh corkboard --as agent:implementation index search --in topic:specs --status active --json
			# Use the document UUID returned by search.
			gh corkboard --as agent:implementation doc get "$doc_id"
			gh corkboard --as agent:specs doc update "$doc_id" --status review --review-comment "Acceptance criteria cover the edge cases."
			gh corkboard --as agent:test doc update "$doc_id" --status approved --review-comment "The test plan covers every criterion."
			gh corkboard --as agent:implementation index search --in topic:specs --status approved --json
			gh corkboard --as agent:implementation doc create --in agent:implementation --name follow-up --format md < follow-up.md
			```

			Exercise scoped create and denied cross-scope mutations. Verify that an admin can grant topic access and issue private read and status grants. Verify that a member cannot change grants, including in its own scope. Verify that a writer cannot explicitly set a status without a status grant, a reviewer with only status can set any status and update only its own comment, and a combined update fails before any changes if the caller lacks a required permission. Verify that agent and skill owners can edit their own documents and are denied deletion, even if they hold delete access in a topic. Verify that topic delete leaves the folder intact and that a board admin can remove an empty topic with no index entries. Also exercise read filtering, malformed policy and index data, symlink rejection, invalid drafts, schema changes, concurrent updates, archived documents edited after preview, and snapshot failures. Build the packaged binary using GitHub CLI naming conventions. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
			"""#
	},
	{
		name:  "board.validation"
		title: "Validation rules"
		markdown: #"""
			Use JSON Schema Draft 2020-12 directly. A registry file is `schemas/<format>/<id>.schema.json`. Schema writes compile the definition and reject unsupported dialects and unresolved references. V1 accepts same-resource fragment references and uses bundled Draft 2020-12 meta-schema resources. Fetching schema references from the filesystem or network is disabled. Apply standard annotation semantics for `format`. A schema update changes subsequent validation for documents bound to that ID. Authors can register a new ID such as `spec-v2` when both contracts must coexist. Schema deletion checks bound filenames on disk, including files absent from the index. [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)

			The CSV projection supports required columns, header order, row count and string constraints. For example, a source file with the header `id,priority` and the row `pagination,01` projects to the JSON shown in the CSV projection example. Typed numeric, date and relational constraints would need a future profile. CSV on the Web provides a standardized metadata vocabulary if that interoperability becomes a requirement. [W3C CSV on the Web](https://www.w3.org/TR/tabular-metadata/)

			The Markdown projection supports required headings, order and multiplicity. For example, `## Acceptance **criteria**` yields the heading shown in the Markdown projection example. A schema can check for that entry with `contains` or enforce a sequence with `prefixItems`. Markdown prose and section body content remain outside the v1 schema contract. [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/)

			Every document remains writable even when validation would fail. On explicit validation, parse the native format first, report syntax or projection errors, then evaluate the declared schema when present. A well-formed plain document returns `schemaless`. Bound documents return `valid` or `invalid`. Report the instance path and source coordinates where available. CommonMark accepts free-form text. Schemaless Markdown therefore only requires decodable UTF-8. The index and access policy use built-in schemas and additional Go checks for uniqueness and references. User schema operations cannot alter those contracts.

			CUE supports composition and constraints across fields, and exposes validation and JSON Schema conversion through its Go API. Both engines would consume the same CSV and Markdown projections. Keeping one native JSON Schema engine gives authors reusable JSON artifacts for documents and the index. Revisit CUE if real board schemas need shared constraints or cross-field relationships that become hard to express in JSON Schema. [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)
			"""#
	},
	{
		name:  "board.index"
		title: "Catalog and reconciliation"
		markdown: #"""
			`index/documents.json` is one catalog for the whole board. Its top-level object has `version: 1` and a `documents` array ordered by canonical path. Each entry stores logical document metadata, its derived board-relative `path`, and the lowercase hexadecimal SHA-256 digest of the indexed bytes. A built-in schema validates structure, the UUID form, status, review comments, and digest. Go checks uniqueness of IDs and paths, that the scope and comment authors are registered, and that the stored path exactly matches the derived path. A malformed index blocks document selections and writes that would depend on it.

			`index search` applies ACL filtering first and combines supplied filters with AND. `--in` matches one logical scope. The query is a case-insensitive substring of the name or description. Search returns indexed metadata, including comments. `doc get` reads the authorized file on disk. New entries default to an empty description, `draft` status, and an empty `reviewComments` object. A principal with `status` may set any lifecycle value. An `obsolete` document remains readable by authorized principals and is discouraged for new references. Mark it `archived` when it is ready for admin cleanup. When CLI or external content changes, a prior `archived` status reverts to `draft` unless the CLI update explicitly sets a status for the new bytes.

			`reviewComments` sits alongside `status` in each index entry. It maps canonical principal IDs such as `agent:specs` or `agent:test` to one current comment per author. The CLI takes the author from the resolved caller identity. `doc update --review-comment <text>` inserts or replaces a nonempty comment under that identity. `--clear-review-comment` removes only that identity's comment and can be repeated safely. Use one comment flag at a time. Both flags require `status` and may be used independently of a status change. Existing comments remain attributed to their authors after grant revocation.

			Administrators use `index audit` to report missing files, unindexed documents, digest changes, malformed or missing index, and unsupported entries. `index resync` adopts files only in registered managed scopes and assigns new UUIDs, draft status, and empty review comments. It preserves UUIDs, descriptions, and comments at surviving paths and preserves status except when bytes change for an archived document. It drops records for missing files and reports those removals and status resets. External renames appear as a missing path and a new file. Resync can rebuild a missing catalog with new IDs. It refuses to overwrite a malformed or unsupported index so its metadata remains recoverable. Use `doc validate --all` to check content in readable indexed documents.

			All CLI operations affecting the board use an OS-backed lock released on process exit. Mutations take an exclusive lock. Reads hold a compatible lock from authorization through IO so an ACL change cannot interleave with a CLI read. Keep lock bookkeeping under `index/` and write replacements atomically at the individual-file level. Write document bytes before the corresponding index entry, and report partial success if the index write fails. Audit and resync recover file/catalog consistency after interrupted updates. Direct editors can create drift and must finish writes before a snapshot or destructive command.
			"""#
	},
	{
		name:  "board.retention"
		title: "Prune and archive"
		markdown: #"""
			An admin uses `index prune` to preview archived documents whose recorded digest still matches the disk file. The preview reports skipped archived entries with missing or changed files and emits a deterministic plan ID derived from the sorted candidate IDs, paths and digests. Apply requires `--apply --expect <plan-id>`. Recompute the plan under the board lock and reject stale input before deleting. Recheck each candidate immediately before unlinking. If any unlink fails, stop, reconcile entries for files successfully removed, and report completed and remaining IDs. Report reconciliation failures so `index audit` can identify the drift. Prune eligibility depends on archived status and matching bytes.

			`snapshot create` writes the ZIP to stdout and can archive a board with index drift or invalid documents. ZIP entries start at `.gh-corkboard/` and include the five directories and their stored files, including the access policy. Exclude CLI lock bookkeeping and temporary files. Fail on unsafe entries or output errors. `--expect <plan-id>` compares the current prune plan before the CLI emits bytes. For a pre-prune archive, save the ZIP outside the board and apply the same plan only after the snapshot command succeeds. Prune checks the plan again, so changes to candidates between commands invalidate it. Discard incomplete output after an archive failure.

			```sh
			gh corkboard --as human:owner index prune --json
			# Use the plan ID from the preview.
			gh corkboard --as human:owner snapshot create --expect "$plan_id" > ../corkboard-before-prune.zip &&
			  gh corkboard --as human:owner index prune --apply --expect "$plan_id"
			```
			"""#
	},
]

package flyb

_proseNotes: [
	{
		name:  "board.access"
		title: "Identity and access policy"
		markdown: #"""
			Every board command requires a global `--as <principal>` selector before the command. Principals are `human:<id>`, `agent:<id>`, or `skill:<id>`. The CLI resolves the principal's state and role through `index/access.json`. Help and version output are available without a principal. On first initialization, the supplied human becomes the first administrator. On an existing board, `init` requires an active administrator. A missing or damaged policy on an existing board blocks access and must be recovered by the operator. A repeated `init` cannot claim that board.

			The policy is plaintext JSON validated against a built-in version-1 schema. Check unique principal and topic IDs, one grant per scope and principal with a nonempty set of distinct permissions, registered grant subjects and scopes, and at least one active administrator. New principals default to `member`. Only a human can hold `admin`. A human admin can assign `topic-admin` to a designated human or agent. Disabling a principal removes its effective access immediately while retaining its folder and grants. Re-enabling it restores access under the current policy.

			| Scope | Default access | Delegation |
			| --- | --- | --- |
			| `agent:<id>` and `skill:<id>` | The matching active principal has read and write access to its scope. Document deletion requires a board admin. | The owner or an admin can grant or revoke read access for another principal. |
			| `topic:<id>` | Document access requires a grant or the `admin` role. | Admins and delegated topic admins manage read, write, and delete grants subject to their role. Each grant applies to the entire topic. |

			`read` allows discovery, retrieval, and validation. `write` allows creation and updates and implies `read`. `delete` allows explicit document removal and implies `read`. In topics, a principal can hold either `write` or `delete` independently. Setting `obsolete` requires both permissions because it makes the document eligible for bulk cleanup. On an agent or skill channel, only a board admin can set `obsolete` or delete a document. Board-wide prune requires an administrator.

			An active admin can manage all scopes and principals, read and write any document, mutate schemas, and run board-wide audit, resync, prune, and snapshots. A topic admin can list principals, create topics, delete empty topics, inspect topic grants, and grant or revoke topic access for principals with the `member` role. A human admin manages roles and grants topic document access to a topic admin when needed. The `topic-admin` role confers no document access or private scope administration on its own. With no scope filter, `access list` shows all grants for an admin and only topic grants for a topic admin. A private scope owner can inspect grants by selecting that scope.

			All active principals can read the shared schema registry. `access whoami` returns the selected principal's role and effective permissions. `access grant` adds one permission and `access revoke` removes one. Both are idempotent. Remove the grant record when its permission set becomes empty. Revoking an explicit `read` grant leaves the read access implied by `write` or `delete` in place. Owners and admins retain their implicit rights. Private scope grants allow only `read`. Reject write and delete grants for private scopes in both CLI commands and policy validation, including grants to the owner. Topic deletion requires an empty directory and no remaining index entries for that scope. Reject changes that would leave the board without an active admin.

			```json
			{
			  "version": 1,
			  "principals": [
			    {"id": "human:owner", "role": "admin", "state": "active"},
			    {"id": "agent:coordinator", "role": "topic-admin", "state": "active"},
			    {"id": "agent:specs", "role": "member", "state": "active"},
			    {"id": "agent:implementation", "role": "member", "state": "active"}
			  ],
			  "topics": ["specs"],
			  "grants": [
			    {"scope": "topic:specs", "principal": "agent:specs", "permissions": ["write"]},
			    {"scope": "topic:specs", "principal": "agent:implementation", "permissions": ["read"]}
			  ]
			}
			```

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
			Commands in the table follow `gh corkboard --as <principal>`. They address board resources by scope, slug, format, schema ID, document UUID, or prune plan ID. Use stdin for document and schema bytes, and shell redirection for local input or archived output. The CLI accepts no filesystem path parameters. Create refuses to replace an existing resource. Update and delete require an existing target. Updates change only explicitly supplied fields. `doc update` and `access principal set` require at least one change flag. `doc validate --all` checks readable indexed documents only.

			Use `--json` on commands that return metadata, mutation summaries, reports or diagnostics. Normal document results expose UUID, scope, name, format, optional schema ID, description and status. They omit the internal path and digest. Admin maintenance diagnostics may include paths for manual recovery. The two `get` commands and `snapshot create` emit original bytes or ZIP bytes as appropriate. Send usage, authorization and operational errors to stderr. Exit 0 for success, including a clean audit and successful validation of schemaless documents. Exit 1 for validation failures or audit findings. Exit 2 for usage, authorization or operational failure. Search and validation list results use a stable lexical scope and filename order.

			```sh
			gh corkboard --as human:owner init
			gh corkboard --as human:owner access principal add agent:coordinator --role topic-admin
			gh corkboard --as human:owner access principal add agent:specs
			gh corkboard --as human:owner access principal add agent:implementation
			gh corkboard --as agent:coordinator access topic create specs
			gh corkboard --as agent:coordinator access grant topic:specs --to agent:specs --permission write
			gh corkboard --as agent:coordinator access grant topic:specs --to agent:implementation --permission read
			gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --status active --json < pagination.md
			gh corkboard --as agent:implementation index search --in topic:specs --status active --json
			# Use the document UUID returned by search.
			gh corkboard --as agent:implementation doc get "$doc_id"
			gh corkboard --as agent:implementation doc create --in agent:implementation --name follow-up --format md < follow-up.md
			```

			Exercise scoped create, denied cross-scope mutations, owner read sharing, topic administration by a delegated agent, and denial of attempts by that agent to grant itself topic access. Verify that agent and skill owners can edit their own documents and are denied deletion and obsolete status, even if they hold delete access in a topic. Verify admin deletion in private scopes, rejection of private delete grants, and revoking topic delete while preserving write. Also exercise UUID lookups after grant revocation, read filtering in search and bulk validation, malformed policy and forged index paths, symlink rejection, invalid drafts, schema changes, concurrent updates, stale prune plans, and archive failures. Build the packaged binary using GitHub CLI naming conventions. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)
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

			Every document remains writable even when validation would fail. On explicit validation, parse the native format first, report syntax or projection errors, then evaluate the declared schema when present. A well-formed plain document returns `schemaless`. Bound documents return `valid` or `invalid`. Report the instance path and source coordinates where available. CommonMark accepts free-form text. Schemaless Markdown therefore only requires decodable UTF-8. The index and access policy use built-in schemas and additional Go checks for uniqueness and references. User schema operations cannot alter those contracts.

			CUE supports composition and constraints across fields, and exposes validation and JSON Schema conversion through its Go API. Both engines would consume the same CSV and Markdown projections. Keeping one native JSON Schema engine gives authors reusable JSON artifacts for documents and the index. Revisit CUE if real board schemas need shared constraints or cross-field relationships that become hard to express in JSON Schema. [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)
			"""#
	},
	{
		name:  "board.index"
		title: "Catalog and reconciliation"
		markdown: #"""
			`index/documents.json` is one catalog for the whole board. Its top-level object has `version: 1` and a `documents` array ordered by canonical path. Each entry stores logical document metadata, its derived board-relative `path`, and the lowercase hexadecimal SHA-256 digest of the indexed bytes. A built-in schema validates structure, the UUID form, status and digest. Go checks uniqueness of IDs and paths, that the scope is registered, and that the stored path exactly matches the derived path. A malformed index blocks document selections and writes that would depend on it.

			`index search` applies ACL filtering first and combines supplied filters with AND. `--in` matches one logical scope. The query is a case-insensitive substring of the name or description. Search returns indexed metadata. `doc get` reads the authorized file on disk. New entries default to an empty description and `draft`. Setting `obsolete` requires both write and delete access. When CLI or external content changes, a prior `obsolete` status reverts to `draft` unless an authorized CLI write explicitly sets a status for the new bytes.

			Administrators use `index audit` to report missing files, unindexed documents, digest changes, malformed or missing index, and unsupported entries. `index resync` adopts files only in registered managed scopes and assigns new UUIDs and draft status. It preserves UUIDs and descriptions at surviving paths and preserves status except when bytes change for an obsolete document. It drops records for missing files and reports those removals and status resets. External renames appear as a missing path and a new file. Resync can rebuild a missing catalog with new IDs. It refuses to overwrite a malformed or unsupported index so its metadata remains recoverable. Use `doc validate --all` to check content in readable indexed documents.

			All CLI operations affecting the board use an OS-backed lock released on process exit. Mutations take an exclusive lock. Reads hold a compatible lock from authorization through IO so an ACL change cannot interleave with a CLI read. Keep lock bookkeeping under `index/` and write replacements atomically at the individual-file level. Write document bytes before the corresponding index entry, and report partial success if the index write fails. Audit and resync recover file/catalog consistency after interrupted updates. Direct editors can create drift and must finish writes before a snapshot or destructive command.
			"""#
	},
	{
		name:  "board.retention"
		title: "Prune and archive"
		markdown: #"""
			An admin uses `index prune` to preview explicitly obsolete documents whose recorded digest still matches the disk file. The preview reports skipped obsolete entries with missing or changed files and emits a deterministic plan ID derived from the sorted candidate IDs, paths and digests. Apply requires `--apply --expect <plan-id>`. Recompute the plan under the board lock and reject stale input before deleting. Recheck each candidate immediately before unlinking. If any unlink fails, stop, reconcile entries for files successfully removed, and report completed and remaining IDs. Report reconciliation failures so `index audit` can identify the drift. Resolved, invalid, schemaless and unindexed documents have no implicit deletion eligibility.

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

# gh-corkboard v1 specification

Proposed specification for a local project corkboard and messaging system.

## Access control

### Principals and permissions

#### Identity and access policy

Every board command requires a global `--as <principal>` selector before the command. Principals are `human:<id>`, `agent:<id>`, or `skill:<id>`. The CLI checks registration and resolves the principal's role through `index/access.json`. Help and version output are available without a principal. On first initialization, the supplied human becomes the first administrator. On an existing board, `init` requires an administrator. A missing or damaged policy on an existing board blocks access and must be recovered by the operator. A repeated `init` cannot claim that board.

The policy is plaintext JSON validated against a built-in version-1 schema. Each principal entry has an ID and an optional role. After schema validation, the CLI resolves an omitted `role` to `member`. An explicit role must be valid and non-null. Check unique principal and topic IDs, one grant per scope and principal with a nonempty set of distinct permissions, registered grant subjects and scopes, and at least one administrator after applying defaults. Only a human can hold `admin`.

`read` allows discovery, retrieval, and validation. `write` allows creating documents and changing their content, description, milestone, or concept. A topic `delete` grant allows deleting individual documents in that topic. `status` allows setting any lifecycle status and setting or clearing the caller's own review. Each of `write`, `delete`, and `status` implies `read` and can be granted independently where the scope permits it. Board admins have all four permissions in every scope and run board-wide prune. Only a board admin can delete a document from an agent or skill channel.

Folder removal requires the board `admin` role. `doc delete` retains the folder even when deleting its last document. `access topic delete` removes the topic folder and its grants only when the folder is empty and the scope has no remaining index entries.

Only a principal with the `admin` role can add or revoke grants in any scope. An admin can also manage principals, create topics, remove empty topics, read and write any document, mutate schemas, and run board-wide audit, resync, prune, and snapshots. With no scope filter, `access list` shows all grants for an admin. A private scope owner can inspect grants by selecting that scope.

All registered principals can read the shared schema registry. `access whoami` returns the selected principal's role and effective permissions. `access grant` adds one permission and `access revoke` removes one. Both are idempotent. Remove the grant record when its permission set becomes empty. Revoking an explicit `read` grant leaves the read access implied by `write`, `delete`, or `status` in place. Owners and admins retain their implicit rights. Agent and skill owners get read and write in their own channels. Private scopes allow explicit `read` and `status` grants, so an owner needs a separate `status` grant to review or change status. Reject write and delete grants for private scopes in both CLI commands and policy validation. Reject changes that would leave the board without an admin.

Check the current ACL under the board lock before file access or exposing document metadata. Filter results from `doc search` by effective scope permissions and require read access for bulk validation results. A syntactically valid UUID yields the same unavailable error for missing and inaccessible documents. Policy errors fail closed. Reuse the same checks for humans, agents, and skills.

In local CLI mode, `--as` declares the caller's identity. It provides routing and guardrails for agents using their assigned identity. An agent with unrestricted shell access under the same OS account can impersonate a principal or edit the board directly. For enforcement against a rogue agent, run the CLI through a trusted launcher or broker that binds each agent session to its principal, rejects identity overrides, and keeps board files and administration outside the agent's direct filesystem access. The same JSON policy can be used in that setup.

#### Permissions by scope

| default access | grants available to admins | scope |
| --- | --- | --- |
| The matching registered principal has read and write access to its scope. Document deletion requires a board admin. | `read` for other principals and `status` for any principal. | `agent:<id>` and `skill:<id>` |
| Document access requires a document grant or the `admin` role. | `read`, `write`, `delete`, and `status`. | `topic:<id>` |

#### Access policy example

```json
{
  "version": 1,
  "principals": [
    {"id": "human:owner", "role": "admin"},
    {"id": "agent:specs"},
    {"id": "agent:implementation"},
    {"id": "agent:test"}
  ],
  "topics": ["specs"],
  "grants": [
    {"scope": "agent:test", "principal": "agent:implementation", "permissions": ["read"]},
    {"scope": "topic:specs", "principal": "agent:specs", "permissions": ["write", "status"]},
    {"scope": "topic:specs", "principal": "agent:implementation", "permissions": ["read"]},
    {"scope": "topic:specs", "principal": "agent:test", "permissions": ["status"]}
  ]
}
```

## Board and communication

### Use cases

#### Core workflows

| actor | concept | id | outcome | workflow |
| --- | --- | --- | --- | --- |
| Specification agent | board.communication | handoff | The implementation agent reads the handoff and follows the link if it can read the spec | Publish a handoff in a shared topic with a link to the spec UUID |
| Any agent | board.communication | self-note | The agent can recover local context in a later session | Write under its own agent scope |
| Skill principal | board.access | skill-reference | The agents can find guidance by scope or description | Ask a board admin to grant read access to selected agents |
| Specification and test agents | board.index | review | Readers can see the comments and open linked documents where they have read access | Record separate review outcomes and link to supporting documents by UUID |
| Any agent | board.index | find-work | The agent can find a small set of related documents for the current milestone | Filter readable documents by milestone and concept |
| Document author | board.documents | reuse-name | Each document has its own UUID and the milestone resolves which pagination document to open | Reuse the name pagination in the same scope for two milestones |
| Document author | board.validation | draft | The draft is formatted on write and the validator reports the missing fields | Save a well-formed draft missing required schema fields and run validation |
| Document author | board.patching | patch | The patch updates one document or reports a stale base without changing it | Send a small patch against the exact bytes returned by doc get |
| Board administrator | board.retention | retire | The selected files are removed and the prior board can be inspected offline | Preview archived documents then save a ZIP snapshot before applying cleanup |

### Workspace

#### Workspace and handoffs

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

Use a document's UUIDv4 to get, modify, delete, validate, or reference it. `doc create` and `doc search` return the UUID as the document's ID. `doc locate` resolves an exact name within a scope, optionally qualified by milestone. The CLI keeps the name, UUID, and path stable through content and metadata updates. For a handoff, include the document's UUID in a shared topic. The recipient needs read access to both the handoff and the referenced document. Self-notes go in the writer's own scope. [UUIDv4 specification](https://www.rfc-editor.org/rfc/rfc9562.html#name-uuid-version-4)

## Command interface

### Commands

#### Command reference

| access | command | concept | contract |
| --- | --- | --- | --- |
| Public | `help [<command> ...]` | board.cli | Show top-level or command-specific help without opening a board |
| Human bootstrap or existing admin | `init` | board.cli | Initialize the worktree board and ensure a root ignore rule |
| Registered principal | `access whoami` | board.access | Return the selected principal and its effective permissions by scope |
| Admin | `access principal add <principal> [--role <member\|admin>]` | board.access | Register a principal and provision its folder for an agent or skill |
| Admin | `access principal list` | board.access | List registered principals and their roles |
| Admin | `access principal set <principal> --role <member\|admin>` | board.access | Change a principal's role while retaining existing documents and grants |
| Admin | `access topic create <topic-id>` | board.access | Register a topic with no grants |
| Admin | `access topic delete <topic-id>` | board.access | Remove an empty topic folder with no remaining index entries and remove its grants |
| Admin or private scope owner | `access list [--scope <scope>]` | board.access | Show grants the caller is authorized to inspect |
| Admin | `access grant <scope> --to <principal> --permission <read\|write\|delete\|status>` | board.access | Add one explicit permission allowed for the scope |
| Admin | `access revoke <scope> --from <principal> --permission <read\|write\|delete\|status>` | board.access | Remove one explicit permission allowed for the scope |
| Admin | `schema create <format> <schema-id>` | board.validation | Register a valid schema from stdin |
| Registered principal | `schema get <format> <schema-id>` | board.validation | Write stored schema bytes to stdout |
| Registered principal | `schema list [--format <format>]` | board.validation | List registered schemas ordered by format and ID |
| Admin | `schema update <format> <schema-id>` | board.validation | Replace an existing schema with a valid definition from stdin |
| Admin | `schema delete <format> <schema-id>` | board.validation | Delete an unused schema |
| Scope write plus status when supplied | `doc create --in <scope> --name <name> --format <format> [--schema <schema-id>] [--description <text>] [--milestone <text>] [--concept <text>] [--status <status>] [--preserve-row-order]` | board.formatting | Create a unique name within the selected scope and milestone and return its UUID |
| Scope read | `doc get <doc-id>` | board.documents | Write stored document bytes to stdout |
| Scope read | `doc locate --in <scope> --name <name> [--milestone <text>]` | board.documents | Resolve an exact name within a scope and require the milestone when the name is ambiguous |
| Registered principal | `doc search [--in <scope>] [--status <status>] [--milestone <text>] [--concept <text>] [--query <text>] [--format <table\|csv\|json>]` | board.documents | Filter accessible documents by scope and metadata with board-wide access for admins |
| Scope write | `doc patch <doc-id> --expect <sha256> [--preserve-row-order]` | board.patching | Apply a patch from stdin to the exact base revision and format the result |
| Scope write for content, description, milestone or concept. Scope status for lifecycle or own review. | `doc update <doc-id> [--stdin [--preserve-row-order]] [--description <text>] [--milestone <text>] [--concept <text>] [--status <status>] [--review-comment <text> --review-status <pass\|warning\|fail> [--review-locator <doc-id>]\|--clear-review-comment]` | board.formatting | Apply requested content or metadata updates and report any formatting changes to replacement bytes |
| Topic delete or board admin | `doc delete <doc-id>` | board.documents | Delete one document and remove its index entry while retaining the folder |
| Scope read | `doc validate <doc-id\|--all>` | board.validation | Validate the selected document or all readable indexed documents |
| Admin | `index audit` | board.index | Report filesystem and index discrepancies without changing board data |
| Admin | `index resync` | board.index | Reconcile the catalog with managed locations and preserve metadata and filename UUIDs |
| Admin | `index prune [--apply --expect <plan-id>]` | board.retention | Preview or apply removal of archived documents |
| Admin | `snapshot create [--expect <plan-id>]` | board.retention | Write a ZIP of the board to stdout and optionally verify a prune plan first |

#### Command conventions

Board commands in the table follow `gh corkboard --as <principal>`. Run help directly with `gh corkboard help`. Board commands address resources by scope, slug, format, schema ID, document UUID, or prune plan ID. Use stdin for document and schema bytes, and shell redirection for local input or snapshot output. The CLI accepts no filesystem path parameters. Create refuses to replace an existing resource. Update and delete require an existing target. `doc update` requires at least one change flag. Check permission for every requested field before changing document bytes or metadata. `doc validate --all` checks readable indexed documents only.

Use `--json` on commands that return metadata, mutation summaries, reports or diagnostics. Document metadata results expose UUID, scope, name, format, description, status, and review comments keyed by author. They also expose schema ID, milestone, and concept when present. They omit the internal path and digest. Admin maintenance diagnostics may include paths for manual recovery. Exit 0 for success. Exit 1 for validation failures, audit findings, or malformed document or patch content, with diagnostics on stdout. Exit 2 for usage, authorization, or operational failure, with errors on stderr. Sort search and validation lists lexically by scope, name, case-folded milestone, and UUID. Sort an unset milestone as an empty string.

`doc get` and `schema get` write only the stored bytes to stdout, with no envelope, status message, or extra newline. Pipe JSON directly to `jq`, and Markdown or CSV to tools suited to those formats. Errors go to stderr. In Bash, use `set -o pipefail` when the pipeline should fail if the get command fails.

Find default text, JSON, raw streams, and representative diagnostics in the [CLI output examples](command-outputs.md).

Exercise scoped create and denied cross-scope mutations. Verify that an admin can grant topic access and issue private read and status grants. Verify that a member cannot change grants, including in its own scope. Verify that a writer cannot explicitly set a status without a status grant, a reviewer with only status can set any status and update only its own review, and a combined update fails before any changes if the caller lacks a required permission. Check that locate returns the same ID as create and search, that linked reviews can span scopes, and that inaccessible targets are rejected when linking. Verify that agent and skill owners can edit their own documents and are denied deletion, even if they hold delete access in a topic. Verify that topic delete leaves the folder intact and that a board admin can remove an empty topic with no index entries. Verify formatting on both content write commands, CSV order preservation when requested, and unchanged file and index state when syntax is invalid. Verify that milestone and concept filters combine with other filters after read authorization and that clearing either value leaves the other intact. Also exercise malformed policy and index data, symlink rejection, well-formed drafts that fail schema validation, schema changes, concurrent updates, archived documents edited after preview, and snapshot failures. Build the packaged binary using GitHub CLI naming conventions. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)

#### Review and handoff example

```bash
#!/usr/bin/env bash

# Source this file in Bash and call example_review_workflow in a fresh Git worktree.
# Have pagination.md, pagination-review.md, and follow-up.md in the current directory.
example_review_workflow() (
  set -euo pipefail

  gh corkboard --as human:owner init
  gh corkboard --as human:owner access principal add agent:specs
  gh corkboard --as human:owner access principal add agent:implementation
  gh corkboard --as human:owner access principal add agent:test
  gh corkboard --as human:owner access topic create specs
  gh corkboard --as human:owner access grant topic:specs --to agent:specs --permission write
  gh corkboard --as human:owner access grant topic:specs --to agent:specs --permission status
  gh corkboard --as human:owner access grant topic:specs --to agent:implementation --permission read
  gh corkboard --as human:owner access grant topic:specs --to agent:test --permission status
  gh corkboard --as human:owner access grant agent:test --to agent:implementation --permission read
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md \
    --description "Requirements for cursor pagination" --milestone "v1 launch/Specs" --concept Cursor/Pagination \
    --status active --json < pagination.md
  doc_id=$(gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination \
    --milestone "v1 launch/Specs")
  gh corkboard --as agent:implementation doc get "$doc_id"
  gh corkboard --as agent:specs doc update "$doc_id" --status review --review-status pass --review-comment "Acceptance criteria cover the edge cases."
  gh corkboard --as agent:test doc create --in agent:test --name pagination-review --format md --description "Test review of the cursor pagination spec" < pagination-review.md
  review_id=$(gh corkboard --as agent:test doc locate --in agent:test --name pagination-review)
  gh corkboard --as agent:test doc update "$doc_id" --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator "$review_id"
  gh corkboard --as agent:implementation doc search --in topic:specs --status approved \
    --milestone "v1 launch/specs" --concept cursor/pagination --json
  gh corkboard --as agent:implementation doc get "$review_id"
  gh corkboard --as agent:implementation doc create --in agent:implementation --name follow-up --format md < follow-up.md
)
```

### Help and discovery

#### Help for the CLI and its subcommands

`gh corkboard help` shows the root usage, global flags, command groups, and examples for initializing a board and discovering commands. Running `gh corkboard` with no arguments, `gh corkboard --help`, or `gh corkboard -h` shows the same help.

Use `gh corkboard help <command> ...` or append `--help` or `-h` to a command path. Group help lists immediate subcommands with short descriptions. Leaf help includes its usage syntax, required arguments and permissions, available flags and defaults, stdin and stdout behavior, and a representative invocation. Commands with format-specific behavior explain which formats and options apply.

Render help from the same command definitions used for argument parsing. Help works outside a Git worktree without a board, principal, or stdin. It writes plain text to stdout and exits 0. An unknown command path exits 2, sends the error and the closest valid help invocation to stderr, and leaves stdout empty.

## Implementation

### Go dependencies

#### Dependency choices and review scope

Start with the five direct modules in the dependency table. Cobra follows the command framework used by `gh-flarebyte`, including explicit stdout and stderr streams. The reviewed revisions use Go 1.26.0 or newer because of `golang.org/x/sys`. Keep `golang.org/x/text` at v0.41.0 or later as an indirect requirement. [x/text advisory](https://pkg.go.dev/vuln/GO-2026-6629)

The review on 2026-11-21 checked exact versions and used the 100-star floor from the neighboring project. A temporary Go module exercised Cobra help, the Markdown heading parser, offline JSON Schema loading, and strict unified diff application. The macOS, Linux, and Windows package import lists contained eight third-party modules in total. OSV returned no matching advisories for their selected versions. The broader module metadata included tooling requirements with advisories outside those imports. Scan the final application's imports when they exist. [OSV](https://osv.dev/) · [Go Vulnerability Database](https://pkg.go.dev/vuln/)

#### Reviewed direct dependencies

| concept | github_stars | license | minimum_go | module | purpose | security_review | upstream | version |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| board.dependencies | 44809 | Apache-2.0 | 1.15 | github.com/spf13/cobra | Nested commands and flags and contextual help | No matching OSV advisory for this release. Uses the same command framework as gh-flarebyte. | https://github.com/spf13/cobra | v1.10.2 |
| board.dependencies | 5111 | MIT | 1.22 | github.com/yuin/goldmark | Parse CommonMark and extract headings from its AST | No matching OSV advisory for this release. Includes the fix for GO-2026-5320 in v1.7.17. Use the parser without HTML rendering. | https://github.com/yuin/goldmark | v1.8.7 |
| board.dependencies | 1298 | Apache-2.0 | 1.21 | github.com/santhosh-tekuri/jsonschema/v6 | Validate Draft 2020-12 document schemas and built-in metadata schemas | No matching OSV advisory for this release. Resolve indirect golang.org/x/text at v0.41.0 or later for GO-2026-6629. Use a loader restricted to bundled schema resources. | https://github.com/santhosh-tekuri/jsonschema | v6.0.3 |
| board.dependencies | 159 | MIT | 1.21 | github.com/bluekeyes/go-gitdiff | Parse and strictly apply single-document unified text patches | No matching OSV advisory for this release. Apply to a buffer after checking the single-document patch shape. Header labels carry no filesystem authority. | https://github.com/bluekeyes/go-gitdiff | v0.9.0 |
| board.dependencies | 1364 | BSD-3-Clause | 1.26.0 | golang.org/x/sys | Lock an already opened board file with unix.Flock or windows.LockFileEx | No matching OSV advisory for this release. Includes the fix for GO-2026-5024 in v0.44.0. Lock the descriptor from the root-confined open. | https://github.com/golang/sys | v0.50.0 |

## Specification sources

### Concept registry

#### Concept registry

| description | id | parent |
| --- | --- | --- |
| Local project corkboard | board |  |
| Principals and scope permissions | board.access | board |
| Go extension command interface | board.cli | board |
| Agent handoffs self-notes and human review | board.communication | board |
| Go dependency choices and advisory evidence | board.dependencies | board |
| Document formats names and lifecycle | board.documents | board |
| Automatic formatting on document writes | board.formatting | board |
| Metadata search and filesystem reconciliation | board.index | board |
| Revision checked document patches | board.patching | board |
| Obsolete documents and snapshots | board.retention | board |
| Schema registry and validation profiles | board.validation | board |
| Strict validation of the index and access policy | board.validation.internal | board.validation |
| Principal topic and grant field validation | board.validation.internal.access | board.validation.internal |
| Document index field validation | board.validation.internal.index | board.validation.internal |

## Storage and validation

### Document formatting

#### Automatic formatting on write

Every CLI write of document content runs the format-specific formatter before saving. `doc create` formats stdin using the requested format. `doc update --stdin` formats replacement bytes using the indexed document format. `doc patch` formats the patched result. Metadata-only updates and index resync operate on the index. Use deterministic and idempotent rules so the same content and formatting options always produce the same bytes.

For CSV, sort data records by the first column when its header is `id` or `name`, matched case-insensitively. Compare cell text lexicographically and case-sensitively, keeping the input order for equal keys. For example, IDs `b2` and `a1` appear as `a1` and `b2` after formatting. For other first headers, preserve row order. Use `--preserve-row-order` on a CSV content write when the order carries meaning, including when a schema constrains row positions. It keeps the supplied row order and applies the other CSV formatting rules. Using it for another format or without content to write is a usage error.

Reject malformed JSON or CSV, including duplicate JSON keys or malformed CSV headers, and invalid UTF-8 in any format before writing the file or index. Report diagnostics against the supplied input. A syntactically valid draft can be saved even when it would fail its declared schema. `doc validate` checks schema conformance separately.

Compute the index digest from the formatted bytes. During an update, compare those bytes to the current file and rewrite the file only if they differ. Apply any requested metadata changes, and use the archived-to-draft rule when the stored content changes.

For a successful `doc update --stdin`, set `formattingChanged` in the JSON result to whether the formatter changed the supplied bytes. Include it even when the formatted output already matches the file. The default text output adds `Formatting changed the supplied bytes.` when the value is true. Metadata-only updates omit the field.

#### Formatting rules by format

| format | formatting rules |
| --- | --- |
| Markdown | Normalize line endings to LF and ensure a final newline for nonempty documents. Preserve author layout including paragraph wrapping, line breaks created by trailing spaces, list indentation, and the contents and indentation of code blocks apart from newline normalization. |
| JSON | Indent with two spaces and use one final newline. Sort object keys lexicographically at every level. Preserve array order, string values, and the numeric token spelling so formatting cannot lose precision. |
| CSV | Use comma delimiters and LF record endings. Quote fields containing commas, double quotes or line breaks, and escape embedded quotes by doubling them. Preserve header order and cell text, including whitespace, leading zeros, and newlines inside quoted fields. Quote an empty cell when it is the only field in a record so the record survives parsing. |

### Document patching

#### Patch an existing document

`doc patch <doc-id> --expect <sha256>` reads a patch from stdin and requires write access. The indexed document format selects the patch syntax. Keep `doc update --stdin` for full replacements. To create a patch, save the exact bytes from `doc get`, derive the patch from that copy, and pass its lowercase hexadecimal SHA-256 digest to `--expect`. Use the hash of the saved base, including its newlines.

For Markdown and CSV, accept one textual unified diff with `---` and `+++` headers and at least one hunk. Use `diff -u` on the saved base and an edited copy to generate the hunk ranges. Header names are opaque labels, and `document` works for both. The UUID selects the target. Match every hunk's position and context exactly, including line endings. Reject multi-file patches, binary data, file creation or deletion markers, and Git rename or mode metadata. For CSV, hunks refer to physical lines, including lines inside quoted cells. Parse the full resulting CSV and apply the normal formatting and sorting rules. `--preserve-row-order` works for CSV patches.

For JSON, use RFC 7396. Merge objects recursively, delete an object member whose patch value is `null`, and replace arrays as whole values. A non-object root patch replaces the root value. Parse the base and patch using the native JSON rules for document content and preserve numeric tokens. The merge operates on values and serializes the result in the canonical JSON format. Use full replacement to give an object member the literal value `null`. [RFC 7396](https://datatracker.ietf.org/doc/html/rfc7396#section-2)

Under the board lock, check authorization and hash the current file bytes before applying the patch. A changed base or mismatched hunk exits 2 and writes nothing. In JSON mode, stderr carries a diagnostic object with `code` and `message`. Use `patch_base_changed` for a digest mismatch and `patch_context_mismatch` for a hunk mismatch. Read the document again and regenerate the patch from that base. Malformed patch input or a result with invalid native syntax exits 1 with diagnostics and performs no write. Schema validation remains a separate command.

Apply a patch to a temporary result and use the established document write and index update flow after all checks pass. The document keeps its ID and existing metadata, subject to the usual archived-to-draft rule when bytes change. Successful patch output has the same document metadata shape as a content update. For text patches, `formattingChanged` compares the patched text with the formatted result. When true, the default text output adds `Formatting changed the patched content.` JSON patches produce canonical JSON directly, so their results omit this field.

Use `go-gitdiff` for parsing unified text and applying it in strict mode to a buffer. Validate the single-document patch shape before applying it. For JSON, use the existing strict decoder and implement the RFC merge over the parsed values, preserving numeric tokens with `json.Number`. The upstream `json-patch/v5` source contains paths that prune nulls from replacement arrays. Check the merge implementation against RFC 7396 Appendix A, nulls inside arrays, null roots, and large numbers. Test the unified diff adapter with strict context, insertions, and the missing-final-newline marker. [Go unified diff parser](https://github.com/bluekeyes/go-gitdiff/blob/master/gitdiff/parser.go) · [Strict text application](https://github.com/bluekeyes/go-gitdiff/blob/master/gitdiff/apply_text.go) · [Go JSON number decoding](https://pkg.go.dev/encoding/json#Decoder.UseNumber) · [Merge Patch implementation](https://github.com/evanphx/json-patch/blob/master/v5/merge.go)

### Index and lifecycle

#### Catalog and reconciliation

`index/documents.json` is one catalog for the whole board. Its top-level object has `version: 1` and a `documents` array ordered by canonical path. Each entry stores logical document metadata, its derived board-relative `path`, and the lowercase hexadecimal SHA-256 digest of the indexed bytes. Validate structure with the built-in schema and check that IDs, paths, and logical name keys are unique. Go also checks scope and review author registration and exact path derivation. A malformed index blocks document selections and writes that would depend on it.

The logical name key is scope, case-folded milestone, and name. An unset milestone is its own namespace. Format and schema do not distinguish names in the same namespace. For example, `pagination` can coexist in `topic:specs` at milestones `v1 launch/Specs` and `v2/Specs`. Check uniqueness under the board lock before create or a milestone update, including clearing the milestone. A collision exits 2 before changing the file or index. A milestone update changes the logical lookup key and retains the filename and UUID.

`doc locate --in <scope> --name <name> [--milestone <text>]` finds the exact logical name after checking read access to the scope. A supplied milestone matches the entire value case-insensitively after trimming. Supply `--milestone ""` to select an unassigned milestone explicitly. With the flag omitted, look across milestones in that scope and return the UUID for a single indexed match. Multiple matches exit 2 with an ambiguity error that asks for a milestone. Check file presence before returning the UUID and use the same unavailable error for a missing or inaccessible match. Output the UUID on one line or an object with an `id` field in JSON mode.

`milestone` and `concept` are optional strings on each index entry. Writers choose their own values subject to the field rules. Use `/` to separate hierarchy segments. A concept has 1–48 characters total and each segment uses ASCII letters, as in `Cursor/Pagination`. A milestone has 1–64 characters total and allows digits, spaces, and version punctuation under the field rules, as in `v1 launch/Specs`. The length includes separators. Every segment must be nonempty. These labels are stored as metadata and have no effect on the derived document path.

Set these fields with `--milestone` and `--concept` on `doc create` or `doc update` with write access. Trim leading and trailing whitespace from the whole flag value before checking it and preserve its case. Omitted flags leave existing values intact on update. An explicitly empty or whitespace-only value clears the field. The index and metadata output omit unset values.

`doc search` starts with all indexed documents for an admin. For other principals, keep only documents in scopes where the caller has at least one effective permission, including the implicit rights on its own channel. Apply this check before matching any search terms or preparing output, in every output format. A search for an inaccessible scope returns an empty successful result.

Combine supplied filters with AND. `--in` matches one exact logical scope, such as `topic:specs`, `agent:test`, or `skill:review`. `--milestone` and `--concept` match only entries with that field set. Trim outer whitespace from each filter. A term without an asterisk must be a valid field value and matches the entire value case-insensitively. A term with one trailing `*` matches values that begin with the text before it, also case-insensitively. The prefix must be a possible beginning of a valid label and may end at a slash. For example, `--concept 'Cursor/*'` matches `Cursor/Pagination`, and `--milestone 'v1 launch/*'` matches `v1 launch/Specs`. Quote the filter so the shell passes the asterisk literally. A lone `*` matches any set value. Blank terms and asterisks anywhere other than the single trailing position are usage errors.

`--query` matches a case-insensitive substring in the name, description, milestone, or concept. Search returns indexed metadata, including comments. `doc get` reads the authorized file on disk.

New entries default to an empty description, `draft` status, and an empty `reviewComments` object. A principal with `status` may set any lifecycle value. An `obsolete` document remains readable by authorized principals and is discouraged for new references. Mark it `archived` when it is ready for admin cleanup. A CLI content write or resync that records changed bytes resets `archived` to `draft`, unless an authorized CLI update supplies a status for the new bytes. Until resync records an external edit, prune checks the digest and skips the changed file.

`reviewComments` sits alongside the document's lifecycle `status` in each index entry. It maps canonical principal IDs such as `agent:specs` or `agent:test` to one current review per author. Each review contains a nonblank `comment`, a review `status`, and an optional document UUID in `locator`. Review statuses are `pass` for criteria met, `warning` for concerns to call out, and `fail` for unmet criteria. These outcomes are independent of the document lifecycle status.

`doc update --review-comment <text> --review-status <status>` inserts or replaces the caller's review. Supply `--review-locator <doc-id>` to include a link in that review. `--clear-review-comment` removes the caller's entire review and can be repeated safely. A review replacement and a clear are mutually exclusive. Both require the `status` permission and can be used independently of a lifecycle change. Author attribution comes from the resolved caller and is retained after grant revocation.

A review can link to any existing indexed document on the board that the reviewer can read. When returning review metadata, the CLI includes the locator only if the target is available to the reader. If the target is later deleted or the reader loses access, the index retains the stored locator and the CLI still returns the review text and outcome.

Administrators use `index audit` to report missing files, unindexed documents, digest changes, malformed or missing index, and unsupported or conflicting entries. `index resync` preserves metadata at surviving paths subject to the archived-to-draft rule. It drops records for missing files and reports removals and status resets. Retain the UUID in an unindexed canonical filename. For a manually added `<name>.<format>` or `<name>.<schema-id>.<format>` file in a registered scope, allocate a UUID and rename it to the canonical filename. Adopted entries default to `draft` with empty reviews and no milestone or concept.

Before resync writes anything, reject duplicate UUIDs, a known UUID at a different path, or a logical name collision. Report the conflicting entries so an admin can resolve them. Rename newly adopted files atomically before publishing the reconciled index. If an index write fails, the UUIDs in any renamed files let a retry reuse them. A missing index can be rebuilt with the filename UUIDs, but milestones and other metadata come from the index. If same-name files would collide in the unassigned namespace, recover the index from a snapshot before reconciling. Resync preserves malformed or unsupported indexes for operator recovery.

All CLI operations affecting the board use an OS-backed lock released on process exit. Mutations take an exclusive lock. Reads hold a compatible lock from authorization through IO so an ACL change cannot interleave with a CLI read. Keep lock bookkeeping under `index/` and write replacements atomically at the individual-file level. Write document bytes before the corresponding index entry, and report partial success if the index write fails. Audit and resync recover file/catalog consistency after interrupted updates. Direct editors can create drift and must finish writes before a snapshot or destructive command.

#### Document index example

```json
{
  "version": 1,
  "documents": [
    {
      "id": "b69ac8e7-6755-4249-83ef-211daa24ab6f",
      "scope": "agent:test",
      "name": "pagination-review",
      "format": "md",
      "path": "agents/test/b69ac8e7-6755-4249-83ef-211daa24ab6f--pagination-review.md",
      "description": "Test review of the cursor pagination spec",
      "status": "draft",
      "reviewComments": {},
      "sha256": "20af3722d897cdf3dcbdba27702361b45e6776517925456763adf2e68767e26d"
    },
    {
      "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
      "scope": "topic:specs",
      "name": "pagination",
      "format": "md",
      "path": "topics/specs/29b4dd81-af57-4c98-b3f0-ff0bf66da7bb--pagination.md",
      "description": "Requirements for cursor pagination",
      "milestone": "v1 launch/Specs",
      "concept": "Cursor/Pagination",
      "status": "approved",
      "reviewComments": {
        "agent:specs": {
          "comment": "Acceptance criteria cover the edge cases.",
          "status": "pass"
        },
        "agent:test": {
          "comment": "The test plan covers every criterion.",
          "status": "pass",
          "locator": "b69ac8e7-6755-4249-83ef-211daa24ab6f"
        }
      },
      "sha256": "bf706ffa8256beee042e4d36fd262ed2afed5c0408962814edcaddc83be4f44a"
    },
    {
      "id": "cf89eab1-3451-4e41-a6bb-0c6110a9d934",
      "scope": "topic:specs",
      "name": "pagination",
      "format": "md",
      "path": "topics/specs/cf89eab1-3451-4e41-a6bb-0c6110a9d934--pagination.md",
      "description": "Cursor pagination requirements for the v2 API",
      "milestone": "v2/Specs",
      "concept": "Cursor/Pagination",
      "status": "approved",
      "reviewComments": {},
      "sha256": "4ce5fb589b68ce9c3c8812d1b222e24ca3cb5926d5176425a8b7af3ae0c8b736"
    }
  ]
}
```

#### Internal locator and path resolution

The validated document index maps UUIDs to canonical board-relative paths in both directions. A UUID is stable and contains no location information. The filename carries the UUID so it can be recovered during resync. Perform lookups against the current index under the board lock.

**Locator to path.** Look up the UUID in the index, check the caller's permission for the requested operation on the entry's scope, then use the entry's validated path for root-confined filesystem access. Index validation checks that the path exactly matches the path derived from the entry's scope, UUID, name, format, and optional schema ID. Return the standard unavailable error for a missing or inaccessible document.

**Path to locator.** Validate the canonical path under a registered scope and look up the exact path in the index. Audit reports an absent entry as unindexed. Resync can read the UUID from the filename and reuse it when adopting that document.

In the index example, the lookup pairs UUID `b69ac8e7-6755-4249-83ef-211daa24ab6f` with `agents/test/b69ac8e7-6755-4249-83ef-211daa24ab6f--pagination-review.md`. Both conversions are internal storage functions. The public `doc locate` command looks up the logical name in the index and returns the UUID.

#### Search output formats

Use `doc search --format table` or `--format csv` for a compact summary. `--format json` is equivalent to `--json` and returns full document metadata. Specify one output selector per invocation. With neither selector, search prints the detailed text summary.

Both compact formats start with `id`, `name`, `scope`, and `description`, followed in order by `status`, `milestone`, and `concept` where applicable. Keep `scope` even when filtering with `--in`. Omit `status`, `milestone`, or `concept` when its own filter is an exact match. Keep the corresponding column for an unfiltered field or a trailing-asterisk filter. An unset field appears as an empty cell when its column is present. Choose the columns from the filters so the header stays the same even when the result is empty.

Cap each compact description at 60 Unicode code points including the suffix. For longer descriptions, keep the first 57 and append `...`. Use one row per document and preserve search result order. CSV writes a lowercase header and standard quoted records. Table output uses uppercase headings and aligned columns. Both write just the header when no documents match. JSON retains full descriptions and metadata regardless of applied filters.

#### Lifecycle statuses

| meaning | status |
| --- | --- |
| Work in progress or newly discovered file | draft |
| Current handoff note or reference | active |
| Work is waiting on a dependency or decision | blocked |
| Ready for human or agent review | review |
| Accepted in review | approved |
| Declined in review | rejected |
| Handoff or action completed | resolved |
| Temporarily retained for existing references and discouraged for new use | obsolete |
| Ready for administrator prune while the content is unchanged | archived |

### Internal JSON validation

#### Validating the index and access policy

Apply the field rules to the board-owned JSON files `index/documents.json` and `index/access.json`. Enforce the encoded size and nesting limits during parsing, reject duplicate member names before decoding into Go structs or maps, and validate the resulting objects with built-in JSON Schema Draft 2020-12. Run the Go checks for references, derived paths, unique IDs, logical name keys, grant pairs, and the required human administrator. [JSON Schema validation keywords](https://json-schema.org/draft/2020-12/json-schema-validation.html#name-validation-keywords-for-str) · [Go JSON decoding behavior](https://pkg.go.dev/encoding/json#hdr-Security_Considerations)

Validate the persisted access policy before trusting its grants and the persisted index before using its paths. Validate the prospective internal JSON before changing a document or replacing its metadata. Invalid metadata supplied via CLI flags exits 2 with a field-specific diagnostic and leaves files and index entries unchanged. Treat descriptive text and comments as data, escape control and layout characters when rendering them in terminal summaries, and keep stored paths confined to the board during filesystem access.

#### Field validation rules

| concept | document | field | presence | stage | type | validation |
| --- | --- | --- | --- | --- | --- | --- |
| board.validation.internal | both | $ | required | parser | object | Accept exactly one complete UTF-8 JSON object. Reject duplicate keys at every depth and trailing non-whitespace after the root. Reject unpaired Unicode surrogates in string escapes. Allow at most 8 object or array levels counting the root as level 1. |
| board.validation.internal | both | every object | applies when present | schema | object | Reject unknown properties and null values. Only the reviewComments object uses dynamic keys and each key must be a canonical principal ID. |
| board.validation.internal | both | slug | shared rule | schema and Go | string | 1–64 ASCII characters matching ^[a-z0-9][a-z0-9_-]{0,63}$ over the whole value. Reject platform-reserved names. |
| board.validation.internal | both | display text | shared rule | Go | string | Reject Unicode control characters, bidi controls, U+2028, and U+2029. Multiline display text also allows LF and tabs. |
| board.validation.internal.index | index/documents.json | $ | required | parser and writer | object | Limit to 32 MiB of encoded JSON including whitespace. |
| board.validation.internal.index | index/documents.json | version | required | schema and Go | integer | JSON number 1. Reject unsupported versions. |
| board.validation.internal.index | index/documents.json | documents | required | schema and Go | array of objects | Allow an empty list. Store entries in lexical path order. UUIDs and canonical paths are independently unique. Check name uniqueness across entries with the same scope and case-folded milestone. Omitted milestones share one namespace. |
| board.validation.internal.index | index/documents.json | documents[].id | required | schema | string | Exactly 36 ASCII characters. Canonical lowercase UUIDv4 with the RFC variant. |
| board.validation.internal.index | index/documents.json | documents[].scope | required | schema and Go | string | Prefix agent: or skill: or topic: followed by a slug. The scope must be registered in the access policy. |
| board.validation.internal.index | index/documents.json | documents[].name | required | schema and Go | string | Slug. Reject platform-reserved names. |
| board.validation.internal.index | index/documents.json | documents[].format | required | schema | string | One of json or csv or md. |
| board.validation.internal.index | index/documents.json | documents[].schemaId | optional | schema and Go | string | Slug. Reject platform-reserved names. A missing schema is reported by document validation. |
| board.validation.internal.index | index/documents.json | documents[].path | required | schema and Go | string | At most 244 ASCII characters. Must exactly equal the board-relative path derived from scope, UUID, name, format, and optional schema ID. Resolve filesystem access inside the board root and reject symlinks. |
| board.validation.internal.index | index/documents.json | documents[].description | required | schema and Go | string | 0–500 Unicode code points. Single-line display text. |
| board.validation.internal.index | index/documents.json | documents[].milestone | optional | schema and Go | string | 1–64 ASCII characters total including slash separators. Full-string pattern ^[A-Za-z0-9][A-Za-z0-9 ._-]*(/[A-Za-z0-9][A-Za-z0-9 ._-]*)*$. Reject trailing whitespace in each segment. |
| board.validation.internal.index | index/documents.json | documents[].concept | optional | schema | string | 1–48 ASCII characters total including slash separators. Each segment contains only letters. Full-string pattern ^[A-Za-z]+(/[A-Za-z]+)*$. |
| board.validation.internal.index | index/documents.json | documents[].status | required | schema | string | One of the lifecycle states in lifecycle.csv. |
| board.validation.internal.index | index/documents.json | documents[].sha256 | required | schema and audit | string | Exactly 64 lowercase hexadecimal characters. Audit compares it to the digest of the stored bytes on disk. |
| board.validation.internal.index | index/documents.json | documents[].reviewComments | required | schema and Go | object | Allow an empty object. Each key is a registered principal ID. Each value is one review object. |
| board.validation.internal.index | index/documents.json | documents[].reviewComments.*.comment | required | schema and Go | string | 1–2000 Unicode code points. Must contain non-whitespace text. Multiline display text. |
| board.validation.internal.index | index/documents.json | documents[].reviewComments.*.status | required | schema | string | One of pass or warning or fail. |
| board.validation.internal.index | index/documents.json | documents[].reviewComments.*.locator | optional | schema and Go | string | Canonical lowercase UUIDv4 with the RFC variant. Check the target exists and is readable when assigning it. A stored reference can outlive its target. |
| board.validation.internal.access | index/access.json | $ | required | parser and writer | object | Limit to 4 MiB of encoded JSON including whitespace. |
| board.validation.internal.access | index/access.json | version | required | schema and Go | integer | JSON number 1. Reject unsupported versions. |
| board.validation.internal.access | index/access.json | principals | required | schema and Go | array of objects | Require at least one human admin. Principal IDs must be unique. |
| board.validation.internal.access | index/access.json | principals[].id | required | schema and Go | string | Prefix human: or agent: or skill: followed by a slug. Reject platform-reserved slug names. |
| board.validation.internal.access | index/access.json | principals[].role | optional | schema and Go | string | Either member or admin. Omitted defaults to member. Only a human principal can have the admin role. |
| board.validation.internal.access | index/access.json | topics | required | schema and Go | array of strings | Allow an empty list. Each topic is a unique slug. Reject platform-reserved names. |
| board.validation.internal.access | index/access.json | grants | required | schema and Go | array of objects | Allow an empty list. Require at most one entry per scope and principal pair. |
| board.validation.internal.access | index/access.json | grants[].scope | required | schema and Go | string | Prefix agent: or skill: or topic: followed by a slug. Must refer to an existing private owner or registered topic. |
| board.validation.internal.access | index/access.json | grants[].principal | required | schema and Go | string | Canonical principal ID. Must refer to a registered principal. |
| board.validation.internal.access | index/access.json | grants[].permissions | required | schema | array of strings | 1–4 distinct values. A topic allows read or write or status or delete. A private scope allows read or status. |

### Retention

#### Prune and archive

An admin uses `index prune` to preview archived documents whose recorded digest still matches the disk file. The preview reports skipped archived entries with missing or changed files and emits a deterministic plan ID derived from the sorted candidate IDs, paths and digests. Apply requires `--apply --expect <plan-id>`. Recompute the plan under the board lock and reject stale input before deleting. Recheck each candidate immediately before unlinking. If any unlink fails, stop, reconcile entries for files successfully removed, and report completed and remaining IDs. Report reconciliation failures so `index audit` can identify the drift. Prune eligibility depends on archived status and matching bytes.

`snapshot create` writes the ZIP to stdout and can archive a board with index drift or invalid documents. ZIP entries start at `.gh-corkboard/` and include the five directories and their stored files, including the access policy. Exclude CLI lock bookkeeping and temporary files. Fail on unsafe entries or output errors. `--expect <plan-id>` compares the current prune plan before the CLI emits bytes. For a pre-prune archive, save the ZIP outside the board and apply the same plan only after the snapshot command succeeds. Prune checks the plan again, so changes to candidates between commands invalidate it. Discard incomplete output after an archive failure.

#### Snapshot and prune example

```bash
#!/usr/bin/env bash

# Source this file and run example_preview_prune. Review the candidates and planId.
example_preview_prune() {
  gh corkboard --as human:owner index prune --json
}

# Call with the reviewed plan ID and a new ZIP filename outside .gh-corkboard.
# For example: example_archive_and_prune "$plan_id" ../corkboard-before-prune.zip
example_archive_and_prune() (
  plan_id=${1:?Supply the reviewed prune plan ID}
  archive_path=${2:?Supply a ZIP filename outside the board}
  set -o noclobber

  gh corkboard --as human:owner snapshot create --expect "$plan_id" > "$archive_path" &&
    gh corkboard --as human:owner index prune --apply --expect "$plan_id"
)
```

### Schema registry and projections

#### Data and diagnostic shapes

```ts
type Status =
  | "draft"
  | "active"
  | "blocked"
  | "review"
  | "approved"
  | "rejected"
  | "resolved"
  | "obsolete"
  | "archived";
type Format = "json" | "csv" | "md";
type PrincipalId = `${"human" | "agent" | "skill"}:${string}`;
type Scope = `${"agent" | "skill" | "topic"}:${string}`;
type Permission = "read" | "write" | "delete" | "status";
type DocumentId = string; // Canonical lowercase UUIDv4
type ReviewStatus = "pass" | "warning" | "fail";

interface AccessPolicy {
  version: 1;
  principals: Principal[];
  topics: string[];
  grants: ScopeGrant[];
}

interface Principal {
  id: PrincipalId;
  role?: "member" | "admin"; // Defaults to member
}

interface ScopeGrant {
  scope: Scope;
  principal: PrincipalId;
  permissions: Permission[];
}

interface ReviewComment {
  comment: string;
  status: ReviewStatus;
  locator?: DocumentId;
}

interface DocumentMetadata {
  id: DocumentId;
  scope: Scope;
  name: string;
  format: Format;
  schemaId?: string;
  description: string;
  milestone?: string;
  concept?: string;
  status: Status;
  reviewComments: Partial<Record<PrincipalId, ReviewComment>>;
}

interface DocumentUpdateResult extends DocumentMetadata {
  formattingChanged?: boolean; // Present for stdin replacements and text patches
}

interface Index {
  version: 1;
  documents: IndexEntry[];
}

interface IndexEntry extends DocumentMetadata {
  path: string;
  sha256: string;
}

interface CsvProjection {
  columns: string[];
  rows: Record<string, string>[];
}

interface MarkdownProjection {
  headings: { level: 1 | 2 | 3 | 4 | 5 | 6; text: string }[];
}

interface Diagnostic {
  code: string;
  message: string;
  instancePath?: string; // JSON Pointer into the original JSON or projection
  line?: number; // One-based coordinates when available
  column?: number;
}

interface ValidationResult {
  id: DocumentId;
  format: Format;
  schemaId?: string;
  state: "valid" | "invalid" | "schemaless";
  diagnostics: Diagnostic[];
}
```

#### Validation rules

Use JSON Schema Draft 2020-12 directly. A registry file is `schemas/<format>/<id>.schema.json`. Schema writes compile the definition and reject unsupported dialects and unresolved references. V1 accepts same-resource fragment references and uses bundled Draft 2020-12 meta-schema resources. Fetching schema references from the filesystem or network is disabled. Apply standard annotation semantics for `format`. A schema update changes subsequent validation for documents bound to that ID. Authors can register a new ID such as `spec-v2` when both contracts must coexist. Schema deletion checks bound filenames on disk, including files absent from the index. [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)

The CSV projection supports required columns, header order, row count and string constraints. For example, a source file with the header `id,priority` and the row `pagination,01` projects to the JSON shown in the CSV projection example. Typed numeric, date and relational constraints would need a future profile. CSV on the Web provides a standardized metadata vocabulary if that interoperability becomes a requirement. [W3C CSV on the Web](https://www.w3.org/TR/tabular-metadata/)

The Markdown projection supports required headings, order and multiplicity. For example, `## Acceptance **criteria**` yields the heading shown in the Markdown projection example. A schema can check for that entry with `contains` or enforce a sequence with `prefixItems`. Markdown prose and section body content remain outside the v1 schema contract. [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/)

Well-formed documents remain writable even when schema validation would fail. `doc validate` parses the native format and evaluates any declared schema. It returns `invalid` for syntax or projection errors, a missing declared schema, or schema violations. It returns `schemaless` for well-formed documents without a declared schema and `valid` for documents that satisfy their declared schema. Report the instance path and source coordinates where available. CommonMark accepts free-form text. Schemaless Markdown therefore only requires decodable UTF-8. The index and access policy use built-in schemas and additional Go checks for uniqueness and references. User schema operations cannot alter those contracts.

CUE supports composition and constraints across fields, and exposes validation and JSON Schema conversion through its Go API. Both engines would consume the same CSV and Markdown projections. Keeping one native JSON Schema engine gives authors reusable JSON artifacts for documents and the index. Revisit CUE if real board schemas need shared constraints or cross-field relationships that become hard to express in JSON Schema. [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)

#### Validation inputs by format

| format | validation input |
| --- | --- |
| JSON | Parse exactly one UTF-8 JSON value and reject duplicate object keys. Preserve numeric precision through validation. |
| CSV | Parse UTF-8 comma-separated records with standard quoting and a mandatory header. Use the header verbatim as ordered `columns` and each data record as an object in `rows`. Header names must be nonempty and unique. Reject records whose field count differs from the header. Preserve data cell strings, including empty strings and whitespace. Skip physically empty lines outside quoted fields. |
| Markdown | Parse UTF-8 as CommonMark 0.31.2. Collect document-level headings in source order into `headings`, with parsed level and visible text from inline nodes. Strip markup and link destinations, include image alt text and inline code, turn inline breaks into spaces, trim edges and collapse whitespace runs. Headings inside containers such as quotes and lists stay outside this projection. |

#### CSV projection example

```json
{
  "columns": [
    "id",
    "priority"
  ],
  "rows": [
    {
      "id": "pagination",
      "priority": "01"
    }
  ]
}
```

#### Markdown projection example

```json
{
  "headings": [
    {
      "level": 2,
      "text": "Acceptance criteria"
    }
  ]
}
```


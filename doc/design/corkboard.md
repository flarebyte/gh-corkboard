# gh-corkboard v1 specification

Proposed specification for a local project corkboard and messaging system.

## Access control

### Principals and permissions

#### Identity and access policy

Every board command requires a global `--as <principal>` selector before the command. Principals are `human:<id>`, `agent:<id>`, or `skill:<id>`. The CLI checks registration and resolves the principal's role through `index/access.json`. Help and version output are available without a principal. On first initialization, the supplied human becomes the first administrator. On an existing board, `init` requires an administrator. A missing or damaged policy on an existing board blocks access and must be recovered by the operator. A repeated `init` cannot claim that board.

The policy is plaintext JSON validated against a built-in version-1 schema. Each principal entry has an ID and an optional role. After schema validation, the CLI resolves an omitted `role` to `member`. An explicit role must be valid and non-null. Check unique principal and topic IDs, one grant per scope and principal with a nonempty set of distinct permissions, registered grant subjects and scopes, and at least one administrator after applying defaults. Only a human can hold `admin`.

`read` allows discovery, retrieval, and validation. `write` allows creating documents and changing their content or description. A topic `delete` grant allows deleting individual documents in that topic. `status` allows setting any lifecycle status and setting or clearing the caller's own review. Each of `write`, `delete`, and `status` implies `read` and can be granted independently where the scope permits it. Board admins have all four permissions in every scope and run board-wide prune. Only a board admin can delete a document from an agent or skill channel.

Folder removal requires the board `admin` role. `doc delete` retains the folder even when deleting its last document. `access topic delete` removes the topic folder and its grants only when the folder is empty and the scope has no remaining index entries.

Only a principal with the `admin` role can add or revoke grants in any scope. An admin can also manage principals, create topics, remove empty topics, read and write any document, mutate schemas, and run board-wide audit, resync, prune, and snapshots. With no scope filter, `access list` shows all grants for an admin. A private scope owner can inspect grants by selecting that scope.

All registered principals can read the shared schema registry. `access whoami` returns the selected principal's role and effective permissions. `access grant` adds one permission and `access revoke` removes one. Both are idempotent. Remove the grant record when its permission set becomes empty. Revoking an explicit `read` grant leaves the read access implied by `write`, `delete`, or `status` in place. Owners and admins retain their implicit rights. Agent and skill owners get read and write in their own channels. Private scopes allow explicit `read` and `status` grants, so an owner needs a separate `status` grant to review or change status. Reject write and delete grants for private scopes in both CLI commands and policy validation. Reject changes that would leave the board without an admin.

Check the current ACL under the board lock before file access. Apply the read check before returning search or bulk-validation results, descriptions, or other metadata. A syntactically valid UUID yields the same unavailable error for missing and inaccessible documents. Policy errors fail closed. Reuse the same checks for humans, agents, and skills.

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
| Document author | board.validation | draft | The document remains editable and the validator reports actionable diagnostics | Save an incomplete schema-bound document and run validation |
| Board administrator | board.retention | retire | The selected files are removed and the prior board can be inspected offline | Preview archived documents then save a ZIP snapshot before applying cleanup |

### Workspace

#### Workspace and handoffs

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

Every indexed document has a UUIDv4 locator, including documents adopted by resync. Use that ID to get, modify, delete, validate, or reference a document. Create and index search return the locator as the document's ID. `doc locate` resolves an exact scope, name, format, and optional schema ID to the same UUID. Omitting the schema matches a schemaless document. The command checks read access and file presence and uses the same unavailable error for missing and inaccessible documents. It prints the UUID on its own line or an object with an `id` field with `--json`. IDs stay stable through content and metadata updates. Authorization comes from the current principal and scope policy. For a handoff, post in a shared topic and include the referenced document's UUID. The recipient needs read access to both the handoff and the referenced document. Self-notes go in the writer's own scope. [UUIDv4 specification](https://www.rfc-editor.org/rfc/rfc9562.html#name-uuid-version-4)

## Command interface

### Commands

#### Command reference

| access | command | concept | contract |
| --- | --- | --- | --- |
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
| Scope write plus status when supplied | `doc create --in <scope> --name <name> --format <format> [--schema <schema-id>] [--description <text>] [--status <status>]` | board.documents | Read bytes from stdin and return the new document UUID |
| Scope read | `doc get <doc-id>` | board.documents | Write stored document bytes to stdout |
| Scope read | `doc locate --in <scope> --name <name> --format <format> [--schema <schema-id>]` | board.documents | Return the UUID for one exact logical document name |
| Scope write for content or description and scope status for lifecycle or own review | `doc update <doc-id> [--stdin] [--description <text>] [--status <status>] [--review-comment <text> --review-status <pass\|warning\|fail> [--review-locator <doc-id>]\|--clear-review-comment]` | board.documents | Apply the supplied changes to the document and index |
| Topic delete or board admin | `doc delete <doc-id>` | board.documents | Delete one document and remove its index entry while retaining the folder |
| Scope read | `doc validate <doc-id\|--all>` | board.validation | Validate the selected document or all readable indexed documents |
| Registered principal | `index search [--in <scope>] [--status <status>] [--query <text>]` | board.index | Search metadata of readable documents and return logical identifiers |
| Admin | `index audit` | board.index | Report filesystem and index discrepancies without changing board data |
| Admin | `index resync` | board.index | Reconcile the catalog with valid managed locations and preserve surviving metadata |
| Admin | `index prune [--apply --expect <plan-id>]` | board.retention | Preview or apply removal of archived documents |
| Admin | `snapshot create [--expect <plan-id>]` | board.retention | Write a ZIP of the board to stdout and optionally verify a prune plan first |

#### Command conventions

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
gh corkboard --as human:owner access grant agent:test --to agent:implementation --permission read
gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --description "Requirements for cursor pagination" --status active --json < pagination.md
doc_id=$(gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination --format md)
gh corkboard --as agent:implementation doc get "$doc_id"
gh corkboard --as agent:specs doc update "$doc_id" --status review --review-status pass --review-comment "Acceptance criteria cover the edge cases."
gh corkboard --as agent:test doc create --in agent:test --name pagination-review --format md --description "Test review of the cursor pagination spec" < pagination-review.md
review_id=$(gh corkboard --as agent:test doc locate --in agent:test --name pagination-review --format md)
gh corkboard --as agent:test doc update "$doc_id" --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator "$review_id"
gh corkboard --as agent:implementation index search --in topic:specs --status approved --json
gh corkboard --as agent:implementation doc get "$review_id"
gh corkboard --as agent:implementation doc create --in agent:implementation --name follow-up --format md < follow-up.md
```

Exercise scoped create and denied cross-scope mutations. Verify that an admin can grant topic access and issue private read and status grants. Verify that a member cannot change grants, including in its own scope. Verify that a writer cannot explicitly set a status without a status grant, a reviewer with only status can set any status and update only its own review, and a combined update fails before any changes if the caller lacks a required permission. Check that locate returns the same ID as create and search, that linked reviews can span scopes, and that inaccessible targets are rejected when linking. Verify that agent and skill owners can edit their own documents and are denied deletion, even if they hold delete access in a topic. Verify that topic delete leaves the folder intact and that a board admin can remove an empty topic with no index entries. Also exercise read filtering, malformed policy and index data, symlink rejection, invalid drafts, schema changes, concurrent updates, archived documents edited after preview, and snapshot failures. Build the packaged binary using GitHub CLI naming conventions. [GitHub CLI extension guide](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions)

## Specification sources

### Concept registry

#### Concept registry

| description | id | parent |
| --- | --- | --- |
| Local project corkboard | board |  |
| Principals and scope permissions | board.access | board |
| Go extension command interface | board.cli | board |
| Agent handoffs self-notes and human review | board.communication | board |
| Document formats names and lifecycle | board.documents | board |
| Metadata search and filesystem reconciliation | board.index | board |
| Obsolete documents and snapshots | board.retention | board |
| Schema registry and validation profiles | board.validation | board |

## Storage and validation

### Index and lifecycle

#### Catalog and reconciliation

`index/documents.json` is one catalog for the whole board. Its top-level object has `version: 1` and a `documents` array ordered by canonical path. Each entry stores logical document metadata, its derived board-relative `path`, and the lowercase hexadecimal SHA-256 digest of the indexed bytes. A built-in schema validates structure, the UUID form, status, review comments, and digest. Go checks uniqueness of IDs and paths, that the scope and comment authors are registered, and that the stored path exactly matches the derived path. A malformed index blocks document selections and writes that would depend on it.

`index search` applies ACL filtering first and combines supplied filters with AND. `--in` matches one logical scope. The query is a case-insensitive substring of the name or description. Search returns indexed metadata, including comments. `doc get` reads the authorized file on disk. New entries default to an empty description, `draft` status, and an empty `reviewComments` object. A principal with `status` may set any lifecycle value. An `obsolete` document remains readable by authorized principals and is discouraged for new references. Mark it `archived` when it is ready for admin cleanup. When CLI or external content changes, a prior `archived` status reverts to `draft` unless the CLI update explicitly sets a status for the new bytes.

`reviewComments` sits alongside the document's lifecycle `status` in each index entry. It maps canonical principal IDs such as `agent:specs` or `agent:test` to one current review per author. Each review contains a nonempty `comment`, a review `status`, and an optional document UUID in `locator`. Review statuses are `pass` for criteria met, `warning` for concerns to call out, and `fail` for unmet criteria. These outcomes are independent of the document lifecycle status.

`doc update --review-comment <text> --review-status <status>` inserts or replaces the caller's review. Supply `--review-locator <doc-id>` to include a link in that review. `--clear-review-comment` removes the caller's entire review and can be repeated safely. A review replacement and a clear are mutually exclusive. Both require the `status` permission and can be used independently of a lifecycle change. Author attribution comes from the resolved caller and is retained after grant revocation.

A review can link to any existing indexed document on the board that the reviewer can read. When returning review metadata, the CLI includes the locator only if the target is available to the reader. If the target is later deleted or the reader loses access, the index retains the stored locator and the CLI still returns the review text and outcome.

Administrators use `index audit` to report missing files, unindexed documents, digest changes, malformed or missing index, and unsupported entries. `index resync` adopts files only in registered managed scopes and assigns new UUIDs, draft status, and empty review comments. It preserves UUIDs, descriptions, and comments at surviving paths and preserves status except when bytes change for an archived document. It drops records for missing files and reports those removals and status resets. External renames appear as a missing path and a new file. Resync can rebuild a missing catalog with new IDs. It refuses to overwrite a malformed or unsupported index so its metadata remains recoverable. Use `doc validate --all` to check content in readable indexed documents.

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
      "path": "agents/test/pagination-review.md",
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
      "path": "topics/specs/pagination.md",
      "description": "Requirements for cursor pagination",
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
    }
  ]
}
```

#### Internal locator and path resolution

The validated document index is the source of the mapping in both directions. Use its unique UUID and canonical board-relative path as lookup keys. A UUID is assigned at creation or resync adoption and carries no path information. Perform lookups against the current index under the board lock.

**Locator to path.** Look up the UUID in the index, check the caller's permission for the requested operation on the entry's scope, then use the entry's validated path for root-confined filesystem access. Index validation has already checked that the path exactly matches the path derived from the entry's scope, name, format, and optional schema ID. Return the standard unavailable error for a missing or inaccessible document.

**Path to locator.** Use a canonical path derived from validated logical fields or discovered under the board root during audit or resync. Check that it fits a registered scope and the managed filename rules, then look up that exact path in the index and use the entry's UUID. A path without an entry has no locator yet. Audit reports it as unindexed, and resync can adopt it and assign a new UUID.

In the index example, the lookup pairs UUID `b69ac8e7-6755-4249-83ef-211daa24ab6f` with `agents/test/pagination-review.md`. Both conversions are internal storage functions. The public `doc locate` command takes logical fields and returns the UUID. It derives the canonical path and performs the reverse lookup internally, with read authorization and a file-presence check.

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

### Retention

#### Prune and archive

An admin uses `index prune` to preview archived documents whose recorded digest still matches the disk file. The preview reports skipped archived entries with missing or changed files and emits a deterministic plan ID derived from the sorted candidate IDs, paths and digests. Apply requires `--apply --expect <plan-id>`. Recompute the plan under the board lock and reject stale input before deleting. Recheck each candidate immediately before unlinking. If any unlink fails, stop, reconcile entries for files successfully removed, and report completed and remaining IDs. Report reconciliation failures so `index audit` can identify the drift. Prune eligibility depends on archived status and matching bytes.

`snapshot create` writes the ZIP to stdout and can archive a board with index drift or invalid documents. ZIP entries start at `.gh-corkboard/` and include the five directories and their stored files, including the access policy. Exclude CLI lock bookkeeping and temporary files. Fail on unsafe entries or output errors. `--expect <plan-id>` compares the current prune plan before the CLI emits bytes. For a pre-prune archive, save the ZIP outside the board and apply the same plan only after the snapshot command succeeds. Prune checks the plan again, so changes to candidates between commands invalidate it. Discard incomplete output after an archive failure.

```sh
gh corkboard --as human:owner index prune --json
# Use the plan ID from the preview.
gh corkboard --as human:owner snapshot create --expect "$plan_id" > ../corkboard-before-prune.zip &&
  gh corkboard --as human:owner index prune --apply --expect "$plan_id"
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
  status: Status;
  reviewComments: Partial<Record<PrincipalId, ReviewComment>>;
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

Every document remains writable even when validation would fail. On explicit validation, parse the native format first, report syntax or projection errors, then evaluate the declared schema when present. A well-formed plain document returns `schemaless`. Bound documents return `valid` or `invalid`. Report the instance path and source coordinates where available. CommonMark accepts free-form text. Schemaless Markdown therefore only requires decodable UTF-8. The index and access policy use built-in schemas and additional Go checks for uniqueness and references. User schema operations cannot alter those contracts.

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


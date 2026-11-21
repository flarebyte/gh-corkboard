# CLI output examples

These examples specify the proposed output for each command. Each example assumes the board state needed for that operation. The examples are independent.

For structured commands, the shell transcript shows default stdout. The following JSON block shows stdout for the same operation with `--json`. Add the flag before any shell redirection. Commands exit 0 unless an example specifies a different outcome.

[Setup and access](#board-setup-and-access) · [Schemas](#schemas) · [Documents](#documents) · [Index and snapshots](#index-and-snapshots) · [Diagnostics and exit codes](#diagnostics-and-exit-codes)

## Board setup and access

### `init`

```console
$ gh corkboard --as human:owner init
Initialized .gh-corkboard. The root ignore rule is already present.
```

```json
{
  "initialized": true,
  "ignoreRuleAdded": false
}
```

### `access whoami`

```console
$ gh corkboard --as agent:implementation access whoami
agent:implementation (member)
agent:implementation  read, write
agent:test            read
topic:specs           read
```

```json
{
  "principal": "agent:implementation",
  "role": "member",
  "scopes": [
    {
      "scope": "agent:implementation",
      "permissions": [
        "read",
        "write"
      ]
    },
    {
      "scope": "agent:test",
      "permissions": [
        "read"
      ]
    },
    {
      "scope": "topic:specs",
      "permissions": [
        "read"
      ]
    }
  ]
}
```

### `access principal add`

```console
$ gh corkboard --as human:owner access principal add agent:test
Registered agent:test (member).
```

```json
{
  "principal": "agent:test",
  "role": "member"
}
```

### `access principal list`

```console
$ gh corkboard --as human:owner access principal list
agent:implementation  member
agent:specs           member
agent:test            member
human:owner           admin
```

```json
{
  "principals": [
    {
      "id": "agent:implementation",
      "role": "member"
    },
    {
      "id": "agent:specs",
      "role": "member"
    },
    {
      "id": "agent:test",
      "role": "member"
    },
    {
      "id": "human:owner",
      "role": "admin"
    }
  ]
}
```

### `access principal set`

```console
$ gh corkboard --as human:owner access principal set human:reviewer --role admin
Changed human:reviewer to admin.
```

```json
{
  "principal": "human:reviewer",
  "role": "admin",
  "changed": true
}
```

### `access topic create`

```console
$ gh corkboard --as human:owner access topic create scratch
Created topic:scratch.
```

```json
{
  "scope": "topic:scratch",
  "created": true
}
```

### `access topic delete`

```console
$ gh corkboard --as human:owner access topic delete scratch
Removed the empty topic:scratch folder and its grants.
```

```json
{
  "scope": "topic:scratch",
  "deleted": true
}
```

### `access list`

```console
$ gh corkboard --as human:owner access list --scope topic:specs
topic:specs  agent:implementation  read
topic:specs  agent:specs           write, status
topic:specs  agent:test            status
```

```json
{
  "grants": [
    {
      "scope": "topic:specs",
      "principal": "agent:implementation",
      "permissions": [
        "read"
      ]
    },
    {
      "scope": "topic:specs",
      "principal": "agent:specs",
      "permissions": [
        "write",
        "status"
      ]
    },
    {
      "scope": "topic:specs",
      "principal": "agent:test",
      "permissions": [
        "status"
      ]
    }
  ]
}
```

### `access grant`

```console
$ gh corkboard --as human:owner access grant topic:specs --to agent:implementation --permission status
Granted status on topic:specs to agent:implementation.
```

```json
{
  "scope": "topic:specs",
  "principal": "agent:implementation",
  "permission": "status",
  "changed": true
}
```

### `access revoke`

```console
$ gh corkboard --as human:owner access revoke topic:specs --from agent:implementation --permission status
Revoked status on topic:specs from agent:implementation.
```

```json
{
  "scope": "topic:specs",
  "principal": "agent:implementation",
  "permission": "status",
  "changed": true
}
```

## Schemas

### `schema create`

```console
$ gh corkboard --as human:owner schema create json spec < spec.schema.json
Registered the json schema spec.
```

```json
{
  "format": "json",
  "schemaId": "spec",
  "created": true
}
```

### `schema get`

Stdout is the stored schema. The JSON below is the file content.

```sh
gh corkboard --as agent:implementation schema get json spec
```

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "title": {
      "type": "string"
    }
  },
  "required": [
    "title"
  ],
  "type": "object"
}
```

### `schema list`

```console
$ gh corkboard --as agent:implementation schema list --format json
json  spec
```

```json
{
  "schemas": [
    {
      "format": "json",
      "schemaId": "spec"
    }
  ]
}
```

### `schema update`

```console
$ gh corkboard --as human:owner schema update json spec < spec-v2.schema.json
Updated the json schema spec.
```

```json
{
  "format": "json",
  "schemaId": "spec",
  "updated": true
}
```

### `schema delete`

```console
$ gh corkboard --as human:owner schema delete json spec
Deleted the unused json schema spec.
```

```json
{
  "format": "json",
  "schemaId": "spec",
  "deleted": true
}
```

## Documents

### `doc create`

```console
$ gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --description "Requirements for cursor pagination" --status active < pagination.md
Created 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb (topic:specs / pagination, md, active).
```

```json
{
  "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
  "scope": "topic:specs",
  "name": "pagination",
  "format": "md",
  "description": "Requirements for cursor pagination",
  "status": "active",
  "reviewComments": {}
}
```

### `doc get`

Stdout contains exactly the stored Markdown bytes.

```sh
gh corkboard --as agent:implementation doc get 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
```

```markdown
# Pagination

## Acceptance criteria

- Return the next cursor when more results are available.
```

### `doc locate`

```console
$ gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination --format md
29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
```

```json
{
  "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb"
}
```

### `doc update`

```console
$ gh corkboard --as agent:test doc update 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator b69ac8e7-6755-4249-83ef-211daa24ab6f
Updated 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb. Status: approved. Review by agent:test: pass.
```

```json
{
  "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
  "scope": "topic:specs",
  "name": "pagination",
  "format": "md",
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
  }
}
```

### `doc delete`

```console
$ gh corkboard --as human:owner doc delete b69ac8e7-6755-4249-83ef-211daa24ab6f
Deleted document b69ac8e7-6755-4249-83ef-211daa24ab6f.
```

```json
{
  "id": "b69ac8e7-6755-4249-83ef-211daa24ab6f",
  "deleted": true
}
```

### `doc validate`

```console
$ gh corkboard --as agent:implementation doc validate 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  schemaless
```

```json
{
  "results": [
    {
      "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
      "format": "md",
      "state": "schemaless",
      "diagnostics": []
    }
  ]
}
```

The `--all` variant uses the same result shape and includes readable indexed documents only.

## Index and snapshots

### `index search`

```console
$ gh corkboard --as agent:implementation index search --in topic:specs --status approved
29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  approved  topic:specs / pagination (md)
  Requirements for cursor pagination
  agent:specs [pass] Acceptance criteria cover the edge cases.
  agent:test [pass] The test plan covers every criterion.
    Review document: b69ac8e7-6755-4249-83ef-211daa24ab6f
```

```json
{
  "documents": [
    {
      "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
      "scope": "topic:specs",
      "name": "pagination",
      "format": "md",
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
      }
    }
  ]
}
```

### `index audit`

```console
$ gh corkboard --as human:owner index audit
Index is in sync.
```

```json
{
  "findings": []
}
```

### `index resync`

```console
$ gh corkboard --as human:owner index resync
Indexed 1 new document as draft.
7ee9c97c-dcbb-4355-898a-2fbb18868362  agents/test/local-results.csv
```

```json
{
  "added": [
    {
      "id": "7ee9c97c-dcbb-4355-898a-2fbb18868362",
      "path": "agents/test/local-results.csv",
      "status": "draft"
    }
  ],
  "removed": [],
  "updated": [],
  "resetToDraft": []
}
```

### `index prune`

In this example the pagination document has been marked archived and its recorded digest matches the file.

```console
$ gh corkboard --as human:owner index prune
Plan c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3
1 archived document is eligible for deletion.
29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  topics/specs/pagination.md
```

```json
{
  "planId": "c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3",
  "candidates": [
    {
      "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
      "path": "topics/specs/pagination.md"
    }
  ],
  "skipped": []
}
```

For the same plan, apply mode reports the documents actually removed.

```console
$ gh corkboard --as human:owner index prune --apply --expect c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3
Deleted 1 document.
29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
```

```json
{
  "planId": "c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3",
  "deleted": [
    "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb"
  ],
  "remaining": []
}
```

### `snapshot create`

Stdout is a binary ZIP stream. The first command saves the stream. The second inspects the archive and produces the listing shown here.

```console
$ gh corkboard --as human:owner snapshot create > corkboard.zip
$ unzip -Z1 corkboard.zip
.gh-corkboard/
.gh-corkboard/agents/
.gh-corkboard/agents/implementation/
.gh-corkboard/agents/specs/
.gh-corkboard/agents/test/
.gh-corkboard/agents/test/pagination-review.md
.gh-corkboard/index/
.gh-corkboard/index/access.json
.gh-corkboard/index/documents.json
.gh-corkboard/schemas/
.gh-corkboard/schemas/csv/
.gh-corkboard/schemas/json/
.gh-corkboard/schemas/md/
.gh-corkboard/skills/
.gh-corkboard/topics/
.gh-corkboard/topics/specs/
.gh-corkboard/topics/specs/pagination.md
```

## Diagnostics and exit codes

### Schema validation finding

A schema violation exits 1. The diagnostic report goes to stdout in both text and JSON modes.

```console
$ gh corkboard --as agent:implementation doc validate ae3f039f-33f8-4906-99e5-43e0f21095c2
ae3f039f-33f8-4906-99e5-43e0f21095c2  invalid
  Required property "title" is missing.
```

```json
{
  "results": [
    {
      "id": "ae3f039f-33f8-4906-99e5-43e0f21095c2",
      "format": "json",
      "schemaId": "spec",
      "state": "invalid",
      "diagnostics": [
        {
          "code": "schema_required",
          "message": "Required property \"title\" is missing.",
          "instancePath": ""
        }
      ]
    }
  ]
}
```

### Index audit finding

Index drift exits 1. The diagnostic report goes to stdout.

```console
$ gh corkboard --as human:owner index audit
Unindexed document: agents/test/local-results.csv
```

```json
{
  "findings": [
    {
      "code": "unindexed_document",
      "path": "agents/test/local-results.csv"
    }
  ]
}
```

### Malformed input on write

Malformed input exits 1, leaves the file and index untouched, and reports a diagnostic on stdout.

```console
$ printf '%s' '{"title":' | gh corkboard --as agent:specs doc create --in topic:specs --name broken --format json
Invalid JSON: unexpected end of input.
```

```json
{
  "diagnostics": [
    {
      "code": "invalid_json",
      "message": "Unexpected end of input."
    }
  ]
}
```

### Unavailable document

A missing or inaccessible UUID exits 2. The error goes to stderr and stdout is empty.

```console
$ gh corkboard --as agent:implementation doc get 2a0e27a3-9a98-43b5-a1a9-a6079c93fcd2
error: document unavailable
```

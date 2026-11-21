# CLI output examples

Each Bash function shows an independent command example and its expected output. Success examples exit 0. In the JSON files, jsonStdout maps commands to the response with --json, and rawStdout contains stored JSON emitted directly. For commands with input redirection, place --json before the redirection.

## Board setup and access

### Default terminal output

#### access.sh

```bash
#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# init
example_init() {
  gh corkboard --as human:owner init
  # Expected stdout
  # Initialized .gh-corkboard. The root ignore rule is already present.
}

# access whoami
example_access_whoami() {
  gh corkboard --as agent:implementation access whoami
  # Expected stdout
  # agent:implementation (member)
  # agent:implementation  read, write
  # agent:test            read
  # topic:specs           read
}

# access principal add
example_access_principal_add() {
  gh corkboard --as human:owner access principal add agent:test
  # Expected stdout
  # Registered agent:test (member).
}

# access principal list
example_access_principal_list() {
  gh corkboard --as human:owner access principal list
  # Expected stdout
  # agent:implementation  member
  # agent:specs           member
  # agent:test            member
  # human:owner           admin
}

# access principal set
example_access_principal_set() {
  gh corkboard --as human:owner access principal set human:reviewer --role admin
  # Expected stdout
  # Changed human:reviewer to admin.
}

# access topic create
example_access_topic_create() {
  gh corkboard --as human:owner access topic create scratch
  # Expected stdout
  # Created topic:scratch.
}

# access topic delete
example_access_topic_delete() {
  gh corkboard --as human:owner access topic delete scratch
  # Expected stdout
  # Removed the empty topic:scratch folder and its grants.
}

# access list
example_access_list() {
  gh corkboard --as human:owner access list --scope topic:specs
  # Expected stdout
  # topic:specs  agent:implementation  read
  # topic:specs  agent:specs           write, status
  # topic:specs  agent:test            status
}

# access grant
example_access_grant() {
  gh corkboard --as human:owner access grant topic:specs --to agent:implementation --permission status
  # Expected stdout
  # Granted status on topic:specs to agent:implementation.
}

# access revoke
example_access_revoke() {
  gh corkboard --as human:owner access revoke topic:specs --from agent:implementation --permission status
  # Expected stdout
  # Revoked status on topic:specs from agent:implementation.
}
```

### JSON output

#### access.json

```json
{
  "jsonStdout": {
    "init": {
      "initialized": true,
      "ignoreRuleAdded": false
    },
    "access whoami": {
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
    },
    "access principal add": {
      "principal": "agent:test",
      "role": "member"
    },
    "access principal list": {
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
    },
    "access principal set": {
      "principal": "human:reviewer",
      "role": "admin",
      "changed": true
    },
    "access topic create": {
      "scope": "topic:scratch",
      "created": true
    },
    "access topic delete": {
      "scope": "topic:scratch",
      "deleted": true
    },
    "access list": {
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
    },
    "access grant": {
      "scope": "topic:specs",
      "principal": "agent:implementation",
      "permission": "status",
      "changed": true
    },
    "access revoke": {
      "scope": "topic:specs",
      "principal": "agent:implementation",
      "permission": "status",
      "changed": true
    }
  }
}
```

## Diagnostics and exit codes

### Default terminal output

#### diagnostics.sh

```bash
#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# Schema validation finding
# A schema violation exits 1. The diagnostic report goes to stdout in both text and JSON modes.
example_schema_validation_finding() {
  gh corkboard --as agent:implementation doc validate ae3f039f-33f8-4906-99e5-43e0f21095c2
  # Expected stdout
  # ae3f039f-33f8-4906-99e5-43e0f21095c2  invalid
  #   Required property "title" is missing.
}

# Index audit finding
# Index drift exits 1. The diagnostic report goes to stdout.
example_index_audit_finding() {
  gh corkboard --as human:owner index audit
  # Expected stdout
  # Unindexed document: agents/test/local-results.csv
}

# Malformed input on write
# Malformed input exits 1, leaves the file and index untouched, and reports a diagnostic on stdout.
example_malformed_input_on_write() {
  printf '%s' '{"title":' | gh corkboard --as agent:specs doc create --in topic:specs --name broken --format json
  # Expected stdout
  # Invalid JSON: unexpected end of input.
}

# Unavailable document
# A missing or inaccessible UUID exits 2. The error goes to stderr and stdout is empty.
example_unavailable_document() {
  gh corkboard --as agent:implementation doc get 2a0e27a3-9a98-43b5-a1a9-a6079c93fcd2
  # Expected stderr
  # error: document unavailable
}
```

### JSON output

#### diagnostics.json

```json
{
  "jsonStdout": {
    "doc validate (schema violation)": {
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
    },
    "index audit (drift)": {
      "findings": [
        {
          "code": "unindexed_document",
          "path": "agents/test/local-results.csv"
        }
      ]
    },
    "doc create (malformed input)": {
      "diagnostics": [
        {
          "code": "invalid_json",
          "message": "Unexpected end of input."
        }
      ]
    }
  }
}
```

## Documents

### Default terminal output

#### documents.sh

```bash
#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# doc create
example_doc_create() {
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --description "Requirements for cursor pagination" --status active < pagination.md
  # Expected stdout
  # Created 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb (topic:specs / pagination, md, active).
}

# doc get
# Stdout contains exactly the stored Markdown bytes.
example_doc_get() {
  gh corkboard --as agent:implementation doc get 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
  # Expected stdout
  # # Pagination
  #
  # ## Acceptance criteria
  #
  # - Return the next cursor when more results are available.
}

# doc locate
example_doc_locate() {
  gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination --format md
  # Expected stdout
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
}

# doc update
example_doc_update() {
  gh corkboard --as agent:test doc update 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator b69ac8e7-6755-4249-83ef-211daa24ab6f
  # Expected stdout
  # Updated 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb. Status: approved. Review by agent:test: pass.
}

# doc delete
example_doc_delete() {
  gh corkboard --as human:owner doc delete b69ac8e7-6755-4249-83ef-211daa24ab6f
  # Expected stdout
  # Deleted document b69ac8e7-6755-4249-83ef-211daa24ab6f.
}

# doc validate
example_doc_validate() {
  gh corkboard --as agent:implementation doc validate 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
  # Expected stdout
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  schemaless
}
# The --all variant uses the same result shape and includes readable indexed documents only.
```

### JSON output

#### documents.json

```json
{
  "jsonStdout": {
    "doc create": {
      "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
      "scope": "topic:specs",
      "name": "pagination",
      "format": "md",
      "description": "Requirements for cursor pagination",
      "status": "active",
      "reviewComments": {}
    },
    "doc locate": {
      "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb"
    },
    "doc update": {
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
    },
    "doc delete": {
      "id": "b69ac8e7-6755-4249-83ef-211daa24ab6f",
      "deleted": true
    },
    "doc validate": {
      "results": [
        {
          "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
          "format": "md",
          "state": "schemaless",
          "diagnostics": []
        }
      ]
    }
  }
}
```

## Index and snapshots

### Default terminal output

#### maintenance.sh

```bash
#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# index search
example_index_search() {
  gh corkboard --as agent:implementation index search --in topic:specs --status approved
  # Expected stdout
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  approved  topic:specs / pagination (md)
  #   Requirements for cursor pagination
  #   agent:specs [pass] Acceptance criteria cover the edge cases.
  #   agent:test [pass] The test plan covers every criterion.
  #     Review document: b69ac8e7-6755-4249-83ef-211daa24ab6f
}

# index audit
example_index_audit() {
  gh corkboard --as human:owner index audit
  # Expected stdout
  # Index is in sync.
}

# index resync
example_index_resync() {
  gh corkboard --as human:owner index resync
  # Expected stdout
  # Indexed 1 new document as draft.
  # 7ee9c97c-dcbb-4355-898a-2fbb18868362  agents/test/local-results.csv
}

# index prune
# In this example the pagination document has been marked archived and its recorded digest matches the file.
example_index_prune() {
  gh corkboard --as human:owner index prune
  # Expected stdout
  # Plan c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3
  # 1 archived document is eligible for deletion.
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  topics/specs/pagination.md
}

# index prune --apply
# For the same plan, apply mode reports the documents actually removed.
example_index_prune_apply() {
  gh corkboard --as human:owner index prune --apply --expect c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3
  # Expected stdout
  # Deleted 1 document.
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
}

# snapshot create
# Stdout is a binary ZIP stream. The first command saves the stream. The second inspects the archive and produces the listing shown here.
example_snapshot_create() {
  gh corkboard --as human:owner snapshot create > corkboard.zip
  unzip -Z1 corkboard.zip
  # Expected stdout from unzip
  # .gh-corkboard/
  # .gh-corkboard/agents/
  # .gh-corkboard/agents/implementation/
  # .gh-corkboard/agents/specs/
  # .gh-corkboard/agents/test/
  # .gh-corkboard/agents/test/pagination-review.md
  # .gh-corkboard/index/
  # .gh-corkboard/index/access.json
  # .gh-corkboard/index/documents.json
  # .gh-corkboard/schemas/
  # .gh-corkboard/schemas/csv/
  # .gh-corkboard/schemas/json/
  # .gh-corkboard/schemas/md/
  # .gh-corkboard/skills/
  # .gh-corkboard/topics/
  # .gh-corkboard/topics/specs/
  # .gh-corkboard/topics/specs/pagination.md
}
```

### JSON output

#### maintenance.json

```json
{
  "jsonStdout": {
    "index search": {
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
    },
    "index audit": {
      "findings": []
    },
    "index resync": {
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
    },
    "index prune": {
      "planId": "c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3",
      "candidates": [
        {
          "id": "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb",
          "path": "topics/specs/pagination.md"
        }
      ],
      "skipped": []
    },
    "index prune --apply": {
      "planId": "c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3",
      "deleted": [
        "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb"
      ],
      "remaining": []
    }
  }
}
```

## Schemas

### Default terminal output

#### schemas.sh

```bash
#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# schema create
example_schema_create() {
  gh corkboard --as human:owner schema create json spec < spec.schema.json
  # Expected stdout
  # Registered the json schema spec.
}

# schema get
# Stdout is the stored schema. The JSON below is the file content.
example_schema_get() {
  gh corkboard --as agent:implementation schema get json spec
  # Expected stdout is rawStdout["schema get"] in schemas.json.
}

# schema list
example_schema_list() {
  gh corkboard --as agent:implementation schema list --format json
  # Expected stdout
  # json  spec
}

# schema update
example_schema_update() {
  gh corkboard --as human:owner schema update json spec < spec-v2.schema.json
  # Expected stdout
  # Updated the json schema spec.
}

# schema delete
example_schema_delete() {
  gh corkboard --as human:owner schema delete json spec
  # Expected stdout
  # Deleted the unused json schema spec.
}
```

### JSON output

#### schemas.json

```json
{
  "jsonStdout": {
    "schema create": {
      "format": "json",
      "schemaId": "spec",
      "created": true
    },
    "schema list": {
      "schemas": [
        {
          "format": "json",
          "schemaId": "spec"
        }
      ]
    },
    "schema update": {
      "format": "json",
      "schemaId": "spec",
      "updated": true
    },
    "schema delete": {
      "format": "json",
      "schemaId": "spec",
      "deleted": true
    }
  },
  "rawStdout": {
    "schema get": {
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
  }
}
```


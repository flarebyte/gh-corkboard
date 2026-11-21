#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# help
example_help() {
  gh corkboard help
  # Expected stdout
  # Manage a local project corkboard for agent handoffs and human review.
  #
  # Usage
  #   gh corkboard --as <principal> <command> [flags]
  #   gh corkboard help [<command> ...]
  #
  # Commands
  #   init       Initialize the local board
  #   access     Manage principals, topics, and grants
  #   schema     Manage document schemas
  #   doc        Create, read, edit, delete, and validate documents
  #   index      Search, reconcile, and prune the document index
  #   snapshot   Create a ZIP archive of the board
  #   help       Show root or command-specific help
  #
  # Global flags
  #   --as <principal>   Select the caller for a board command
  #   --json             Emit JSON for supported results
  #   -h, --help          Show help for the selected command
  #   --version          Show the CLI version
  #
  # Examples
  #   gh corkboard --as human:owner init
  #   gh corkboard help doc
  #   gh corkboard help access principal add
}

# Command group help
example_help_doc() {
  gh corkboard help doc
  # Expected stdout
  # Usage
  #   gh corkboard --as <principal> doc <command> [flags]
  #
  # Commands
  #   create    Create a document from stdin
  #   get       Write the stored bytes to stdout
  #   locate    Resolve a logical document name to a UUID
  #   patch     Apply a patch against an exact base revision
  #   update    Replace content or update metadata
  #   delete    Remove one document
  #   validate  Check a document or all readable indexed documents
  #
  # Run gh corkboard help doc <command> for details.
}

# Leaf command help
example_help_doc_patch() {
  gh corkboard help doc patch
  # Expected stdout
  # Apply a patch from stdin to an existing document. Requires write access.
  #
  # Usage
  #   gh corkboard --as <principal> doc patch <doc-id> --expect <sha256> [flags]
  #
  # Input
  #   Markdown and CSV use a unified diff. JSON uses RFC 7396 Merge Patch.
  #
  # Output
  #   Writes a patch summary to stdout. A stale base exits 2 with an error on stderr.
  #
  # Flags
  #   --expect <sha256>      Required SHA-256 of the exact bytes read with doc get
  #   --preserve-row-order   Keep CSV row order even when the first header is id or name
  #   --json                 Emit updated document metadata as JSON
  #   -h, --help             Show this help
  #
  # Example
  #   gh corkboard --as agent:specs doc patch "$doc_id" --expect "$base_sha256" < changes.diff
}

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

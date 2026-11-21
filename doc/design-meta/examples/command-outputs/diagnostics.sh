#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# Unknown help target
# Exits 2 with empty stdout.
example_unknown_help_target() {
  gh corkboard help doc open
  # Expected stderr
  # error: unknown command "open" under "doc"
  # Run gh corkboard help doc to list document commands.
}

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

# Invalid concept
# Invalid metadata flags exit 2 and leave the document and index untouched.
example_invalid_concept() {
  gh corkboard --as agent:specs doc update 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb +    --concept "cursor pagination"
  # Expected stderr
  # error: --concept must be 1–48 characters in slash-separated segments of ASCII letters.
}

# Unavailable document
# A missing or inaccessible UUID exits 2. The error goes to stderr and stdout is empty.
example_unavailable_document() {
  gh corkboard --as agent:implementation doc get 2a0e27a3-9a98-43b5-a1a9-a6079c93fcd2
  # Expected stderr
  # error: document unavailable
}

# Ambiguous document name
# The two pagination documents in topic:specs have different milestones. Exits 2.
example_ambiguous_locate() {
  gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination
  # Expected stderr
  # error: pagination matches multiple documents in topic:specs. Supply --milestone.
}

# Stale patch base
# A changed base exits 2 and leaves the file and index untouched.
example_stale_patch_base() {
  local expected
  expected=$(shasum -a 256 < pagination.before.json) || return
  printf '%s\n' '{"pageSize":50}' |
    gh corkboard --as agent:specs doc patch fa334448-c8de-42b0-b013-ebc87dc5f7ae --expect "${expected%% *}"
  # Expected stderr
  # error: document changed since the base was read. Read it again and regenerate the patch.
}

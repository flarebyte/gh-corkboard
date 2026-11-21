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

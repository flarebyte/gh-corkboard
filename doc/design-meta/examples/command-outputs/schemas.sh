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

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

#!/usr/bin/env bash

# Source this file and call individual functions in a prepared worktree.
# Each use case is independent. Counts are Corkboard calls after setup.
# Each row with gh is one board call. jq, shasum, and input preparation are local steps.
# The review workflow provisions topic:specs and the three agent principals.

# Use case: handoff
# Board calls: 4
# Start with an indexed spec UUID. After reading the handoff, pass the UUID from its text to open_spec.
usecase_handoff_publish() (
  set -euo pipefail
  spec_id=${1:?Provide the indexed spec UUID}
  printf '# Pagination handoff\n\nImplement the spec with UUID %s.\n' "$spec_id" | gh corkboard --as agent:specs doc create --in topic:specs --name pagination-handoff --format md --milestone "v1 launch/Specs" --concept Cursor/Handoff --assignee agent:implementation --status active
)
usecase_handoff_read() (
  set -euo pipefail
  handoff_id=$(gh corkboard --as agent:implementation doc search --in topic:specs --milestone "v1 launch/Specs" --concept Cursor/Handoff --json | jq -er '.documents | if length == 1 then .[0].id else error("select the handoff UUID") end')
  gh corkboard --as agent:implementation doc get "$handoff_id"
)
usecase_handoff_open_spec() {
  gh corkboard --as agent:implementation doc get "${1:?Provide the spec UUID from the handoff}"
}

# Use case: self-note
# Board calls: 3
# Create the note now. Locate and open it in a later session using its known name.
usecase_self_note_write() {
  gh corkboard --as agent:implementation doc create --in agent:implementation --name follow-up --format md < "${1:?Provide the local note file}"
}
usecase_self_note_read() (
  set -euo pipefail
  note_id=$(gh corkboard --as agent:implementation doc locate --in agent:implementation --name follow-up)
  gh corkboard --as agent:implementation doc get "$note_id"
)

# Use case: skill-reference
# Board calls: 3
# Start with a registered skill:review and its indexed guide. A human approves sharing.
usecase_skill_reference() (
  set -euo pipefail
  gh corkboard --as human:owner access grant skill:review --to agent:implementation --permission read
  guide_id=$(gh corkboard --as agent:implementation doc search --in skill:review --query review-guide --json | jq -er '.documents | if length == 1 then .[0].id else error("select the guide UUID") end')
  gh corkboard --as agent:implementation doc get "$guide_id"
)

# Use case: review
# Board calls: 6
# Start with the spec UUID, then prepare the review file before calling finish.
# The implementation agent already has a read grant on agent:test.
usecase_review_start() (
  set -euo pipefail
  spec_id=${1:?Provide the indexed spec UUID}
  gh corkboard --as agent:specs doc update "$spec_id" --status review --review-status pass --review-comment "Acceptance criteria cover the edge cases."
  gh corkboard --as agent:test doc get "$spec_id"
)
usecase_review_finish() (
  set -euo pipefail
  spec_id=${1:?Provide the indexed spec UUID}
  review_file=${2:?Provide the local test review file}
  review_id=$(gh corkboard --as agent:test doc create --in agent:test --name pagination-review --format md --json < "$review_file" | jq -er '.id')
  gh corkboard --as agent:test doc update "$spec_id" --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator "$review_id"
)
usecase_review_read() (
  set -euo pipefail
  spec_id=${1:?Provide the indexed spec UUID}
  review_id=$(gh corkboard --as agent:implementation doc search --in topic:specs --milestone "v1 launch/Specs" --concept Cursor/Pagination --status approved --json | jq -er --arg id "$spec_id" '.documents[] | select(.id == $id) | .reviewComments["agent:test"].locator')
  gh corkboard --as agent:implementation doc get "$review_id"
)

# Use case: find-work
# Board calls: 1
usecase_find_work() {
  gh corkboard --as agent:implementation doc search --milestone 'v1 launch/*' --concept 'Cursor/*' --format table
}

# Use case: supervise
# Board calls: 2 for the human to list work and open one document.
# The agent publishes with one call and can report a changed note and status with one call.
# Grant the implementation agent status permission on its channel once during setup.
example_supervision_setup() {
  gh corkboard --as human:owner access grant agent:implementation --to agent:implementation --permission status
}
example_publish_progress() {
  gh corkboard --as agent:implementation doc create --in agent:implementation --name pagination-implementation --format md --description "Implement the next cursor" --milestone "v1 launch/Specs" --concept Cursor/Implementation --assignee agent:implementation --status active < "${1:?Provide the local progress note}"
}
example_report_blocker() {
  gh corkboard --as agent:implementation doc update "${1:?Provide the progress document UUID}" --stdin --status blocked < "${2:?Provide the updated progress note}"
}
usecase_supervise_list() {
  gh corkboard --as human:owner doc search --milestone 'v1 launch/*' --status active --status blocked --status review --format table
}
usecase_supervise_open() {
  gh corkboard --as human:owner doc get "${1:?Choose a UUID from the milestone results}"
}

# Use case: reuse-name
# Board calls: 4
# Two local spec files can use the same logical board name in different milestones.
usecase_reuse_name() (
  set -euo pipefail
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --milestone "v1 launch/Specs" < "${1:?Provide the v1 spec}"
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md --milestone v2/Specs < "${2:?Provide the v2 spec}"
  gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination --milestone "v1 launch/Specs"
  gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination --milestone v2/Specs
)

# Use case: draft
# Board calls: 2
# Start with the registered JSON spec schema that requires title. Validation exits 1.
usecase_draft() (
  set -euo pipefail
  draft_id=$(printf '%s\n' '{"pageSize":50}' | gh corkboard --as agent:specs doc create --in topic:specs --name draft-example --format json --schema spec --json | jq -er '.id')
  gh corkboard --as agent:specs doc validate "$draft_id" --json
)

# Use case: patch
# Board calls: 2
# Prepare the patch locally from the saved base before calling usecase_patch_apply.
usecase_patch_read() {
  gh corkboard --as agent:specs doc get "${1:?Provide a document UUID}" > "${2:?Provide the local base filename}"
}
usecase_patch_apply() (
  set -euo pipefail
  doc_id=${1:?Provide the document UUID}
  base_file=${2:?Provide the exact saved base filename}
  patch_file=${3:?Provide the patch prepared against that base}
  digest=$(shasum -a 256 < "$base_file")
  gh corkboard --as agent:specs doc patch "$doc_id" --expect "${digest%% *}" < "$patch_file"
)

# Use case: retire
# Board calls: 3
# Start with archived documents. Review the preview and pass its planId to the apply function.
usecase_retire_preview() {
  gh corkboard --as human:owner index prune --json
}
usecase_retire_apply() (
  plan_id=${1:?Provide the reviewed plan ID}
  archive=${2:?Provide a new ZIP filename outside the board}
  set -o noclobber
  gh corkboard --as human:owner snapshot create --expect "$plan_id" > "$archive" &&
  gh corkboard --as human:owner index prune --apply --expect "$plan_id"
)

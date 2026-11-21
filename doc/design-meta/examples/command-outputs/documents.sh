#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

# doc create
example_doc_create() {
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md \
    --description "Requirements for cursor pagination" --milestone "v1 launch/Specs" \
    --concept Cursor/Pagination --assignee agent:specs --status active < pagination.md
  # Expected stdout
  # Created 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb (topic:specs / pagination, md, active).
}

# doc create with the same name in another milestone
example_doc_create_v2() {
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md \
    --description "Cursor pagination requirements for the v2 API" --milestone v2/Specs \
    --concept Cursor/Pagination < pagination-v2.md
  # Expected stdout
  # Created cf89eab1-3451-4e41-a6bb-0c6110a9d934 (topic:specs / pagination, md, draft).
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

# doc get (content and revision for a guarded write)
# See documents.json for the JSON stdout. Save it before preparing the write.
example_doc_get_with_revision() {
  gh corkboard --as agent:specs doc get 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --with-revision
}

# doc get piped to jq
example_doc_get_json_pipe() (
  set -o pipefail
  gh corkboard --as agent:implementation doc get fa334448-c8de-42b0-b013-ebc87dc5f7ae |
    jq '.pageSize'
  # Expected stdout from jq
  # 100
)

# doc locate
example_doc_locate() {
  gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination \
    --milestone "v1 launch/Specs"
  # Expected stdout
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb
}

# doc search
example_doc_search() {
  gh corkboard --as agent:implementation doc search --in topic:specs --status approved \
    --milestone 'v1 launch/*' --concept 'cursor/*'
  # Expected stdout
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  approved  topic:specs / pagination (md)
  #   Requirements for cursor pagination
  #   Milestone: v1 launch/Specs
  #   Concept: Cursor/Pagination
  #   Assignee: agent:specs
  #   agent:specs [pass] Acceptance criteria cover the edge cases.
  #   agent:test [pass] The test plan covers every criterion.
  #     Review document: b69ac8e7-6755-4249-83ef-211daa24ab6f
}

# doc search (compact table)
# The exact status filter gives every row the same status.
example_doc_search_table() {
  gh corkboard --as agent:implementation doc search --in topic:specs --status approved \
    --milestone 'v1 launch/*' --concept 'cursor/*' --format table
  # Expected stdout
  # ID                                    NAME        SCOPE        DESCRIPTION                         ASSIGNEE     MILESTONE        CONCEPT
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  pagination  topic:specs  Requirements for cursor pagination  agent:specs  v1 launch/Specs  Cursor/Pagination
}

# doc search (compact CSV)
# Exact milestone and concept filters omit those columns.
example_doc_search_csv() {
  gh corkboard --as agent:implementation doc search --in topic:specs \
    --milestone 'v1 launch/Specs' --concept Cursor/Pagination --format csv
  # Expected stdout
  # id,name,scope,description,status,assignee
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb,pagination,topic:specs,Requirements for cursor pagination,approved,agent:specs
}

# doc search (milestone supervision)
# Multiple statuses and the milestone prefix keep those columns visible.
# Use --json in place of --format table for full metadata.
example_doc_search_supervision() {
  gh corkboard --as human:owner doc search --milestone 'v1 launch/*' --status active --status blocked --status review --format table
  # Expected stdout
  # ID                                    NAME                       SCOPE                 DESCRIPTION                STATUS  ASSIGNEE              MILESTONE        CONCEPT
  # 53dd1654-7fab-459f-a414-dc91198c0c61  pagination-implementation  agent:implementation  Implement the next cursor  active  agent:implementation  v1 launch/Specs  Cursor/Implementation
}

# doc get (progress note selected from the milestone search)
example_doc_get_progress() {
  gh corkboard --as human:owner doc get 53dd1654-7fab-459f-a414-dc91198c0c61
  # Expected stdout
  # # Pagination implementation
  #
  # ## Current work
  #
  # Implementing next-cursor encoding for the list endpoint.
  #
  # ## Next step
  #
  # Wire the cursor into the integration tests.
}

# doc search (inaccessible scope)
# The pagination review exists in agent:test. The specs agent has no permission there.
example_doc_search_inaccessible_scope() {
  gh corkboard --as agent:specs doc search --in agent:test
  # Expected stdout
  # No documents matched.
}

# doc update (approval of the saved content)
# The reviewer inspected the content from pagination.base.json.
example_doc_update() {
  local revision
  revision=$(jq -er '.revision' pagination.base.json) || return
  gh corkboard --as agent:test doc update 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --expect "$revision" --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator b69ac8e7-6755-4249-83ef-211daa24ab6f
  # Expected stdout
  # Updated 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb. Status: approved. Review by agent:test: pass.
}

# doc update with milestone and concept
example_doc_update_labels() {
  gh corkboard --as agent:specs doc update 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb \
    --milestone "v1.1" --concept PaginationCursors
  # Expected stdout
  # Updated 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb.
  # Milestone: v1.1
  # Concept: PaginationCursors
}

# doc update with replacement content
example_doc_update_content() {
  printf '%s\n' '{"title":"Cursor pagination","pageSize":100}' |
    gh corkboard --as agent:specs doc update fa334448-c8de-42b0-b013-ebc87dc5f7ae --stdin
  # Expected stdout
  # Updated document fa334448-c8de-42b0-b013-ebc87dc5f7ae.
  # Formatting changed the supplied bytes.
}

# doc patch (Markdown)
# Save doc get --with-revision as pagination.base.json. Draft the patch from its content.
example_doc_patch_markdown() {
  local revision
  revision=$(jq -er '.revision' pagination.base.json) || return
  gh corkboard --as agent:specs doc patch 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb --expect "$revision" <<'PATCH'
--- document
+++ document
@@ -5 +5 @@
-- Return the next cursor when more results are available.
+- Return a next cursor only when more results are available.
PATCH
  # Expected stdout
  # Patched document 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb.
}

# doc patch (CSV)
# The saved priorities.base.json has CSV content with id,priority followed by p1,01 and p2,02.
example_doc_patch_csv() {
  local revision
  revision=$(jq -er '.revision' priorities.base.json) || return
  gh corkboard --as agent:specs doc patch 11290eeb-00c7-44c0-83e9-a4618ec9d3f4 --expect "$revision" <<'PATCH'
--- document
+++ document
@@ -1,3 +1,4 @@
 id,priority
 p1,01
 p2,02
+p0,00
PATCH
  # Expected stdout
  # Patched document 11290eeb-00c7-44c0-83e9-a4618ec9d3f4.
  # Formatting changed the patched content.
}

# doc patch (JSON)
# Save doc get --with-revision as pagination-settings.base.json before drafting the merge patch.
example_doc_patch_json() {
  local revision
  revision=$(jq -er '.revision' pagination-settings.base.json) || return
  gh corkboard --as agent:specs doc patch fa334448-c8de-42b0-b013-ebc87dc5f7ae --expect "$revision" <<'PATCH'
{"pageSize":50}
PATCH
  # Expected stdout
  # Patched document fa334448-c8de-42b0-b013-ebc87dc5f7ae.
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

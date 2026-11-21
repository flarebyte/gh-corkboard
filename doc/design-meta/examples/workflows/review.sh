#!/usr/bin/env bash

# Source this file in Bash and call example_review_workflow in a fresh Git worktree.
# Have pagination.md, pagination-review.md, and follow-up.md in the current directory.
example_review_workflow() (
  set -euo pipefail

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
  gh corkboard --as agent:specs doc create --in topic:specs --name pagination --format md \
    --description "Requirements for cursor pagination" --milestone "v1 launch/Specs" --concept Cursor/Pagination \
    --assignee agent:specs --status active --json < pagination.md
  doc_id=$(gh corkboard --as agent:implementation doc locate --in topic:specs --name pagination \
    --milestone "v1 launch/Specs")
  gh corkboard --as agent:implementation doc get "$doc_id"
  gh corkboard --as agent:specs doc update "$doc_id" --status review --review-status pass --review-comment "Acceptance criteria cover the edge cases."
  gh corkboard --as agent:test doc create --in agent:test --name pagination-review --format md --description "Test review of the cursor pagination spec" < pagination-review.md
  review_id=$(gh corkboard --as agent:test doc locate --in agent:test --name pagination-review)
  gh corkboard --as agent:test doc update "$doc_id" --status approved --review-status pass --review-comment "The test plan covers every criterion." --review-locator "$review_id"
  gh corkboard --as agent:implementation doc search --in topic:specs --status approved \
    --milestone "v1 launch/specs" --concept cursor/pagination --json
  gh corkboard --as agent:implementation doc get "$review_id"
  gh corkboard --as agent:implementation doc create --in agent:implementation --name follow-up --format md < follow-up.md
)

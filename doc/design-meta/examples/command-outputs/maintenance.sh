#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

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
  # 7ee9c97c-dcbb-4355-898a-2fbb18868362  agents/test/7ee9c97c-dcbb-4355-898a-2fbb18868362--local-results.csv
}

# index prune
# In this example the pagination document has been marked archived and its recorded digest matches the file.
example_index_prune() {
  gh corkboard --as human:owner index prune
  # Expected stdout
  # Plan c7f4494f4450f8616639b3d733ab7257407c2d5f43adf751bfc8b0d2f12f07b3
  # 1 archived document is eligible for deletion.
  # 29b4dd81-af57-4c98-b3f0-ff0bf66da7bb  topics/specs/29b4dd81-af57-4c98-b3f0-ff0bf66da7bb--pagination.md
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
  # .gh-corkboard/agents/implementation/53dd1654-7fab-459f-a414-dc91198c0c61--pagination-implementation.md
  # .gh-corkboard/agents/specs/
  # .gh-corkboard/agents/test/
  # .gh-corkboard/agents/test/b69ac8e7-6755-4249-83ef-211daa24ab6f--pagination-review.md
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
  # .gh-corkboard/topics/specs/29b4dd81-af57-4c98-b3f0-ff0bf66da7bb--pagination.md
  # .gh-corkboard/topics/specs/cf89eab1-3451-4e41-a6bb-0c6110a9d934--pagination.md
}

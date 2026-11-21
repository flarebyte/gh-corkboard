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

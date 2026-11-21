#!/usr/bin/env bash

# Each function is an independent example. Call a function explicitly to run it.

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

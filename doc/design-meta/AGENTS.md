# Corkboard specification source

Use [examples/concepts.csv](examples/concepts.csv) to discover the relevant concept. Retrieve matching rows from the CSV inputs, then read the corresponding contracts in [notes.cue](notes.cue) and [examples/contracts.ts](examples/contracts.ts). The [generated specification](../design/corkboard.md) provides the overall design.

Run searches from the repository root.

```sh
rg -n -i 'index|retention' doc/design-meta/examples/concepts.csv
rg -n '(^|,)board\.(index|retention)(,|$)' doc/design-meta/examples -g '*.csv'
rg -n 'board\.(index|retention)' doc/design-meta -g '*.cue'
```

Use the CSV headers and a CSV parser for exact column filters. Some focused inputs, such as the status list, have no concept column. Preserve stable IDs and keep related concepts in the canonical registry.

After editing the sources, validate and regenerate the report.

```sh
flyb validate --config doc/design-meta
flyb generate markdown --config doc/design-meta
```

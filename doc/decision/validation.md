# Validation engine for gh-corkboard

## Problem

### Title
Validation engine for gh-corkboard

### Goal
Select a v1 schema language for agent-authored documents and the local index.

### Description
Recommended for v1. Use JSON Schema Draft 2020-12 with native JSON and fixed projections for CSV and Markdown.

### Notes
- Both alternatives depend on the same CSV and Markdown parsers and projections.
- Scores are qualitative design judgments on a 1–5 scale.
- The agent authoring scores are provisional. Compare both languages on the same tasks and examples, such as requiring JSON properties, CSV columns, or Markdown headings. Measure first-pass correctness against valid and invalid fixtures, then count revisions after validation feedback.
- Revisit CUE when actual schemas repeatedly need reusable constraints or cross-field relationships.
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) · [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)

## Alternatives

### JSON Schema Draft 2020-12
Store reusable JSON Schema artifacts and validate JSON values and format-specific projections.

### CUE
Author CUE constraints and validate the same parsed values and projections through the CUE Go API.

## Scenario

### Local corkboard v1

## Decision Drivers

### Criteria

#### Schema interoperability
Reuse schema artifacts directly in JSON Schema tools.

#### V1 integration fit
Use the same schema representation for the registry and the JSON index.

#### Constraint expressiveness
Compose reusable schemas and encode relationships between fields.

#### Ease of schema authoring for AI agents
Create and revise a correct v1 schema from a short requirement and a format example with minimal iteration.

### Preference Justifications
- Local corkboard v1: Schema interoperability over V1 integration fit (strength 1.000000)
- Local corkboard v1: Schema interoperability over Constraint expressiveness (strength 3.000000)
- Local corkboard v1: V1 integration fit over Constraint expressiveness (strength 3.000000)
- Local corkboard v1: Ease of schema authoring for AI agents over Schema interoperability (strength 1.000000)
- Local corkboard v1: Ease of schema authoring for AI agents over V1 integration fit (strength 1.000000)
- Local corkboard v1: Ease of schema authoring for AI agents over Constraint expressiveness (strength 3.000000)

### Criteria Weights
- Local corkboard v1: Ease of schema authoring for AI agents=0.300000, Constraint expressiveness=0.100000, V1 integration fit=0.300000, Schema interoperability=0.300000

## Scenario Ranking

### Local corkboard v1
1. JSON Schema Draft 2020-12 (0.767244)
2. CUE (0.232756)

#### Evaluation Notes

##### JSON Schema Draft 2020-12
For agent authoring, standard keyword patterns are easy to demonstrate with examples. Required properties and rules for extra properties must be specified separately.

Scores:
- Schema interoperability: 5
- V1 integration fit: 5
- Constraint expressiveness: 3
- Ease of schema authoring for AI agents: 4

##### CUE
For agent authoring, CUE expresses common constraints compactly. Examples need to make optional fields and open or closed structs explicit.

Scores:
- Schema interoperability: 3
- V1 integration fit: 4
- Constraint expressiveness: 5
- Ease of schema authoring for AI agents: 4

## Final Ranking

1. JSON Schema Draft 2020-12 (0.767244)
2. CUE (0.232756)

## Notes and Tradeoffs

- Aggregation method: equal_average

### Scenario Weights
- Local corkboard v1: 1.000000

### Scenario Criteria Weights
- Local corkboard v1: Ease of schema authoring for AI agents=0.300000, Constraint expressiveness=0.100000, V1 integration fit=0.300000, Schema interoperability=0.300000

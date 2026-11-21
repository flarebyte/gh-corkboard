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

### Preference Justifications
- Local corkboard v1: Schema interoperability over V1 integration fit (strength 1.000000)
- Local corkboard v1: Schema interoperability over Constraint expressiveness (strength 3.000000)
- Local corkboard v1: V1 integration fit over Constraint expressiveness (strength 3.000000)

### Criteria Weights
- Local corkboard v1: Constraint expressiveness=0.142857, V1 integration fit=0.428571, Schema interoperability=0.428571

## Scenario Ranking

### Local corkboard v1
1. JSON Schema Draft 2020-12 (0.767244)
2. CUE (0.232756)

#### Evaluation Notes

##### JSON Schema Draft 2020-12

Scores:
- Schema interoperability: 5
- V1 integration fit: 5
- Constraint expressiveness: 3

##### CUE

Scores:
- Schema interoperability: 3
- V1 integration fit: 4
- Constraint expressiveness: 5

## Final Ranking

1. JSON Schema Draft 2020-12 (0.767244)
2. CUE (0.232756)

## Notes and Tradeoffs

- Aggregation method: equal_average

### Scenario Weights
- Local corkboard v1: 1.000000

### Scenario Criteria Weights
- Local corkboard v1: Constraint expressiveness=0.142857, V1 integration fit=0.428571, Schema interoperability=0.428571

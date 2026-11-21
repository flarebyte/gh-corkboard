package seer

config: {
	problem: {
		name:        "corkboard-validation"
		title:       "Validation engine for gh-corkboard"
		goal:        "Select a v1 schema language for agent-authored documents and the local index."
		description: "Recommended for v1. Use JSON Schema Draft 2020-12 with native JSON and fixed projections for CSV and Markdown."
		notes: [
			"Both alternatives depend on the same CSV and Markdown parsers and projections.",
			"Scores are qualitative design judgments on a 1–5 scale.",
			"The agent authoring scores are provisional. Compare both languages on the same tasks and examples, such as requiring JSON properties, CSV columns, or Markdown headings. Measure first-pass correctness against valid and invalid fixtures, then count revisions after validation feedback.",
			"Revisit CUE when actual schemas repeatedly need reusable constraints or cross-field relationships.",
			"[JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) · [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)",
		]
	}
	reports: [{
		name:     "decision"
		title:    "Validation engine for gh-corkboard"
		format:   "markdown"
		filepath: "../decision/validation.md"
		arguments: ["detail=standard", "include-context=true", "include-weights=true", "include-evaluation-notes=true", "include-tradeoffs=true"]
	}]
	criteriaCatalog: [
		{name: "interoperability", title: "Schema interoperability", description: "Reuse schema artifacts directly in JSON Schema tools.", polarity: "benefit", valueType: "number"},
		{name: "integration", title: "V1 integration fit", description: "Use the same schema representation for the registry and the JSON index.", polarity: "benefit", valueType: "number"},
		{name: "expressiveness", title: "Constraint expressiveness", description: "Compose reusable schemas and encode relationships between fields.", polarity: "benefit", valueType: "number"},
		{name: "agent_authoring", title: "Ease of schema authoring for AI agents", description: "Create and revise a correct v1 schema from a short requirement and a format example with minimal iteration.", polarity: "benefit", valueType: "number"},
	]
	alternatives: [
		{name: "json_schema", title: "JSON Schema Draft 2020-12", description: "Store reusable JSON Schema artifacts and validate JSON values and format-specific projections."},
		{name: "cue", title: "CUE", description: "Author CUE constraints and validate the same parsed values and projections through the CUE Go API."},
	]
	scenarios: [{
		name:  "v1"
		title: "Local corkboard v1"
		activeCriteria: [
			{criterionName: "interoperability"},
			{criterionName: "integration"},
			{criterionName: "expressiveness"},
			{criterionName: "agent_authoring"},
		]
		preferences: {
			method: "ahp_pairwise"
			scale:  "saaty_1_9"
			comparisons: [
				{moreImportantCriterionName: "interoperability", lessImportantCriterionName: "integration", strength: 1},
				{moreImportantCriterionName: "interoperability", lessImportantCriterionName: "expressiveness", strength: 3},
				{moreImportantCriterionName: "integration", lessImportantCriterionName: "expressiveness", strength: 3},
				{moreImportantCriterionName: "agent_authoring", lessImportantCriterionName: "interoperability", strength: 1},
				{moreImportantCriterionName: "agent_authoring", lessImportantCriterionName: "integration", strength: 1},
				{moreImportantCriterionName: "agent_authoring", lessImportantCriterionName: "expressiveness", strength: 3},
			]
		}
	}]
	evaluations: [{
		scenarioName: "v1"
		evaluations: [
			{
				alternativeName: "json_schema"
				description:     "For agent authoring, standard keyword patterns are easy to demonstrate with examples. Required properties and rules for extra properties must be specified separately."
				values: {
					interoperability: {kind: "number", value: 5}
					integration: {kind: "number", value: 5}
					expressiveness: {kind: "number", value: 3}
					agent_authoring: {kind: "number", value: 4}
				}
			},
			{
				alternativeName: "cue"
				description:     "For agent authoring, CUE expresses common constraints compactly. Examples need to make optional fields and open or closed structs explicit."
				values: {
					interoperability: {kind: "number", value: 3}
					integration: {kind: "number", value: 4}
					expressiveness: {kind: "number", value: 5}
					agent_authoring: {kind: "number", value: 4}
				}
			},
		]
	}]
	aggregation: {method: "equal_average"}
}

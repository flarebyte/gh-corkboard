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
			"Revisit CUE when actual schemas repeatedly need reusable constraints or cross-field relationships.",
			"[JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) · [CUE and JSON Schema](https://cuelang.org/docs/concept/how-cue-works-with-json-schema/)",
		]
	}
	reports: [{
		name:     "decision"
		title:    "Validation engine for gh-corkboard"
		format:   "markdown"
		filepath: "../decision/validation.md"
		arguments: ["detail=standard", "include-context=true", "include-weights=true", "include-tradeoffs=true"]
	}]
	criteriaCatalog: [
		{name: "interoperability", title: "Schema interoperability", description: "Reuse schema artifacts directly in JSON Schema tools.", polarity: "benefit", valueType: "number"},
		{name: "integration", title: "V1 integration fit", description: "Use the same schema representation for the registry and the JSON index.", polarity: "benefit", valueType: "number"},
		{name: "expressiveness", title: "Constraint expressiveness", description: "Compose reusable schemas and encode relationships between fields.", polarity: "benefit", valueType: "number"},
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
		]
		preferences: {
			method: "ahp_pairwise"
			scale:  "saaty_1_9"
			comparisons: [
				{moreImportantCriterionName: "interoperability", lessImportantCriterionName: "integration", strength: 1},
				{moreImportantCriterionName: "interoperability", lessImportantCriterionName: "expressiveness", strength: 3},
				{moreImportantCriterionName: "integration", lessImportantCriterionName: "expressiveness", strength: 3},
			]
		}
	}]
	evaluations: [{
		scenarioName: "v1"
		evaluations: [
			{
				alternativeName: "json_schema"
				values: {
					interoperability: {kind: "number", value: 5}
					integration: {kind: "number", value: 5}
					expressiveness: {kind: "number", value: 3}
				}
			},
			{
				alternativeName: "cue"
				values: {
					interoperability: {kind: "number", value: 3}
					integration: {kind: "number", value: 4}
					expressiveness: {kind: "number", value: 5}
				}
			},
		]
	}]
	aggregation: {method: "equal_average"}
}

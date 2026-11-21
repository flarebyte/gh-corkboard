package flyb

import "list"

source: "gh-corkboard"
name:   "gh-corkboard"
modules: ["core"]

reports: [{
	title:       "gh-corkboard v1 specification"
	filepath:    "../design/corkboard.md"
	description: "Proposed specification for a local project corkboard and messaging system."
	sections: [{
		title: "Access control"
		sections: [{
			title: "Principals and permissions"
			notes: ["board.access", "board.access.permissions", "board.access.policy-example"]
		}]
	}, {
		title: "Board and communication"
		sections: [{
			title: "Workspace"
			notes: ["board.board"]
		}, {
			title: "Use cases"
			notes: ["board.usecases", "board.usecases.workflow"]
		}, {
			title: "Human supervision"
			notes: ["board.supervision"]
		}]
	}, {
		title: "Command interface"
		sections: [{
			title: "Commands"
			notes: ["board.commands", "board.interface", "board.interface.review-workflow"]
		}, {
			title: "Help and discovery"
			notes: ["board.cli.help"]
		}]
	}, {
		title: "Storage and validation"
		sections: [{
			title: "Document formatting"
			notes: ["board.formatting", "board.formatting.formats"]
		}, {
			title: "Schema registry and projections"
			notes: [
				"board.validation",
				"board.validation.formats",
				"board.validation.projection.csv",
				"board.validation.projection.markdown",
				"board.contracts",
			]
		}, {
			title: "Index and lifecycle"
			notes: ["board.index", "board.index.search-output", "board.index.path-resolution", "board.index.example", "board.lifecycle"]
		}, {
			title: "Internal JSON validation"
			notes: ["board.validation.internal", "board.validation.internal.fields"]
		}, {
			title: "Document patching"
			notes: ["board.patching"]
		}, {
			title: "Retention"
			notes: ["board.retention", "board.retention.archive-workflow"]
		}]
	}, {
		title: "Implementation"
		sections: [{
			title: "Go dependencies"
			notes: ["board.dependencies", "board.dependencies.reviewed"]
		}]
	}, {
		title: "Specification sources"
		sections: [{
			title: "Concept registry"
			notes: ["board.concepts"]
		}]
	}]
}, _commandOutputReport]

notes: list.Concat([[
	{name: "board.access.policy-example", title: "Access policy example", filepath: "examples/access-policy.json"},
	{name: "board.commands", title: "Command reference", filepath: "examples/commands.csv"},
	{name: "board.concepts", title: "Concept registry", filepath: "examples/concepts.csv"},
	{name: "board.contracts", title: "Data and diagnostic shapes", filepath: "examples/contracts.ts"},
	{name: "board.dependencies.reviewed", title: "Reviewed direct dependencies", filepath: "examples/go-dependencies.csv"},
	{name: "board.formatting.formats", title: "Formatting rules by format", filepath: "examples/document-formatting.csv"},
	{name: "board.index.example", title: "Document index example", filepath: "examples/documents.json"},
	{name: "board.interface.review-workflow", title: "Review and handoff example", filepath: "examples/workflows/review.sh"},
	{name: "board.validation.projection.csv", title: "CSV projection example", filepath: "examples/csv-projection.json"},
	{name: "board.lifecycle", title: "Lifecycle statuses", filepath: "examples/lifecycle.csv"},
	{name: "board.retention.archive-workflow", title: "Snapshot and prune example", filepath: "examples/workflows/archive-and-prune.sh"},
	{name: "board.validation.projection.markdown", title: "Markdown projection example", filepath: "examples/markdown-projection.json"},
	{name: "board.access.permissions", title: "Permissions by scope", filepath: "examples/scope-permissions.csv"},
	{name: "board.usecases", title: "Core workflows", filepath: "examples/usecases.csv"},
	{name: "board.usecases.workflow", title: "Bash sequences and command counts", filepath: "examples/workflows/use-cases.sh"},
	{name: "board.validation.formats", title: "Validation inputs by format", filepath: "examples/validation-inputs.csv"},
	{name: "board.validation.internal.fields", title: "Field validation rules", filepath: "examples/internal-json-validation.csv"},
], _proseNotes, _commandOutputNotes])

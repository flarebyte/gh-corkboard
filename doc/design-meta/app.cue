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
		title: "Board and communication"
		sections: [{
			title: "Workspace"
			notes: ["board.board"]
		}, {
			title: "Use cases"
			notes: ["board.usecases"]
		}]
	}, {
		title: "Command interface"
		sections: [{
			title: "Commands"
			notes: ["board.commands", "board.interface"]
		}]
	}, {
		title: "Storage and validation"
		sections: [{
			title: "Schema registry and projections"
			notes: ["board.validation", "board.contracts"]
		}, {
			title: "Index and lifecycle"
			notes: ["board.index", "board.lifecycle"]
		}, {
			title: "Retention"
			notes: ["board.retention"]
		}]
	}, {
		title: "Specification sources"
		sections: [{
			title: "Concept registry"
			notes: ["board.concepts"]
		}]
	}]
}]

notes: list.Concat([[
	{name: "board.commands", title: "Command reference", filepath: "examples/commands.csv"},
	{name: "board.concepts", title: "Concept registry", filepath: "examples/concepts.csv"},
	{name: "board.contracts", title: "Data and diagnostic shapes", filepath: "examples/contracts.ts"},
	{name: "board.lifecycle", title: "Lifecycle statuses", filepath: "examples/lifecycle.csv"},
	{name: "board.usecases", title: "Core workflows", filepath: "examples/usecases.csv"},
], _proseNotes])

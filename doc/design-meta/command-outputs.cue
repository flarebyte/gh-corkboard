package flyb

_commandOutputGroups: [
	{key: "access", title: "Board setup and access"},
	{key: "schemas", title: "Schemas"},
	{key: "documents", title: "Documents"},
	{key: "maintenance", title: "Index and snapshots"},
	{key: "diagnostics", title: "Diagnostics and exit codes"},
]

_commandOutputSources: [
	{extension: "sh", title: "Default terminal output"},
	{extension: "json", title: "JSON output"},
]

_commandOutputReport: {
	title:       "CLI output examples"
	filepath:    "../design/command-outputs.md"
	description: "Each Bash function shows an independent command example and its expected output. Success examples exit 0. In the JSON files, jsonStdout and jsonStderr map commands to output with --json. rawStdout contains stored JSON emitted directly. For commands with input redirection, place --json before the redirection."
	sections: [
		for group in _commandOutputGroups {
			title: group.title
			sections: [
				for source in _commandOutputSources {
					title: source.title
					notes: ["board.outputs.\(group.key).\(source.extension)"]
				},
			]
		},
	]
}

_commandOutputNotes: [
	for group in _commandOutputGroups
	for source in _commandOutputSources {
		name:     "board.outputs.\(group.key).\(source.extension)"
		title:    group.key + "." + source.extension
		filepath: "examples/command-outputs/\(group.key).\(source.extension)"
	},
]

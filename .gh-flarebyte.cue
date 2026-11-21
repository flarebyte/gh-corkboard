package ghflarebyte

project: {
	org:  "flarebyte"
	repo: "gh-corkboard"
}

sync: {
	mode: "push"
}

repository: {
	description:   "A corkboard for essential notes"
	defaultBranch: "main"
	homepage:      "https://github.com/flarebyte/gh-corkboard"
	visibility:    "public"
	template:      false
	topics: ["gh-extension"]
	labels: [
		{name: "bug", color: "d73a4a", description: "Something isn't working"},
		{name: "documentation", color: "0075ca", description: "Improvements or additions to documentation"},
		{name: "duplicate", color: "cfd3d7", description: "This issue or pull request already exists"},
		{name: "enhancement", color: "a2eeef", description: "New feature or request"},
		{name: "good first issue", color: "7057ff", description: "Good for newcomers"},
		{name: "help wanted", color: "008672", description: "Extra attention is needed"},
		{name: "invalid", color: "e4e669", description: "This doesn't seem right"},
		{name: "question", color: "d876e3", description: "Further information is requested"},
		{name: "wontfix", color: "ffffff", description: "This will not be worked on"},
	]

	// Verify a GitHub Actions job named "CI passed" on a pull request before syncing.
	branchProtection: {
		enabled:            true
		allowAdminCIBypass: true
	}
}

build: {
	language:             "go"
	mode:                 "binary"
	outputDir:            "build"
	checksumFile:         "build/checksums.txt"
	artifactTargetSuffix: true
	targets: [
		"darwin-arm64",
		"linux-amd64",
		"windows-amd64",
	]
}

release: {
	versionSource:    "main.project.yaml"
	tagPrefix:        "v"
	notesMode:        "generate-notes"
	includeArtifacts: true
	artifactDir:      "build"
	includeChecksums: true
}

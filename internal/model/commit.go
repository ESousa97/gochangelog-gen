// Package model contains the domain types for the changelog generator.
package model

// CommitData represents a parsed Conventional Commit message.
type CommitData struct {
	Type                string
	Scope               string
	Message             string
	IsBreaking          bool
	BreakingDescription string
}

// ChangelogData holds the context for rendering the changelog template.
type ChangelogData struct {
	Version         string
	Date            string
	ImpactScore     int
	BreakingChanges []CommitData
	Categories      []string
	GroupedCommits  map[string][]CommitData
}

// BumpType defines the semantic version bump required by a set of commits.
type BumpType int

const (
	// BumpNone indicates no version bump is required.
	BumpNone BumpType = iota
	// BumpPatch indicates a patch version bump (bug fixes, chores).
	BumpPatch
	// BumpMinor indicates a minor version bump (new features).
	BumpMinor
	// BumpMajor indicates a major version bump (breaking changes).
	BumpMajor
)

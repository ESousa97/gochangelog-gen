// Package parser provides functionality to parse Git commit messages
// following the Conventional Commits specification.
//
// See: https://www.conventionalcommits.org/
package parser

import (
	"regexp"
	"strings"

	"github.com/esousa97/gochangelog-gen/internal/model"
)

// Regex patterns for Conventional Commits parsing.
var (
	commitRegex         = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!?): (.+)$`)
	breakingFooterRegex = regexp.MustCompile(`(?m)^BREAKING[- ]CHANGE: (.*)`)
)

// ParseCommit analyzes a commit message and extracts Conventional Commit data.
// It returns the parsed data and true if the message follows the convention,
// or nil and false otherwise.
func ParseCommit(message string) (*model.CommitData, bool) {
	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return nil, false
	}

	firstLine := strings.TrimSpace(lines[0])
	matches := commitRegex.FindStringSubmatch(firstLine)
	if matches == nil {
		return nil, false
	}

	data := &model.CommitData{
		Type:       matches[1],
		Scope:      matches[2],
		IsBreaking: matches[3] == "!",
		Message:    matches[4],
	}

	// Check for BREAKING CHANGE footer in the body.
	bodyMatches := breakingFooterRegex.FindStringSubmatch(message)
	if bodyMatches != nil {
		data.IsBreaking = true
		data.BreakingDescription = strings.TrimSpace(bodyMatches[1])
	}

	return data, true
}

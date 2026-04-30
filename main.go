package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CommitData represents a parsed Conventional Commit
type CommitData struct {
	Type    string
	Scope   string
	Message string
}

// Conventional Commit Regex
// Pattern: type(scope)!: description
var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!?): (.+)$`)

func parseCommit(message string) (*CommitData, bool) {
	// Conventional commits usually have a short summary line
	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return nil, false
	}
	firstLine := strings.TrimSpace(lines[0])

	matches := commitRegex.FindStringSubmatch(firstLine)
	if matches == nil {
		return nil, false
	}

	return &CommitData{
		Type:    matches[1],
		Scope:   matches[2],
		Message: matches[4],
	}, true
}

func main() {
	// Open the repository in the current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		fmt.Printf("Error opening repository: %s\n", err)
		return
	}

	// Get the commit history (log)
	ref, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD: %s\n", err)
		return
	}

	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		fmt.Printf("Error getting log: %s\n", err)
		return
	}

	var parsedCommits []CommitData

	// Iterate over commits
	err = cIter.ForEach(func(c *object.Commit) error {
		if data, ok := parseCommit(c.Message); ok {
			parsedCommits = append(parsedCommits, *data)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error iterating commits: %s\n", err)
		return
	}

	// Output the results
	fmt.Println("Parsed Conventional Commits:")
	for _, pc := range parsedCommits {
		scope := ""
		if pc.Scope != "" {
			scope = fmt.Sprintf("(%s)", pc.Scope)
		}
		fmt.Printf("- %s%s: %s\n", pc.Type, scope, pc.Message)
	}
}

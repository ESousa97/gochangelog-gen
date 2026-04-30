package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ChangelogData holds the context for rendering the changelog template
type ChangelogData struct {
	Version         string
	Date            string
	BreakingChanges []CommitData
	Categories      []string
	GroupedCommits  map[string][]CommitData
}

const defaultTemplate = `# {{.Version}} ({{.Date}})
{{if .BreakingChanges}}
## ⚠️ BREAKING CHANGES
{{range .BreakingChanges}}- {{if .Scope}}**{{.Scope}}**: {{end}}{{.Message}}{{if .BreakingDescription}}
  *Note: {{.BreakingDescription}}*{{end}}
{{end}}{{end}}
{{- range $cat := .Categories}}
{{- $commits := index $.GroupedCommits $cat}}
{{- if $commits}}
## {{$cat}}
{{range $commits}}- {{if .Scope}}**{{.Scope}}**: {{end}}{{.Message}}
{{end}}
{{- end}}
{{- end}}
`

// CommitData represents a parsed Conventional Commit
type CommitData struct {
	Type                string
	Scope               string
	Message             string
	IsBreaking          bool
	BreakingDescription string
}

// Conventional Commit Regex
// Pattern: type(scope)!: description
var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!?): (.+)$`)
var breakingFooterRegex = regexp.MustCompile(`(?m)^BREAKING[- ]CHANGE: (.*)`)

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

	data := &CommitData{
		Type:       matches[1],
		Scope:      matches[2],
		IsBreaking: matches[3] == "!",
		Message:    matches[4],
	}

	// Check for BREAKING CHANGE: in the body
	bodyMatches := breakingFooterRegex.FindStringSubmatch(message)
	if bodyMatches != nil {
		data.IsBreaking = true
		data.BreakingDescription = strings.TrimSpace(bodyMatches[1])
	}

	return data, true
}

func main() {
	versionFlag := flag.String("version", "vUnreleased", "Version to be displayed in the changelog title")
	templateFlag := flag.String("template", "", "Path to a custom Markdown template file (.tmpl)")
	flag.Parse()

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

	// Categories and their mapping
	categories := []string{"Features", "Bug Fixes", "Documentation", "Refactor", "Others"}
	categoryMap := map[string]string{
		"feat":     "Features",
		"fix":      "Bug Fixes",
		"docs":     "Documentation",
		"refactor": "Refactor",
	}

	groupedCommits := make(map[string][]CommitData)
	var breakingChanges []CommitData

	// Iterate over commits
	err = cIter.ForEach(func(c *object.Commit) error {
		data, ok := parseCommit(c.Message)
		if ok {
			if data.IsBreaking {
				breakingChanges = append(breakingChanges, *data)
			}
			cat, mapped := categoryMap[data.Type]
			if !mapped {
				cat = "Others"
			}
			groupedCommits[cat] = append(groupedCommits[cat], *data)
		} else {
			// Non-conventional commit or empty message
			lines := strings.Split(c.Message, "\n")
			if len(lines) > 0 {
				msg := strings.TrimSpace(lines[0])
				if msg != "" {
					groupedCommits["Others"] = append(groupedCommits["Others"], CommitData{
						Message: msg,
					})
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error iterating commits: %s\n", err)
		return
	}

	// Prepare data for template
	data := ChangelogData{
		Version:         *versionFlag,
		Date:            time.Now().Format("2006-01-02"),
		BreakingChanges: breakingChanges,
		Categories:      categories,
		GroupedCommits:  groupedCommits,
	}

	// Parse template
	var tmpl *template.Template
	if *templateFlag != "" {
		tmpl, err = template.ParseFiles(*templateFlag)
		if err != nil {
			fmt.Printf("Error parsing external template: %s\n", err)
			return
		}
	} else {
		tmpl, err = template.New("changelog").Parse(defaultTemplate)
		if err != nil {
			fmt.Printf("Error parsing default template: %s\n", err)
			return
		}
	}

	// Open output file
	outFile, err := os.Create("CHANGELOG_PENDING.md")
	if err != nil {
		fmt.Printf("Error creating CHANGELOG_PENDING.md: %s\n", err)
		return
	}
	defer outFile.Close()

	// Execute template
	err = tmpl.Execute(outFile, data)
	if err != nil {
		fmt.Printf("Error executing template: %s\n", err)
		return
	}

	fmt.Println("Successfully generated CHANGELOG_PENDING.md")
}

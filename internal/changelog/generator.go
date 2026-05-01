// Package changelog generates the markdown content for changelogs
// from a set of Git commits, applying Conventional Commits categorization,
// Semantic Versioning, and template-based rendering.
package changelog

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"github.com/esousa97/gochangelog-gen/internal/model"
	"github.com/esousa97/gochangelog-gen/internal/parser"
)

const defaultTemplate = `# {{.Version}} ({{.Date}})
**Release Impact Score:** {{.ImpactScore}}

{{if .BreakingChanges}}
## BREAKING CHANGES
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

// categories defines the ordered list of changelog sections.
var categories = []string{"Features", "Bug Fixes", "Documentation", "Refactor", "Others"}

// categoryMap maps Conventional Commit types to changelog categories.
var categoryMap = map[string]string{
	"feat":     "Features",
	"fix":      "Bug Fixes",
	"docs":     "Documentation",
	"refactor": "Refactor",
}

// Generator handles the aggregation of commits and rendering of the changelog.
type Generator struct {
	commits     object.CommitIter
	tagMap      map[string]string // commit hash -> tag name
	templateStr string
}

// NewGenerator creates a new changelog [Generator].
// If templatePath is non-empty, the template is loaded from that file;
// otherwise, a built-in default template is used.
func NewGenerator(commits object.CommitIter, tagMap map[string]string, templatePath string) (*Generator, error) {
	g := &Generator{
		commits:     commits,
		tagMap:      tagMap,
		templateStr: defaultTemplate,
	}

	if templatePath != "" {
		content, err := os.ReadFile(templatePath) // #nosec G304
		if err != nil {
			return nil, fmt.Errorf("reading custom template file: %w", err)
		}
		g.templateStr = string(content)
	}

	return g, nil
}

// Generate processes the commits and returns the rendered changelog content
// and the calculated next version string.
func (g *Generator) Generate(requestedVersion string) (string, string, error) {
	groupedCommits := make(map[string][]model.CommitData, len(categories))
	var breakingChanges []model.CommitData

	lastVersion := "v0.0.0"
	var bumpType model.BumpType

	err := g.commits.ForEach(func(c *object.Commit) error {
		if tagName, exists := g.tagMap[c.Hash.String()]; exists {
			lastVersion = tagName
			return storer.ErrStop
		}

		data, ok := parser.ParseCommit(c.Message)
		if ok {
			bumpType = updateBumpType(bumpType, data)

			if data.IsBreaking {
				breakingChanges = append(breakingChanges, *data)
			}

			cat := categoryMap[data.Type]
			if cat == "" {
				cat = "Others"
			}
			groupedCommits[cat] = append(groupedCommits[cat], *data)
		} else {
			if bumpType < model.BumpPatch {
				bumpType = model.BumpPatch
			}
			addNonConventionalCommit(c.Message, groupedCommits)
		}
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("iterating commits: %w", err)
	}

	nextVersion := determineNextVersion(requestedVersion, lastVersion, bumpType)
	impactScore := calculateImpactScore(breakingChanges, groupedCommits)

	data := model.ChangelogData{
		Version:         nextVersion,
		Date:            time.Now().Format("2006-01-02"),
		ImpactScore:     impactScore,
		BreakingChanges: breakingChanges,
		Categories:      categories,
		GroupedCommits:  groupedCommits,
	}

	content, err := renderTemplate(g.templateStr, data)
	if err != nil {
		return "", "", err
	}

	return content, nextVersion, nil
}

// updateBumpType returns the higher of the current bump type and the one
// implied by the given commit data.
func updateBumpType(current model.BumpType, data *model.CommitData) model.BumpType {
	switch {
	case data.IsBreaking && current < model.BumpMajor:
		return model.BumpMajor
	case data.Type == "feat" && current < model.BumpMinor:
		return model.BumpMinor
	case current < model.BumpPatch:
		return model.BumpPatch
	default:
		return current
	}
}

// addNonConventionalCommit adds the first line of a non-conventional commit
// message to the "Others" category.
func addNonConventionalCommit(message string, grouped map[string][]model.CommitData) {
	firstLine, _, _ := strings.Cut(message, "\n")
	msg := strings.TrimSpace(firstLine)
	if msg != "" {
		grouped["Others"] = append(grouped["Others"], model.CommitData{
			Message: msg,
		})
	}
}

// calculateImpactScore computes the Release Impact Score.
// Formula: I = (Breaking * 50) + (Features * 10) + (Fixes + Refactors) * 1.
func calculateImpactScore(breaking []model.CommitData, grouped map[string][]model.CommitData) int {
	return (len(breaking) * 50) +
		(len(grouped["Features"]) * 10) +
		(len(grouped["Bug Fixes"]) + len(grouped["Refactor"]))
}

// determineNextVersion calculates the next semantic version based on the
// requested version, the last tagged version, and the bump type.
func determineNextVersion(requestedVersion, lastVersion string, bumpType model.BumpType) string {
	if requestedVersion != "vUnreleased" {
		return requestedVersion
	}

	var major, minor, patch int
	cleanVersion := strings.TrimPrefix(lastVersion, "v")
	if _, err := fmt.Sscanf(cleanVersion, "%d.%d.%d", &major, &minor, &patch); err != nil {
		major, minor, patch = 0, 0, 0
	}

	switch bumpType {
	case model.BumpMajor:
		major++
		minor = 0
		patch = 0
	case model.BumpMinor:
		minor++
		patch = 0
	case model.BumpPatch:
		patch++
	case model.BumpNone:
		// No version bump
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

// renderTemplate parses and executes the changelog template.
func renderTemplate(templateStr string, data model.ChangelogData) (string, error) {
	tmpl, err := template.New("changelog").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

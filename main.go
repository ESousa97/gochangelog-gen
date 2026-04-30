package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// ChangelogData holds the context for rendering the changelog template
type ChangelogData struct {
	Version         string
	Date            string
	ImpactScore     int
	BreakingChanges []CommitData
	Categories      []string
	GroupedCommits  map[string][]CommitData
}

const defaultTemplate = `# {{.Version}} ({{.Date}})
**Release Impact Score:** {{.ImpactScore}}

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

func createGitHubDraftRelease(repo *git.Repository, version string, changelogContent string) {
	remote, err := repo.Remote("origin")
	if err != nil {
		fmt.Printf("Error getting origin remote: %s\n", err)
		return
	}
	if len(remote.Config().URLs) == 0 {
		fmt.Println("No URLs found for origin remote")
		return
	}
	url := remote.Config().URLs[0]

	// Normalize to replace git@github.com: with https://github.com/
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)
	}
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		fmt.Println("Could not parse remote URL")
		return
	}
	owner := parts[len(parts)-2]
	repoName := parts[len(parts)-1]

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Error: GITHUB_TOKEN environment variable not set. Draft release will not be created.")
		return
	}

	payload := map[string]interface{}{
		"tag_name": version,
		"name":     version,
		"body":     changelogContent,
		"draft":    true,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %s\n", err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repoName), bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %s\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending HTTP request: %s\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("Successfully created draft release %s on GitHub!\n", version)
	} else {
		fmt.Printf("Failed to create GitHub release. Status Code: %d\n", resp.StatusCode)
	}
}

func main() {
	versionFlag := flag.String("version", "vUnreleased", "Version to be displayed in the changelog title")
	templateFlag := flag.String("template", "", "Path to a custom Markdown template file (.tmpl)")
	githubReleaseFlag := flag.Bool("github-release", false, "Create a draft release on GitHub")
	flag.Parse()

	// Open the repository in the current directory
	repo, err := git.PlainOpen(".")
	if err != nil {
		fmt.Printf("Error opening repository: %s\n", err)
		return
	}

	// Build map of commit hashes to tag names
	tagRefs, err := repo.Tags()
	tagMap := make(map[plumbing.Hash]string)
	if err == nil {
		tagRefs.ForEach(func(t *plumbing.Reference) error {
			// Resolve annotated tags
			tagObj, err := repo.TagObject(t.Hash())
			var commitHash plumbing.Hash
			if err == nil {
				commitHash = tagObj.Target
			} else {
				commitHash = t.Hash()
			}
			tagMap[commitHash] = t.Name().Short()
			return nil
		})
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

	lastVersion := "v0.0.0"
	bumpType := 0 // 0=None, 1=Patch, 2=Minor, 3=Major

	// Iterate over commits
	err = cIter.ForEach(func(c *object.Commit) error {
		if tagName, exists := tagMap[c.Hash]; exists {
			lastVersion = tagName
			return storer.ErrStop
		}

		data, ok := parseCommit(c.Message)
		if ok {
			if data.IsBreaking {
				breakingChanges = append(breakingChanges, *data)
				if bumpType < 3 {
					bumpType = 3
				}
			} else if data.Type == "feat" {
				if bumpType < 2 {
					bumpType = 2
				}
			} else {
				if bumpType < 1 {
					bumpType = 1
				}
			}

			cat, mapped := categoryMap[data.Type]
			if !mapped {
				cat = "Others"
			}
			groupedCommits[cat] = append(groupedCommits[cat], *data)
		} else {
			// Non-conventional commit or empty message
			if bumpType < 1 {
				bumpType = 1
			}
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

	var nextVersion string
	if *versionFlag != "vUnreleased" {
		nextVersion = *versionFlag
	} else {
		var major, minor, patch int
		cleanVersion := strings.TrimPrefix(lastVersion, "v")
		fmt.Sscanf(cleanVersion, "%d.%d.%d", &major, &minor, &patch)

		if bumpType == 3 {
			major++
			minor = 0
			patch = 0
		} else if bumpType == 2 {
			minor++
			patch = 0
		} else if bumpType == 1 {
			patch++
		}
		nextVersion = fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	}

	impactScore := (len(breakingChanges) * 50) + (len(groupedCommits["Features"]) * 10) + ((len(groupedCommits["Bug Fixes"]) + len(groupedCommits["Refactor"])) * 1)

	// Prepare data for template
	data := ChangelogData{
		Version:         nextVersion,
		Date:            time.Now().Format("2006-01-02"),
		ImpactScore:     impactScore,
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

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		fmt.Printf("Error executing template: %s\n", err)
		return
	}
	changelogContent := buf.String()

	// Open output file
	outFile, err := os.Create("CHANGELOG_PENDING.md")
	if err != nil {
		fmt.Printf("Error creating CHANGELOG_PENDING.md: %s\n", err)
		return
	}
	outFile.WriteString(changelogContent)
	outFile.Close()

	fmt.Println("Successfully generated CHANGELOG_PENDING.md")

	if *githubReleaseFlag {
		createGitHubDraftRelease(repo, nextVersion, changelogContent)
	}
}

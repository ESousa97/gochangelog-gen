// Package gitrepo provides an interface to interact with local Git repositories
// for extracting commit history, tags, and remote configuration.
package gitrepo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ErrRepositoryNotOpen is returned when a repository cannot be opened at the given path.
var ErrRepositoryNotOpen = errors.New("could not open repository")

// Repo represents a local Git repository.
type Repo struct {
	repository *git.Repository
}

// Open opens a Git repository at the specified path.
func Open(path string) (*Repo, error) {
	repository, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("%w at path %s: %w", ErrRepositoryNotOpen, path, err)
	}
	return &Repo{repository: repository}, nil
}

// GetTagMap returns a map connecting commit hashes to their short tag names.
// Both annotated and lightweight tags are resolved to their target commit hash.
func (r *Repo) GetTagMap() (map[plumbing.Hash]string, error) {
	tagRefs, err := r.repository.Tags()
	if err != nil {
		return nil, fmt.Errorf("getting tags: %w", err)
	}

	tagMap := make(map[plumbing.Hash]string)
	err = tagRefs.ForEach(func(t *plumbing.Reference) error {
		tagObj, tagErr := r.repository.TagObject(t.Hash())

		var commitHash plumbing.Hash
		if tagErr == nil {
			// Annotated tag: resolve to the target commit.
			commitHash = tagObj.Target
		} else {
			// Lightweight tag: the reference hash is the commit hash.
			commitHash = t.Hash()
		}

		tagMap[commitHash] = t.Name().Short()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	return tagMap, nil
}

// GetCommits returns an iterator for all commits starting from HEAD.
func (r *Repo) GetCommits() (object.CommitIter, error) {
	ref, err := r.repository.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}

	cIter, err := r.repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("getting commit log: %w", err)
	}

	return cIter, nil
}

// GetOriginOwnerRepo extracts the owner and repository name from the "origin"
// remote URL. It supports both HTTPS and SSH GitHub URLs.
func (r *Repo) GetOriginOwnerRepo() (string, string, error) {
	remote, err := r.repository.Remote("origin")
	if err != nil {
		return "", "", fmt.Errorf("getting origin remote: %w", err)
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", "", errors.New("no URLs found for origin remote")
	}

	return parseGitHubURL(urls[0])
}

// parseGitHubURL extracts owner and repo from a GitHub remote URL.
// Supports formats:
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo.git
func parseGitHubURL(rawURL string) (string, string, error) {
	url := rawURL

	// Strip known prefixes.
	for _, prefix := range []string{"git@github.com:", "https://github.com/"} {
		if strings.HasPrefix(url, prefix) {
			url = strings.TrimPrefix(url, prefix)
			break
		}
	}

	// Strip .git suffix.
	url = strings.TrimSuffix(url, ".git")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("could not parse owner and repo from URL: %s", rawURL)
	}

	return parts[len(parts)-2], parts[len(parts)-1], nil
}

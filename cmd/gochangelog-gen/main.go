// Package main is the entry point for the gochangelog-gen CLI tool.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/esousa97/gochangelog-gen/internal/changelog"
	"github.com/esousa97/gochangelog-gen/internal/github"
	"github.com/esousa97/gochangelog-gen/internal/gitrepo"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	versionFlag := flag.String("version", "vUnreleased", "Version to display in the changelog title")
	templateFlag := flag.String("template", "", "Path to a custom Markdown template file (.tmpl)")
	githubReleaseFlag := flag.Bool("github-release", false, "Create a draft release on GitHub")
	outputFlag := flag.String("output", "CHANGELOG_PENDING.md", "Output file path for the generated changelog")
	showVersion := flag.Bool("v", false, "Show application version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gochangelog-gen %s (built at %s)\n", version, buildTime)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	if err := run(ctx, *versionFlag, *templateFlag, *outputFlag, *githubReleaseFlag); err != nil {
		slog.Error("execution failed", slog.String("error", err.Error()))
		stop()
		os.Exit(1)
	}
	stop()
}

func run(ctx context.Context, targetVersion, templatePath, outputPath string, githubRelease bool) error {
	repo, err := gitrepo.Open(".")
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	tagMap, err := repo.GetTagMap()
	if err != nil {
		return fmt.Errorf("getting tags: %w", err)
	}

	strTagMap := make(map[string]string, len(tagMap))
	for hash, tag := range tagMap {
		strTagMap[hash.String()] = tag
	}

	commitsIter, err := repo.GetCommits()
	if err != nil {
		return fmt.Errorf("getting commits: %w", err)
	}

	generator, err := changelog.NewGenerator(commitsIter, strTagMap, templatePath)
	if err != nil {
		return fmt.Errorf("creating generator: %w", err)
	}

	content, nextVersion, err := generator.Generate(targetVersion)
	if err != nil {
		return fmt.Errorf("generating changelog: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing changelog file %s: %w", outputPath, err)
	}

	slog.Info("changelog generated", slog.String("file", outputPath), slog.String("version", nextVersion))

	if githubRelease {
		if err := createRelease(ctx, repo, nextVersion, content); err != nil {
			return err
		}
	}

	return nil
}

func createRelease(ctx context.Context, repo *gitrepo.Repo, version, content string) error {
	owner, repoName, err := repo.GetOriginOwnerRepo()
	if err != nil {
		return fmt.Errorf("determining GitHub repository: %w", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return errors.New("GITHUB_TOKEN environment variable not set; cannot create release")
	}

	client := github.NewClient(token)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	slog.Info("creating draft release on GitHub",
		slog.String("owner", owner),
		slog.String("repo", repoName),
		slog.String("version", version),
	)

	if err := client.CreateDraftRelease(ctx, owner, repoName, version, content); err != nil {
		return fmt.Errorf("creating github release: %w", err)
	}

	slog.Info("draft release created", slog.String("version", version))
	return nil
}

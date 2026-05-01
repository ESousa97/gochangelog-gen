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

type options struct {
	targetVersion string
	templatePath  string
	outputPath    string
	githubRelease bool
}

func main() {
	setupLogger()

	opts, shouldExit := parseFlags()
	if shouldExit {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, opts); err != nil {
		slog.Error("execution failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func setupLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func parseFlags() (options, bool) {
	var opts options
	var showVersion bool

	flag.StringVar(&opts.targetVersion, "version", "vUnreleased", "Version to display in the changelog title")
	flag.StringVar(&opts.templatePath, "template", "", "Path to a custom Markdown template file (.tmpl)")
	flag.BoolVar(&opts.githubRelease, "github-release", false, "Create a draft release on GitHub")
	flag.StringVar(&opts.outputPath, "output", "CHANGELOG_PENDING.md", "Output file path for the generated changelog")
	flag.BoolVar(&showVersion, "v", false, "Show application version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("gochangelog-gen %s (built at %s)\n", version, buildTime)
		return opts, true
	}

	return opts, false
}

func run(ctx context.Context, opts options) error {
	repo, err := gitrepo.Open(".")
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	strTagMap, err := getTagMap(repo)
	if err != nil {
		return err
	}

	commitsIter, err := repo.GetCommits()
	if err != nil {
		return fmt.Errorf("getting commits: %w", err)
	}

	generator, err := changelog.NewGenerator(commitsIter, strTagMap, opts.templatePath)
	if err != nil {
		return fmt.Errorf("creating generator: %w", err)
	}

	content, nextVersion, err := generator.Generate(opts.targetVersion)
	if err != nil {
		return fmt.Errorf("generating changelog: %w", err)
	}

	if err := os.WriteFile(opts.outputPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing changelog file %s: %w", opts.outputPath, err)
	}

	slog.Info("changelog generated", slog.String("file", opts.outputPath), slog.String("version", nextVersion))

	if opts.githubRelease {
		return createRelease(ctx, repo, nextVersion, content)
	}

	return nil
}

func getTagMap(repo *gitrepo.Repo) (map[string]string, error) {
	tagMap, err := repo.GetTagMap()
	if err != nil {
		return nil, fmt.Errorf("getting tags: %w", err)
	}

	strTagMap := make(map[string]string, len(tagMap))
	for hash, tag := range tagMap {
		strTagMap[hash.String()] = tag
	}
	return strTagMap, nil
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

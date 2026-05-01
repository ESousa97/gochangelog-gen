package gitrepo

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS URL with .git",
			url:       "https://github.com/esousa97/gochangelog-gen.git",
			wantOwner: "esousa97",
			wantRepo:  "gochangelog-gen",
		},
		{
			name:      "HTTPS URL without .git",
			url:       "https://github.com/esousa97/gochangelog-gen",
			wantOwner: "esousa97",
			wantRepo:  "gochangelog-gen",
		},
		{
			name:      "SSH URL with .git",
			url:       "git@github.com:esousa97/gochangelog-gen.git",
			wantOwner: "esousa97",
			wantRepo:  "gochangelog-gen",
		},
		{
			name:      "SSH URL without .git",
			url:       "git@github.com:esousa97/gochangelog-gen",
			wantOwner: "esousa97",
			wantRepo:  "gochangelog-gen",
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseGitHubURL() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGitHubURL() unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

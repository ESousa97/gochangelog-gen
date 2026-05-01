package changelog

import (
	"testing"

	"github.com/esousa97/gochangelog-gen/internal/model"
)

func TestDetermineNextVersion(t *testing.T) {
	tests := []struct {
		name             string
		requestedVersion string
		lastVersion      string
		bumpType         model.BumpType
		want             string
	}{
		{
			name:             "explicit version overrides bump",
			requestedVersion: "v2.0.0",
			lastVersion:      "v1.0.0",
			bumpType:         model.BumpPatch,
			want:             "v2.0.0",
		},
		{
			name:             "patch bump from v0.0.0",
			requestedVersion: "vUnreleased",
			lastVersion:      "v0.0.0",
			bumpType:         model.BumpPatch,
			want:             "v0.0.1",
		},
		{
			name:             "minor bump from v1.2.3",
			requestedVersion: "vUnreleased",
			lastVersion:      "v1.2.3",
			bumpType:         model.BumpMinor,
			want:             "v1.3.0",
		},
		{
			name:             "major bump from v1.2.3",
			requestedVersion: "vUnreleased",
			lastVersion:      "v1.2.3",
			bumpType:         model.BumpMajor,
			want:             "v2.0.0",
		},
		{
			name:             "no bump from v1.0.0",
			requestedVersion: "vUnreleased",
			lastVersion:      "v1.0.0",
			bumpType:         model.BumpNone,
			want:             "v1.0.0",
		},
		{
			name:             "malformed last version defaults to v0.0.0",
			requestedVersion: "vUnreleased",
			lastVersion:      "invalid",
			bumpType:         model.BumpMinor,
			want:             "v0.1.0",
		},
		{
			name:             "empty requested version is treated as explicit",
			requestedVersion: "",
			lastVersion:      "v1.0.0",
			bumpType:         model.BumpPatch,
			want:             "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineNextVersion(tt.requestedVersion, tt.lastVersion, tt.bumpType)
			if got != tt.want {
				t.Errorf("determineNextVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateBumpType(t *testing.T) {
	tests := []struct {
		name    string
		current model.BumpType
		data    *model.CommitData
		want    model.BumpType
	}{
		{
			name:    "breaking change sets major",
			current: model.BumpNone,
			data:    &model.CommitData{IsBreaking: true, Type: "feat"},
			want:    model.BumpMajor,
		},
		{
			name:    "feat sets minor",
			current: model.BumpNone,
			data:    &model.CommitData{Type: "feat"},
			want:    model.BumpMinor,
		},
		{
			name:    "fix sets patch",
			current: model.BumpNone,
			data:    &model.CommitData{Type: "fix"},
			want:    model.BumpPatch,
		},
		{
			name:    "major is not downgraded by feat",
			current: model.BumpMajor,
			data:    &model.CommitData{Type: "feat"},
			want:    model.BumpMajor,
		},
		{
			name:    "minor is not downgraded by fix",
			current: model.BumpMinor,
			data:    &model.CommitData{Type: "fix"},
			want:    model.BumpMinor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateBumpType(tt.current, tt.data)
			if got != tt.want {
				t.Errorf("updateBumpType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateImpactScore(t *testing.T) {
	breaking := []model.CommitData{{}, {}}
	grouped := map[string][]model.CommitData{
		"Features":  {{}, {}, {}},
		"Bug Fixes": {{}},
		"Refactor":  {{}, {}},
	}

	// (2 * 50) + (3 * 10) + (1 + 2) = 100 + 30 + 3 = 133
	got := calculateImpactScore(breaking, grouped)
	want := 133
	if got != want {
		t.Errorf("calculateImpactScore() = %d, want %d", got, want)
	}
}

func TestRenderTemplate(t *testing.T) {
	data := model.ChangelogData{
		Version:     "v1.0.0",
		Date:        "2026-01-01",
		ImpactScore: 10,
		Categories:  []string{"Features"},
		GroupedCommits: map[string][]model.CommitData{
			"Features": {{Type: "feat", Message: "add something"}},
		},
	}

	content, err := renderTemplate(defaultTemplate, data)
	if err != nil {
		t.Fatalf("renderTemplate() unexpected error: %v", err)
	}

	if content == "" {
		t.Fatal("renderTemplate() returned empty content")
	}

	// Check that key elements are present.
	for _, want := range []string{"v1.0.0", "2026-01-01", "Features", "add something"} {
		if !containsStr(content, want) {
			t.Errorf("rendered content missing %q", want)
		}
	}
}

func TestRenderTemplateInvalid(t *testing.T) {
	data := model.ChangelogData{}
	_, err := renderTemplate("{{.Invalid", data)
	if err == nil {
		t.Fatal("renderTemplate() expected error for invalid template, got nil")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

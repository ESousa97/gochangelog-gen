package parser_test

import (
	"testing"

	"github.com/esousa97/gochangelog-gen/internal/model"
	"github.com/esousa97/gochangelog-gen/internal/parser"
)

func TestParseCommit(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    *model.CommitData
		wantOk  bool
	}{
		{
			name:    "simple feat",
			message: "feat: add grouping",
			want: &model.CommitData{
				Type:    "feat",
				Message: "add grouping",
			},
			wantOk: true,
		},
		{
			name:    "feat with scope",
			message: "feat(cli): add grouping",
			want: &model.CommitData{
				Type:    "feat",
				Scope:   "cli",
				Message: "add grouping",
			},
			wantOk: true,
		},
		{
			name:    "breaking feat with bang",
			message: "feat!: breaking change",
			want: &model.CommitData{
				Type:       "feat",
				IsBreaking: true,
				Message:    "breaking change",
			},
			wantOk: true,
		},
		{
			name:    "breaking feat with scope and bang",
			message: "feat(api)!: breaking change",
			want: &model.CommitData{
				Type:       "feat",
				Scope:      "api",
				IsBreaking: true,
				Message:    "breaking change",
			},
			wantOk: true,
		},
		{
			name:    "breaking change footer with space",
			message: "feat: change api\n\nBREAKING CHANGE: the old api is gone",
			want: &model.CommitData{
				Type:                "feat",
				IsBreaking:          true,
				Message:             "change api",
				BreakingDescription: "the old api is gone",
			},
			wantOk: true,
		},
		{
			name:    "breaking change footer with dash",
			message: "fix: fix security\n\nBREAKING-CHANGE: security update required",
			want: &model.CommitData{
				Type:                "fix",
				IsBreaking:          true,
				Message:             "fix security",
				BreakingDescription: "security update required",
			},
			wantOk: true,
		},
		{
			name:    "simple fix",
			message: "fix: bug fix",
			want: &model.CommitData{
				Type:    "fix",
				Message: "bug fix",
			},
			wantOk: true,
		},
		{
			name:    "docs type",
			message: "docs(readme): update installation guide",
			want: &model.CommitData{
				Type:    "docs",
				Scope:   "readme",
				Message: "update installation guide",
			},
			wantOk: true,
		},
		{
			name:    "refactor type",
			message: "refactor: simplify error handling",
			want: &model.CommitData{
				Type:    "refactor",
				Message: "simplify error handling",
			},
			wantOk: true,
		},
		{
			name:    "not conventional",
			message: "random commit message",
			want:    nil,
			wantOk:  false,
		},
		{
			name:    "empty message",
			message: "",
			want:    nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parser.ParseCommit(tt.message)
			if ok != tt.wantOk {
				t.Fatalf("ParseCommit() ok = %v, wantOk %v", ok, tt.wantOk)
			}
			if tt.want == nil && got == nil {
				return
			}
			if tt.want == nil || got == nil {
				t.Fatalf("ParseCommit() = %+v, want %+v", got, tt.want)
			}
			if *got != *tt.want {
				t.Errorf("ParseCommit() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

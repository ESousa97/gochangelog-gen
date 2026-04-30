package main

import (
	"reflect"
	"testing"
)

func TestParseCommit(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		want     *CommitData
		wantOk   bool
	}{
		{
			name:    "simple feat",
			message: "feat: add grouping",
			want: &CommitData{
				Type:    "feat",
				Message: "add grouping",
			},
			wantOk: true,
		},
		{
			name:    "feat with scope",
			message: "feat(cli): add grouping",
			want: &CommitData{
				Type:    "feat",
				Scope:   "cli",
				Message: "add grouping",
			},
			wantOk: true,
		},
		{
			name:    "fix",
			message: "fix: bug fix",
			want: &CommitData{
				Type:    "fix",
				Message: "bug fix",
			},
			wantOk: true,
		},
		{
			name:    "not conventional",
			message: "random commit message",
			want:    nil,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCommit(tt.message)
			if ok != tt.wantOk {
				t.Errorf("parseCommit() ok = %v, wantOk %v", ok, tt.wantOk)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}

package repository

import (
	"context"
	"testing"

	"github.com/codeatlas/codeatlas/internal/model"
)

func TestAcquireLocalProjectDoesNotInventCommit(t *testing.T) {
	project := &model.Project{
		SourceType: model.SourceLocal,
		RootPath:   t.TempDir(),
	}

	commit, err := Acquire(context.Background(), project)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if commit != "" {
		t.Fatalf("Acquire returned local path as commit: %q", commit)
	}
}

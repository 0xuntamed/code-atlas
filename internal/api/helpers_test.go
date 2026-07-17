package api

import (
	"path/filepath"
	"testing"
)

func TestEvidenceWindow(t *testing.T) {
	tests := []struct {
		name              string
		entityStart       int
		entityEnd         int
		lineCount         int
		expectedStartLine int
		expectedEndLine   int
	}{
		{name: "adds context", entityStart: 10, entityEnd: 14, lineCount: 30, expectedStartLine: 7, expectedEndLine: 17},
		{name: "clamps start", entityStart: 1, entityEnd: 2, lineCount: 10, expectedStartLine: 1, expectedEndLine: 5},
		{name: "clamps end", entityStart: 8, entityEnd: 10, lineCount: 10, expectedStartLine: 5, expectedEndLine: 10},
		{name: "bounds missing range", entityStart: 0, entityEnd: 0, lineCount: 1_000, expectedStartLine: 1, expectedEndLine: 124},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			startLine, endLine := evidenceWindow(test.entityStart, test.entityEnd, test.lineCount)
			if startLine != test.expectedStartLine || endLine != test.expectedEndLine {
				t.Fatalf(
					"evidenceWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					test.entityStart,
					test.entityEnd,
					test.lineCount,
					startLine,
					endLine,
					test.expectedStartLine,
					test.expectedEndLine,
				)
			}
		})
	}
}

func TestLoopbackValidation(t *testing.T) {
	validHosts := []string{"localhost:7331", "127.0.0.1:7331", "[::1]:7331"}
	for _, host := range validHosts {
		if !isLoopbackHost(host) {
			t.Errorf("expected host %q to be accepted", host)
		}
	}
	if isLoopbackHost("example.com:7331") {
		t.Error("expected non-loopback host to be rejected")
	}
	if !isLoopbackOrigin("http://127.0.0.1:7331") {
		t.Error("expected loopback origin to be accepted")
	}
	if isLoopbackOrigin("https://example.com") {
		t.Error("expected remote origin to be rejected")
	}
}

func TestProjectPathHelpers(t *testing.T) {
	if name := gitProjectName("https://github.com/example/code-atlas.git"); name != "code-atlas" {
		t.Fatalf("gitProjectName returned %q", name)
	}

	root := t.TempDir()
	if !isWithin(root, filepath.Join(root, "managed", "clone")) {
		t.Error("expected descendant path to be within root")
	}
	if isWithin(root, filepath.Join(root, "..", "outside")) {
		t.Error("expected parent path to be outside root")
	}
}

package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverClassifiesWithoutPersistingContent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/main.ts", "export function main() { return 42 }")
	writeTestFile(t, root, "src/main.test.ts", "test('main', () => main())")
	writeTestFile(t, root, "node_modules/pkg/index.js", "module.exports = secret")
	writeTestFile(t, root, ".env.local", "TOKEN=super-secret-value")
	writeTestFile(t, root, "README.md", "docs")
	writeTestFile(t, root, ".codeatlasignore", "!dist/\n")
	writeTestFile(t, root, "dist/reincluded.py", "def visible():\n    return True")

	items, err := Discover(t.Context(), "project", "run", root)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]DiscoveredFile{}
	for _, item := range items {
		byPath[item.Record.Path] = item
	}
	if byPath["src/main.ts"].Record.Classification != "source" {
		t.Fatalf("source was not classified: %#v", byPath["src/main.ts"].Record)
	}
	if !byPath["src/main.test.ts"].Record.IsTest {
		t.Fatal("test source was not retained as a test dependent")
	}
	if byPath["node_modules"].Record.IgnoreReason == "" {
		t.Fatal("dependency directory was not inventoried")
	}
	if !strings.Contains(byPath[".env.local"].Record.IgnoreReason, "secret-like") {
		t.Fatalf("secret file reason: %q", byPath[".env.local"].Record.IgnoreReason)
	}
	if byPath["dist/reincluded.py"].Record.Classification != "source" {
		t.Fatal(".codeatlasignore negation did not reinclude the default directory")
	}
	for _, item := range items {
		if strings.Contains(item.Record.Path, "super-secret-value") || strings.Contains(item.Record.IgnoreReason, "super-secret-value") {
			t.Fatal("file content leaked into inventory metadata")
		}
	}
}

func writeTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

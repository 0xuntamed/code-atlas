package repository

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// LineRange is a 1-based, inclusive span of lines on the new (working-tree) side
// of a diff.
type LineRange struct {
	Start int
	End   int
}

// FileChange describes one file's changes in a working-tree diff: the new-side
// line ranges that changed, and whether the file was added or deleted outright.
type FileChange struct {
	Path    string
	Ranges  []LineRange
	Added   bool
	Deleted bool
}

// hunkHeader captures the new-side start line and optional line count from a
// unified-diff hunk header, e.g. "@@ -12,3 +14,6 @@" → start=14, count=6.
var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// WorkingTreeDiff returns the uncommitted changes (working tree vs HEAD) of a
// local Git repository as new-side line ranges per file. It requires a real
// working tree, so it is meaningless for shallow managed clones.
func WorkingTreeDiff(ctx context.Context, root string) ([]FileChange, error) {
	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.quotePath=false",
		"-C", root,
		"diff", "--unified=0", "--no-color", "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return nil, fmt.Errorf("read working-tree diff: %s", safeGitError([]byte(stderr)))
	}
	return parseUnifiedDiff(string(out)), nil
}

// parseUnifiedDiff turns `git diff --unified=0` output into per-file changes.
// Kept free of any I/O so it can be unit-tested directly against fixture text.
func parseUnifiedDiff(text string) []FileChange {
	files := make([]FileChange, 0)
	var current *FileChange
	oldPath := ""

	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			files = append(files, FileChange{})
			current = &files[len(files)-1]
			oldPath = ""
		case current == nil:
			continue
		case strings.HasPrefix(line, "--- "):
			path := strings.TrimPrefix(line, "--- ")
			if path == "/dev/null" {
				current.Added = true
				oldPath = ""
			} else {
				oldPath = stripDiffPrefix(path)
			}
		case strings.HasPrefix(line, "+++ "):
			path := strings.TrimPrefix(line, "+++ ")
			if path == "/dev/null" {
				current.Deleted = true
				current.Path = oldPath
			} else {
				current.Path = stripDiffPrefix(path)
			}
		case strings.HasPrefix(line, "@@"):
			match := hunkHeader.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			start, _ := strconv.Atoi(match[1])
			count := 1
			if match[2] != "" {
				count, _ = strconv.Atoi(match[2])
			}
			if count <= 0 {
				// A pure deletion has no new-side lines; anchor it at `start`.
				current.Ranges = append(current.Ranges, LineRange{Start: max(1, start), End: max(1, start)})
			} else {
				current.Ranges = append(current.Ranges, LineRange{Start: start, End: start + count - 1})
			}
		}
	}

	// Drop entries that never resolved a path (e.g. pure mode-change stanzas).
	resolved := files[:0]
	for _, file := range files {
		if file.Path != "" {
			resolved = append(resolved, file)
		}
	}
	return resolved
}

func stripDiffPrefix(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		return path[2:]
	}
	return path
}

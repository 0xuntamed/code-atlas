package repository

import (
	"bufio"
	"os"
	"path"
	"regexp"
	"strings"
)

type ignoreRule struct {
	pattern string
	ignore  bool
	reason  string
	re      *regexp.Regexp
}

type ignoreMatcher struct{ rules []ignoreRule }

var defaultPatterns = []string{
	".git/", ".hg/", ".svn/", "node_modules/", "vendor/", ".venv/", "venv/", "__pypackages__/",
	"dist/", "build/", "out/", "target/", "bin/", "obj/", ".next/", "coverage/", "htmlcov/",
	".cache/", ".pytest_cache/", ".mypy_cache/", ".ruff_cache/", "__pycache__/", ".turbo/", ".pnpm-store/",
	"package-lock.json", "pnpm-lock.yaml", "yarn.lock", "go.sum", "Pipfile.lock", "poetry.lock", "uv.lock",
	"*.map", "*.min.js", "*.min.css",
}

func loadIgnoreMatcher(root string) ignoreMatcher {
	m := ignoreMatcher{}
	for _, p := range defaultPatterns {
		m.add(p, true, "default exclusion")
	}
	m.loadFile(pathFor(root, ".gitignore"), ".gitignore")
	m.loadFile(pathFor(root, ".codeatlasignore"), ".codeatlasignore")
	return m
}

func (m *ignoreMatcher) loadFile(filePath, reason string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ignore := true
		if strings.HasPrefix(line, "!") {
			ignore = false
			line = strings.TrimPrefix(line, "!")
		}
		m.add(line, ignore, reason)
	}
}

func (m *ignoreMatcher) add(pattern string, ignore bool, reason string) {
	pattern = strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
	if pattern == "" {
		return
	}
	if re, err := regexp.Compile(globRegex(pattern)); err == nil {
		m.rules = append(m.rules, ignoreRule{pattern: pattern, ignore: ignore, reason: reason, re: re})
	}
}

func (m ignoreMatcher) match(relative string, isDir bool) (bool, string) {
	relative = strings.TrimPrefix(strings.ReplaceAll(relative, "\\", "/"), "./")
	probe := relative
	if isDir && !strings.HasSuffix(probe, "/") {
		probe += "/"
	}
	ignored := false
	reason := ""
	for _, rule := range m.rules {
		if rule.re.MatchString(probe) {
			ignored = rule.ignore
			if ignored {
				reason = rule.reason + ": " + rule.pattern
			} else {
				reason = ""
			}
		}
	}
	return ignored, reason
}

func globRegex(pattern string) string {
	anchored := strings.HasPrefix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "/")
	directory := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	var b strings.Builder
	if anchored || strings.Contains(pattern, "/") {
		b.WriteString("^")
	} else {
		b.WriteString("(^|.*/)")
	}
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			b.WriteByte('\\')
			b.WriteByte(pattern[i])
		default:
			b.WriteByte(pattern[i])
		}
	}
	if directory {
		b.WriteString("(/.*)?/?$")
	} else {
		b.WriteString("$")
	}
	return b.String()
}

func pathFor(root, name string) string { return path.Join(strings.ReplaceAll(root, "\\", "/"), name) }

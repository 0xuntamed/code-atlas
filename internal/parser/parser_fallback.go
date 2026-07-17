//go:build !cgo

package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/codeatlas/codeatlas/internal/model"
)

var (
	jsFunction  = regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)`)
	jsClass     = regexp.MustCompile(`(?:export\s+)?class\s+([A-Za-z_$][\w$]*)`)
	jsArrow     = regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>`)
	jsImport    = regexp.MustCompile(`(?:from\s+|require\s*\(\s*|import\s*\(\s*)["']([^"']+)["']`)
	jsRoute     = regexp.MustCompile(`(?i)(?:app|router|server|api)\.(get|post|put|patch|delete|options|head|use)\s*\(\s*["']([^"']+)["']\s*,\s*([^)]*)\)`)
	goFunction  = regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(`)
	goType      = regexp.MustCompile(`^\s*type\s+([A-Za-z_]\w*)\s+(struct|interface)\b`)
	goImport    = regexp.MustCompile(`["']([^"']+)["']`)
	goRoute     = regexp.MustCompile(`(?i)(?:http\.HandleFunc|(?:router|engine)\.(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD))\s*\(\s*["']([^"']+)["']\s*,\s*([^,)]+)`)
	pyFunction  = regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(`)
	pyClass     = regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)`)
	pyRoute     = regexp.MustCompile(`(?i)^\s*@(?:app|router|api)\.(get|post|put|patch|delete|options|head)\s*\(\s*["']([^"']+)["']`)
	callPattern = regexp.MustCompile(`([A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*)\s*\(`)
)

func Parse(filePath, language string, source []byte) (ParseResult, error) {
	result := ParseResult{Path: filepath.ToSlash(filePath), Language: language, FileKey: "file"}
	lines := strings.Split(string(source), "\n")
	switch language {
	case "javascript", "typescript":
		parseJSLines(lines, &result)
	case "go":
		parseGoLines(lines, &result)
	case "python":
		parsePythonLines(lines, &result)
	default:
		return ParseResult{}, fmt.Errorf("unsupported language %q", language)
	}
	return result, nil
}

func parseJSLines(lines []string, result *ParseResult) {
	current := "file"
	depth := 0
	scopeDepth := 0
	for index, line := range lines {
		r := lineRange(index, line)
		skipCalls := map[string]bool{"function": true, "if": true, "for": true, "while": true, "switch": true, "require": true}
		routeLine := false
		if match := jsClass.FindStringSubmatch(line); len(match) > 0 {
			seed := fallbackEntity(result, "class", match[1], r)
			result.Entities = append(result.Entities, seed)
			current = seed.Key
			scopeDepth = depth + strings.Count(line, "{")
		}
		if match := jsFunction.FindStringSubmatch(line); len(match) > 0 {
			seed := fallbackEntity(result, "function", match[1], r)
			result.Entities = append(result.Entities, seed)
			current = seed.Key
			skipCalls[match[1]] = true
			scopeDepth = depth + strings.Count(line, "{")
		}
		if match := jsArrow.FindStringSubmatch(line); len(match) > 0 {
			seed := fallbackEntity(result, "function", match[1], r)
			result.Entities = append(result.Entities, seed)
			current = seed.Key
			skipCalls[match[1]] = true
			scopeDepth = depth + strings.Count(line, "{")
		}
		for _, match := range jsImport.FindAllStringSubmatch(line, -1) {
			result.Imports = append(result.Imports, ImportSeed{FromKey: current, Specifier: match[1], Range: r})
		}
		if match := jsRoute.FindStringSubmatch(line); len(match) > 0 {
			routeLine = true
			handlers := splitHandlers(match[3])
			addFallbackRoute(result, strings.ToUpper(match[1]), match[2], handlers, r)
		}
		if !routeLine {
			addFallbackCalls(line, current, r, result, skipCalls)
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if current != "file" && depth < scopeDepth {
			current = "file"
		}
	}
	addFallbackNextRoutes(result)
}

func parseGoLines(lines []string, result *ParseResult) {
	current := "file"
	depth := 0
	scopeDepth := 0
	inImportBlock := false
	for index, line := range lines {
		r := lineRange(index, line)
		trimmed := strings.TrimSpace(line)
		skipCalls := map[string]bool{"func": true, "if": true, "for": true, "switch": true, "select": true, "make": true, "append": true, "len": true}
		routeLine := false
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
		}
		if inImportBlock || strings.HasPrefix(trimmed, "import ") {
			for _, match := range goImport.FindAllStringSubmatch(line, -1) {
				result.Imports = append(result.Imports, ImportSeed{FromKey: "file", Specifier: match[1], Range: r})
			}
		}
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
		}
		if match := goType.FindStringSubmatch(line); len(match) > 0 {
			seed := fallbackEntity(result, match[2], match[1], r)
			result.Entities = append(result.Entities, seed)
		}
		if match := goFunction.FindStringSubmatch(line); len(match) > 0 {
			kind := "function"
			if strings.Contains(line, "func (") {
				kind = "method"
			}
			seed := fallbackEntity(result, kind, match[1], r)
			result.Entities = append(result.Entities, seed)
			current = seed.Key
			skipCalls[match[1]] = true
			scopeDepth = depth + strings.Count(line, "{")
		}
		if match := goRoute.FindStringSubmatch(line); len(match) > 0 {
			routeLine = true
			method := strings.ToUpper(match[1])
			if method == "" {
				method = "ANY"
			}
			addFallbackRoute(result, method, match[2], []string{strings.TrimSpace(match[3])}, r)
		}
		if !routeLine {
			addFallbackCalls(line, current, r, result, skipCalls)
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if current != "file" && depth < scopeDepth {
			current = "file"
		}
	}
}

func parsePythonLines(lines []string, result *ParseResult) {
	type scope struct {
		key    string
		indent int
	}
	stack := []scope{{key: "file", indent: -1}}
	pendingMethod, pendingPath := "", ""
	for index, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		for len(stack) > 1 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}
		current := stack[len(stack)-1].key
		r := lineRange(index, line)
		if match := pyRoute.FindStringSubmatch(line); len(match) > 0 {
			pendingMethod, pendingPath = strings.ToUpper(match[1]), match[2]
			continue
		}
		if match := pyClass.FindStringSubmatch(line); len(match) > 0 {
			seed := fallbackEntity(result, "class", match[1], r)
			result.Entities = append(result.Entities, seed)
			stack = append(stack, scope{seed.Key, indent})
			continue
		}
		if match := pyFunction.FindStringSubmatch(line); len(match) > 0 {
			kind := "function"
			if len(stack) > 1 {
				kind = "method"
			}
			seed := fallbackEntity(result, kind, match[1], r)
			result.Entities = append(result.Entities, seed)
			if pendingPath != "" {
				addFallbackRoute(result, pendingMethod, pendingPath, []string{seed.Name}, r)
				pendingMethod, pendingPath = "", ""
			}
			stack = append(stack, scope{seed.Key, indent})
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "from ") {
			parts := strings.Fields(trimmed)
			if len(parts) > 1 {
				result.Imports = append(result.Imports, ImportSeed{FromKey: current, Specifier: parts[1], Range: r})
			}
		}
		if strings.HasPrefix(trimmed, "import ") {
			imports := strings.Split(strings.TrimPrefix(trimmed, "import "), ",")
			for _, item := range imports {
				result.Imports = append(result.Imports, ImportSeed{FromKey: current, Specifier: strings.TrimSpace(strings.Split(item, " as ")[0]), Range: r})
			}
		}
		addFallbackCalls(line, current, r, result, map[string]bool{"def": true, "if": true, "for": true, "while": true, "print": true, "len": true, "super": true})
	}
}

func fallbackEntity(result *ParseResult, kind, name string, r model.Range) EntitySeed {
	qualified := result.Path + "::" + name
	return EntitySeed{Key: fmt.Sprintf("%s|%s|%d", kind, qualified, r.StartLine), Kind: kind, Name: name, QualifiedName: qualified, Range: r, Metadata: map[string]any{"parser": "structural-fallback"}}
}
func addFallbackRoute(result *ParseResult, method, path string, handlers []string, r model.Range) {
	name := method + " " + path
	route := EntitySeed{Key: fmt.Sprintf("route|%s|%d", name, r.StartLine), Kind: "route", Name: name, QualifiedName: result.Path + "::" + name, Range: r, Metadata: map[string]any{"method": method, "path": path, "parser": "structural-fallback"}}
	result.Entities = append(result.Entities, route)
	for i, handler := range handlers {
		handler = strings.TrimSpace(handler)
		if handler == "" {
			continue
		}
		kind := "uses_middleware"
		if i == len(handlers)-1 {
			kind = "handles_route"
		}
		result.References = append(result.References, ReferenceSeed{FromKey: route.Key, Target: handler, Kind: kind, Range: r, Confidence: .72})
	}
}
func addFallbackCalls(line, current string, r model.Range, result *ParseResult, skip map[string]bool) {
	for _, match := range callPattern.FindAllStringSubmatch(line, -1) {
		target := match[1]
		if skip[target] || strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		result.References = append(result.References, ReferenceSeed{FromKey: current, Target: target, Kind: "calls", Range: r, Confidence: .48})
	}
}
func splitHandlers(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
func lineRange(index int, line string) model.Range {
	return model.Range{StartLine: index + 1, StartColumn: 1, EndLine: index + 1, EndColumn: len(line) + 1}
}
func addFallbackNextRoutes(result *ParseResult) {
	lower := strings.ToLower(result.Path)
	if !strings.Contains(lower, "app/") || !strings.Contains(lower, "/route.") {
		return
	}
	idx := strings.Index(lower, "app/")
	routePath := result.Path[idx+4:]
	routePath = routePath[:strings.LastIndex(strings.ToLower(routePath), "/route.")]
	routePath = "/" + routePath
	for _, entity := range append([]EntitySeed(nil), result.Entities...) {
		method := strings.ToUpper(entity.Name)
		if entity.Kind == "function" && (method == "GET" || method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE") {
			addFallbackRoute(result, method, routePath, []string{entity.Name}, entity.Range)
		}
	}
}

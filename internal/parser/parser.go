//go:build cgo

package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/codeatlas/codeatlas/internal/model"
)

var routeMethodPattern = regexp.MustCompile(`(?i)^(get|post|put|patch|delete|options|head|use)$`)
var pythonRoutePattern = regexp.MustCompile(
	`(?i)@(?:app|router|api)\.` +
		`(get|post|put|patch|delete|options|head)\s*\(\s*["']([^"']+)["']`,
)

func Parse(path, language string, source []byte) (ParseResult, error) {
	p := treesitter.NewParser()
	defer p.Close()
	var languagePointer *treesitter.Language
	switch language {
	case "javascript":
		languagePointer = treesitter.NewLanguage(tree_sitter_javascript.Language())
	case "typescript":
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".tsx" {
			languagePointer = treesitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
		} else {
			languagePointer = treesitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
		}
	case "go":
		languagePointer = treesitter.NewLanguage(tree_sitter_go.Language())
	case "python":
		languagePointer = treesitter.NewLanguage(tree_sitter_python.Language())
	default:
		return ParseResult{}, fmt.Errorf("unsupported language %q", language)
	}
	if err := p.SetLanguage(languagePointer); err != nil {
		return ParseResult{}, fmt.Errorf("configure %s parser: %w", language, err)
	}
	tree := p.Parse(source, nil)
	if tree == nil {
		return ParseResult{}, fmt.Errorf("parser returned no syntax tree")
	}
	defer tree.Close()
	result := ParseResult{
		Path:            filepath.ToSlash(path),
		Language:        language,
		FileKey:         "file",
		HasSyntaxErrors: tree.RootNode().HasError(),
	}
	switch language {
	case "javascript", "typescript":
		walkJS(tree.RootNode(), source, &result, nil)
	case "go":
		walkGo(tree.RootNode(), source, &result, nil)
	case "python":
		walkPython(tree.RootNode(), source, &result, nil, nil)
	}
	addNextRoutes(&result)
	return result, nil
}

func declaration(node *treesitter.Node, source []byte, result *ParseResult, scope []string, kind string) *EntitySeed {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nodeText(nameNode, source)
	if name == "" {
		return nil
	}
	parts := []string{result.Path}
	for _, key := range scope {
		if e := entityByKey(result, key); e != nil {
			parts = append(parts, e.Name)
		}
	}
	parts = append(parts, name)
	qualified := strings.Join(parts, "::")
	r := nodeRange(node)
	key := fmt.Sprintf("%s|%s|%d", kind, qualified, r.StartLine)
	return &EntitySeed{Key: key, Kind: kind, Name: name, QualifiedName: qualified, Range: r, Metadata: map[string]any{}}
}

func addRoute(result *ParseResult, method, path string, args *treesitter.Node, source []byte, r model.Range) {
	route := newRoute(result, method, path, r)
	count := args.NamedChildCount()
	for i := uint(1); i < count; i++ {
		target := nodeText(args.NamedChild(i), source)
		if target == "" {
			continue
		}
		kind := "uses_middleware"
		if i == count-1 {
			kind = "handles_route"
		}
		result.References = append(result.References, ReferenceSeed{
			FromKey:    route.Key,
			Target:     target,
			Kind:       kind,
			Range:      r,
			Confidence: 0.95,
		})
	}
}

func newRoute(result *ParseResult, method, path string, r model.Range) EntitySeed {
	name := strings.ToUpper(method) + " " + path
	key := "route|" + name + "|" + fmt.Sprint(r.StartLine)
	for _, existing := range result.Entities {
		if existing.Key == key {
			return existing
		}
	}
	route := EntitySeed{
		Key:           key,
		Kind:          "route",
		Name:          name,
		QualifiedName: result.Path + "::" + name,
		Range:         r,
		Metadata: map[string]any{
			"method": strings.ToUpper(method),
			"path":   path,
		},
	}
	result.Entities = append(result.Entities, route)
	return route
}

func addNextRoutes(result *ParseResult) {
	p := filepath.ToSlash(result.Path)
	lower := strings.ToLower(p)
	if result.Language != "typescript" && result.Language != "javascript" {
		return
	}
	if !strings.Contains(lower, "/app/") && !strings.HasPrefix(lower, "app/") {
		return
	}
	if !strings.Contains(lower, "/route.") {
		return
	}
	idx := strings.Index(lower, "app/")
	routePath := p[idx+4:]
	routePath = routePath[:strings.LastIndex(strings.ToLower(routePath), "/route.")]
	segments := strings.Split(routePath, "/")
	for i, s := range segments {
		if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
			segments[i] = ":" + strings.Trim(s, "[]")
		}
	}
	routePath = "/" + strings.Join(segments, "/")
	for _, entity := range append([]EntitySeed(nil), result.Entities...) {
		if entity.Kind == "function" && routeMethodPattern.MatchString(entity.Name) {
			route := newRoute(result, strings.ToUpper(entity.Name), routePath, entity.Range)
			result.References = append(result.References, ReferenceSeed{
				FromKey:    route.Key,
				Target:     entity.Name,
				Kind:       "handles_route",
				Range:      entity.Range,
				Confidence: 1,
			})
		}
	}
}

func splitCallee(callee string) (method, receiver string) {
	parts := strings.Split(strings.TrimSpace(callee), ".")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[len(parts)-1], strings.Join(parts[:len(parts)-1], ".")
}
func isRouteReceiver(receiver string) bool {
	r := strings.ToLower(receiver)
	switch r {
	case "app", "router", "server", "api":
		return true
	default:
		return strings.HasSuffix(r, "router") || strings.HasSuffix(r, "engine")
	}
}
func currentKey(scope []string) string {
	if len(scope) == 0 {
		return "file"
	}
	return scope[len(scope)-1]
}
func entityByKey(result *ParseResult, key string) *EntitySeed {
	for i := range result.Entities {
		if result.Entities[i].Key == key {
			return &result.Entities[i]
		}
	}
	return nil
}
func nodeText(node *treesitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	start, end := int(node.StartByte()), int(node.EndByte())
	if start < 0 || end > len(source) || start > end {
		return ""
	}
	return strings.TrimSpace(string(source[start:end]))
}
func nodeRange(node *treesitter.Node) model.Range {
	start, end := node.StartPosition(), node.EndPosition()
	return model.Range{
		StartLine:   int(start.Row) + 1,
		StartColumn: int(start.Column) + 1,
		EndLine:     int(end.Row) + 1,
		EndColumn:   int(end.Column) + 1,
	}
}
func trimLiteral(value string) string { return strings.Trim(strings.TrimSpace(value), "`\"'") }
func pythonImports(text string) []string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "from ") {
		parts := strings.Fields(text)
		if len(parts) >= 2 {
			return []string{parts[1]}
		}
	}
	text = strings.TrimPrefix(text, "import ")
	items := strings.Split(text, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(strings.Split(item, " as ")[0])
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

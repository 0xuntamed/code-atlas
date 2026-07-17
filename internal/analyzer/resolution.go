package analyzer

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/codeatlas/codeatlas/internal/model"
	parserpkg "github.com/codeatlas/codeatlas/internal/parser"
)

func (b *graphBuilder) resolveImportedFile(
	result parserpkg.ParseResult,
	specifier string,
) (string, string, float64) {
	targetPath := resolveImport(
		result.Path,
		result.Language,
		specifier,
		b.fileEntityIDs,
	)
	if targetPath != "" {
		b.importsByFile[result.Path] = append(b.importsByFile[result.Path], targetPath)
		return b.fileEntityIDs[targetPath], "resolved", 0.88
	}
	return b.externalEntity(specifier), "external", 0.7
}

func indexEntity(
	entity model.Entity,
	byName map[string][]model.Entity,
	byQualified map[string]model.Entity,
) {
	name := normalizeTarget(entity.Name)
	byName[name] = append(byName[name], entity)
	byQualified[normalizeTarget(entity.QualifiedName)] = entity
}

func resolveReference(
	target string,
	currentPath string,
	imports []string,
	byName map[string][]model.Entity,
	byQualified map[string]model.Entity,
) (string, string, float64) {
	normalized := normalizeTarget(target)
	if entity, ok := byQualified[normalized]; ok {
		return entity.ID, "resolved", 1
	}

	candidates := byName[normalized]
	if len(candidates) == 0 {
		return "", "", 0
	}
	for _, candidate := range candidates {
		if candidate.QualifiedName == currentPath ||
			strings.HasPrefix(candidate.QualifiedName, currentPath+"::") {
			return candidate.ID, "resolved", 0.95
		}
	}
	for _, imported := range imports {
		for _, candidate := range candidates {
			if candidate.QualifiedName == imported ||
				strings.HasPrefix(candidate.QualifiedName, imported+"::") {
				return candidate.ID, "resolved", 0.9
			}
		}
	}
	if len(candidates) == 1 {
		return candidates[0].ID, "inferred", 0.72
	}
	return "", "", 0
}

func resolveImport(
	currentPath string,
	language string,
	specifier string,
	files map[string]string,
) string {
	specifier = strings.TrimSpace(strings.ReplaceAll(specifier, "\\", "/"))
	if specifier == "" {
		return ""
	}

	base := path.Dir(currentPath)
	candidates := make([]string, 0)
	switch language {
	case "javascript", "typescript":
		candidates = javascriptImportCandidates(base, specifier)
	case "python":
		candidates = pythonImportCandidates(base, specifier)
	case "go":
		candidates = goImportCandidates(specifier, files)
	}

	for _, candidate := range candidates {
		candidate = filepath.ToSlash(path.Clean(candidate))
		if _, ok := files[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func javascriptImportCandidates(base, specifier string) []string {
	if !strings.HasPrefix(specifier, ".") {
		return nil
	}
	root := path.Clean(path.Join(base, specifier))
	candidates := []string{root}
	for _, extension := range []string{
		".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs",
	} {
		candidates = append(candidates, root+extension, path.Join(root, "index"+extension))
	}
	return candidates
}

func pythonImportCandidates(base, specifier string) []string {
	if !strings.HasPrefix(specifier, ".") {
		module := strings.ReplaceAll(specifier, ".", "/")
		return []string{module + ".py", path.Join(module, "__init__.py")}
	}

	dots := len(specifier) - len(strings.TrimLeft(specifier, "."))
	root := base
	for index := 1; index < dots; index++ {
		root = path.Dir(root)
	}
	module := strings.ReplaceAll(strings.TrimLeft(specifier, "."), ".", "/")
	return []string{
		path.Join(root, module) + ".py",
		path.Join(root, module, "__init__.py"),
	}
}

func goImportCandidates(specifier string, files map[string]string) []string {
	lastSegment := path.Base(specifier)
	candidates := make([]string, 0)
	for filePath := range files {
		if path.Base(path.Dir(filePath)) == lastSegment {
			candidates = append(candidates, filePath)
		}
	}
	return candidates
}

func normalizeTarget(value string) string {
	normalized := strings.TrimSpace(value)
	normalized = strings.TrimPrefix(normalized, "await ")
	normalized = strings.TrimPrefix(normalized, "new ")
	if index := strings.IndexAny(normalized, "([<"); index >= 0 {
		normalized = normalized[:index]
	}
	parts := strings.FieldsFunc(normalized, func(character rune) bool {
		return character == '.' || character == ':' || character == '/'
	})
	if len(parts) > 0 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return strings.ToLower(normalized)
}

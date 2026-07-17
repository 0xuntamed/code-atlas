//go:build cgo

package parser

import (
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func walkPython(
	node *treesitter.Node,
	source []byte,
	result *ParseResult,
	scope []string,
	decorators []string,
) {
	if node.Kind() == "decorated_definition" {
		walkDecoratedPythonDefinition(node, source, result, scope)
		return
	}

	nextScope := scope
	var entity *EntitySeed
	switch node.Kind() {
	case "function_definition":
		kind := "function"
		if len(scope) > 0 {
			kind = "method"
		}
		entity = declaration(node, source, result, scope, kind)
	case "class_definition":
		entity = declaration(node, source, result, scope, "class")
	case "import_statement", "import_from_statement":
		addPythonImports(node, source, result, scope)
	case "call":
		addPythonCall(node, source, result, scope)
	}

	if entity != nil {
		result.Entities = append(result.Entities, *entity)
		nextScope = append(scope, entity.Key)
		addPythonRoutes(result, entity, decorators)
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if child := node.NamedChild(index); child != nil {
			walkPython(child, source, result, nextScope, nil)
		}
	}
}

func walkDecoratedPythonDefinition(
	node *treesitter.Node,
	source []byte,
	result *ParseResult,
	scope []string,
) {
	decorators := make([]string, 0)
	var definition *treesitter.Node
	for index := uint(0); index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child.Kind() == "decorator" {
			decorators = append(decorators, nodeText(child, source))
		} else if strings.HasSuffix(child.Kind(), "definition") {
			definition = child
		}
	}
	if definition != nil {
		walkPython(definition, source, result, scope, decorators)
	}
}

func addPythonImports(
	node *treesitter.Node,
	source []byte,
	result *ParseResult,
	scope []string,
) {
	for _, specifier := range pythonImports(nodeText(node, source)) {
		result.Imports = append(result.Imports, ImportSeed{
			FromKey:   currentKey(scope),
			Specifier: specifier,
			Range:     nodeRange(node),
		})
	}
}

func addPythonCall(
	node *treesitter.Node,
	source []byte,
	result *ParseResult,
	scope []string,
) {
	function := node.ChildByFieldName("function")
	if function == nil {
		return
	}
	result.References = append(result.References, ReferenceSeed{
		FromKey:    currentKey(scope),
		Target:     nodeText(function, source),
		Kind:       "calls",
		Range:      nodeRange(node),
		Confidence: 0.7,
	})
}

func addPythonRoutes(result *ParseResult, entity *EntitySeed, decorators []string) {
	for _, decorator := range decorators {
		match := pythonRoutePattern.FindStringSubmatch(decorator)
		if len(match) != 3 {
			continue
		}
		route := newRoute(result, strings.ToUpper(match[1]), match[2], entity.Range)
		result.References = append(result.References, ReferenceSeed{
			FromKey:    route.Key,
			Target:     entity.Name,
			Kind:       "handles_route",
			Range:      entity.Range,
			Confidence: 1,
		})
	}
}

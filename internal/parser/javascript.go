//go:build cgo

package parser

import (
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func walkJS(node *treesitter.Node, source []byte, result *ParseResult, scope []string) {
	nextScope := scope
	var entity *EntitySeed
	switch node.Kind() {
	case "function_declaration", "generator_function_declaration":
		entity = declaration(node, source, result, scope, "function")
	case "class_declaration":
		entity = declaration(node, source, result, scope, "class")
	case "method_definition":
		entity = declaration(node, source, result, scope, "method")
	case "variable_declarator":
		value := node.ChildByFieldName("value")
		if value != nil && (value.Kind() == "arrow_function" || value.Kind() == "function_expression") {
			entity = declaration(node, source, result, scope, "function")
		}
	case "import_statement":
		addJavaScriptImport(node, source, result, scope)
	case "call_expression":
		handleJSCall(node, source, result, scope)
	}

	if entity != nil {
		result.Entities = append(result.Entities, *entity)
		nextScope = append(scope, entity.Key)
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if child := node.NamedChild(index); child != nil {
			walkJS(child, source, result, nextScope)
		}
	}
}

func addJavaScriptImport(
	node *treesitter.Node,
	source []byte,
	result *ParseResult,
	scope []string,
) {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode == nil {
		return
	}
	result.Imports = append(result.Imports, ImportSeed{
		FromKey:   currentKey(scope),
		Specifier: trimLiteral(nodeText(sourceNode, source)),
		Range:     nodeRange(node),
	})
}

func handleJSCall(node *treesitter.Node, source []byte, result *ParseResult, scope []string) {
	function := node.ChildByFieldName("function")
	if function == nil {
		return
	}
	callee := strings.TrimSpace(nodeText(function, source))
	if callee == "" {
		return
	}

	arguments := node.ChildByFieldName("arguments")
	if callee == "require" && arguments != nil && arguments.NamedChildCount() > 0 {
		result.Imports = append(result.Imports, ImportSeed{
			FromKey:   currentKey(scope),
			Specifier: trimLiteral(nodeText(arguments.NamedChild(0), source)),
			Range:     nodeRange(node),
		})
		return
	}

	method, receiver := splitCallee(callee)
	if routeMethodPattern.MatchString(method) &&
		isRouteReceiver(receiver) &&
		arguments != nil &&
		arguments.NamedChildCount() >= 2 {
		routePath := trimLiteral(nodeText(arguments.NamedChild(0), source))
		if strings.HasPrefix(routePath, "/") {
			addRoute(
				result,
				strings.ToUpper(method),
				routePath,
				arguments,
				source,
				nodeRange(node),
			)
			return
		}
	}

	result.References = append(result.References, ReferenceSeed{
		FromKey:    currentKey(scope),
		Target:     callee,
		Kind:       "calls",
		Range:      nodeRange(node),
		Confidence: 0.72,
	})
}

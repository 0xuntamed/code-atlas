//go:build cgo

package parser

import (
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func walkGo(node *treesitter.Node, source []byte, result *ParseResult, scope []string) {
	nextScope := scope
	var entity *EntitySeed
	switch node.Kind() {
	case "function_declaration":
		entity = declaration(node, source, result, scope, "function")
	case "method_declaration":
		entity = declaration(node, source, result, scope, "method")
	case "type_spec":
		entity = goTypeDeclaration(node, source, result, scope)
	case "import_spec":
		addGoImport(node, source, result)
	case "call_expression":
		handleGoCall(node, source, result, scope)
	}

	if entity != nil {
		result.Entities = append(result.Entities, *entity)
		nextScope = append(scope, entity.Key)
	}
	for index := uint(0); index < node.NamedChildCount(); index++ {
		if child := node.NamedChild(index); child != nil {
			walkGo(child, source, result, nextScope)
		}
	}
}

func goTypeDeclaration(
	node *treesitter.Node,
	source []byte,
	result *ParseResult,
	scope []string,
) *EntitySeed {
	kind := "struct"
	if typeNode := node.ChildByFieldName("type"); typeNode != nil && typeNode.Kind() == "interface_type" {
		kind = "interface"
	}
	return declaration(node, source, result, scope, kind)
}

func addGoImport(node *treesitter.Node, source []byte, result *ParseResult) {
	pathNode := node.ChildByFieldName("path")
	if pathNode == nil {
		return
	}
	result.Imports = append(result.Imports, ImportSeed{
		FromKey:   "file",
		Specifier: trimLiteral(nodeText(pathNode, source)),
		Range:     nodeRange(node),
	})
}

func handleGoCall(node *treesitter.Node, source []byte, result *ParseResult, scope []string) {
	function := node.ChildByFieldName("function")
	if function == nil {
		return
	}
	arguments := node.ChildByFieldName("arguments")
	callee := nodeText(function, source)
	method, receiver := splitCallee(callee)

	if arguments != nil && arguments.NamedChildCount() >= 2 {
		if strings.EqualFold(callee, "http.HandleFunc") &&
			addGoRoute(result, "ANY", arguments, source, node) {
			return
		}
		if routeMethodPattern.MatchString(method) &&
			isRouteReceiver(receiver) &&
			addGoRoute(result, strings.ToUpper(method), arguments, source, node) {
			return
		}
	}

	result.References = append(result.References, ReferenceSeed{
		FromKey:    currentKey(scope),
		Target:     callee,
		Kind:       "calls",
		Range:      nodeRange(node),
		Confidence: 0.8,
	})
}

func addGoRoute(
	result *ParseResult,
	method string,
	arguments *treesitter.Node,
	source []byte,
	call *treesitter.Node,
) bool {
	routePath := trimLiteral(nodeText(arguments.NamedChild(0), source))
	if !strings.HasPrefix(routePath, "/") {
		return false
	}
	addRoute(result, method, routePath, arguments, source, nodeRange(call))
	return true
}

package parser

import "testing"

func TestLanguageExtraction(t *testing.T) {
	tests := []struct{ name, path, language, source, wantEntity, wantRoute, wantImport string }{
		{name: "javascript", path: "src/routes/users.js", language: "javascript", source: `import { listUsers } from "../controllers/users";
export function audit() { return listUsers(); }
router.get("/users", auth, listUsers);`, wantEntity: "audit", wantRoute: "GET /users", wantImport: "../controllers/users"},
		{name: "typescript", path: "src/app/api/orders/route.ts", language: "typescript", source: `import { loadOrders } from "@/orders";
export async function GET() { return loadOrders(); }`, wantEntity: "GET", wantRoute: "GET /api/orders", wantImport: "@/orders"},
		{name: "go", path: "internal/http/routes.go", language: "go", source: `package httpapi
import "example.com/app/service"
func listUsers() { service.LoadUsers() }
func Routes(router *gin.Engine) { router.GET("/users", auth, listUsers) }`, wantEntity: "listUsers", wantRoute: "GET /users", wantImport: "example.com/app/service"},
		{name: "python", path: "app/routes/users.py", language: "python", source: `from app.services import users
@app.get("/users")
def list_users():
    return users.list_all()`, wantEntity: "list_users", wantRoute: "GET /users", wantImport: "app.services"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse(test.path, test.language, []byte(test.source))
			if err != nil {
				t.Fatal(err)
			}
			if !hasEntity(result, test.wantEntity) {
				t.Fatalf("missing entity %q in %#v", test.wantEntity, result.Entities)
			}
			if !hasEntity(result, test.wantRoute) {
				t.Fatalf("missing route %q in %#v", test.wantRoute, result.Entities)
			}
			if !hasImport(result, test.wantImport) {
				t.Fatalf("missing import %q in %#v", test.wantImport, result.Imports)
			}
		})
	}
}

func hasEntity(result ParseResult, name string) bool {
	for _, entity := range result.Entities {
		if entity.Name == name {
			return true
		}
	}
	return false
}
func hasImport(result ParseResult, specifier string) bool {
	for _, item := range result.Imports {
		if item.Specifier == specifier {
			return true
		}
	}
	return false
}

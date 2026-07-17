# CodeAtlas

CodeAtlas is a local-first code intelligence graph for JavaScript, TypeScript, Go, and Python repositories. It maps symbols, imports, calls, web routes, execution flow, and upstream/downstream impact without storing source code in PostgreSQL.

## Privacy boundary

- The Go analyzer reads source from the registered repository and discards each source buffer after parsing.
- PostgreSQL stores file metadata, hashes, source ranges, symbols, relationships, and analysis status only.
- Source evidence is read from disk on demand and rejected if its hash changed after analysis.
- The server binds to loopback and rejects non-loopback Host and Origin values.
- Git credentials are never accepted in URLs or stored; managed clones use the local credential helper or SSH agent.
- No AI provider, analytics service, external font, or telemetry endpoint is used.

## Requirements

- Go 1.26+
- Node.js 22+ and npm
- Docker with Compose
- Git
- A C toolchain for production Tree-sitter builds. When CGO is unavailable, CodeAtlas compiles a conservative structural parser fallback so local development remains functional.

## Run locally

```powershell
docker compose up -d postgres
cd web
npm install
npm run build
cd ..
go run ./cmd/codeatlas serve
```

Open `http://127.0.0.1:7331` and paste an absolute local repository path or an HTTPS/SSH Git URL.

The header keeps repository lifecycle actions available after onboarding: add another repository, switch projects, reanalyze, remove derived metadata, or explicitly stop the local CodeAtlas process.

The bundled PostgreSQL container is exposed on loopback port `5433` to avoid colliding with a conventional host PostgreSQL installation on `5432`.

During frontend development, run `npm run dev` in `web`; Vite proxies `/api` to the Go server. Register a folder from the terminal with:

```powershell
go run ./cmd/codeatlas add C:\absolute\path\to\repo
```

Configuration:

| Variable | Default |
| --- | --- |
| `CODEATLAS_DATABASE_URL` | `postgres://codeatlas:codeatlas@127.0.0.1:5433/codeatlas?sslmode=disable` |
| `CODEATLAS_LISTEN_ADDR` | `127.0.0.1:7331` |
| `CODEATLAS_DATA_DIR` | OS user cache directory under `CodeAtlas` |

## Analysis model

The analyzer inventories the repository, applies hard privacy exclusions, `.gitignore`, and `.codeatlasignore`, parses declarations and imports, resolves cross-file relationships, detects supported route patterns, and atomically promotes the completed graph. Failed or partial runs never replace the last valid graph. File hashes key a process-local metadata-only parse cache so unchanged files are reused during later analyses without retaining source or ASTs.

Supported route detectors:

- Express and Next.js route handlers for JavaScript/TypeScript
- `net/http` and Gin for Go
- FastAPI and Flask for Python

The skipped-file view explains every visited exclusion. Entire ignored directories are recorded once and pruned without enumerating their descendants. Tests remain included because they are useful impact dependents.

## Graph experience and render budgets

The architecture explorer uses progressive disclosure instead of flattening every symbol into one canvas:

1. The overview aggregates source files, symbols, routes, and cross-module relationships into module cards.
2. Opening a module loads only its immediate file neighborhood.
3. Opening a file loads only its declarations and direct dependencies.
4. Flow and impact views start from a selected symbol and request at most 120 nodes by default.

React Flow mounts only visible cards, dense views suppress persistent edge labels and animation, and ELK layout runs in a worker with a bounded fallback layout. The skipped-file list also has a fixed DOM budget. Monaco and the four supported syntax tokenizers are bundled locally and lazy-loaded only when source evidence is opened.

## Code layout

- `internal/api`: loopback security, project lifecycle, analysis events, graph handlers, source evidence, and process shutdown.
- `internal/analyzer`: explicit inventory, structural entity, declaration, relationship, and resolution phases.
- `internal/parser`: Tree-sitter adapters plus a no-CGO development fallback.
- `internal/repository`: safe Git acquisition, discovery, hashing, and ignore rules.
- `internal/store`: PostgreSQL migrations, durable jobs, atomic promotion, search, and graph traversal.
- `web/src/features`: project, analysis, graph, search, file inventory, source, and workspace UI boundaries.
- `web/src/styles`: formatted design-system, shell, feature, graph, inspector, and responsive styles.

## API

The local JSON API is rooted at `/api/v1`. Key endpoints include project creation and reanalysis, SSE progress, skipped-file inventory, symbol search, architecture graph, execution flow, impact traversal, entity details, live source evidence, and explicit local shutdown. Architecture requests accept a `scope` entity ID for drill-down. Graph responses retain a 500-node hard safety cap and traversal depth is capped at 12; the UI uses smaller 80/120-node budgets.

## Tests

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
cd web
npm run check
npm run build
```

`npm run check` runs strict TypeScript, ESLint with React Hooks rules, Vitest, and Prettier verification. The repository includes parser fixtures for all supported languages, privacy/ignore tests, API security and source-window tests, graph resolution tests, and a cross-platform CGO CI build that exercises the official Tree-sitter grammars.

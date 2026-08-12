# CodeAtlas

CodeAtlas is a local-first code intelligence graph for JavaScript, TypeScript, Go, and Python repositories. It maps symbols, imports, calls, web routes, execution flow, and upstream/downstream impact without storing source code in its database. It ships as a single self-contained binary: metadata lives in an embedded SQLite database (no external database process, no Docker), and the Go server serves the production React interface as embedded static assets.

## Privacy boundary

- The Go analyzer reads source from the registered repository and discards each source buffer after parsing.
- The embedded SQLite database stores file metadata, hashes, source ranges, symbols, relationships, and analysis status only.
- Source evidence is read from disk on demand and rejected if its hash changed after analysis.
- The server binds to loopback and rejects non-loopback Host and Origin values.
- Git credentials are never accepted in URLs or stored; managed clones use the local credential helper or SSH agent.
- No AI provider, analytics service, external font, or telemetry endpoint is used.

## Requirements

- Go 1.26+
- Node.js 22.12+ (or 20.19+) and npm — only to build the frontend bundle
- Git
- A C toolchain for production Tree-sitter builds. When CGO is unavailable, CodeAtlas compiles a conservative structural parser fallback so local development remains functional. (The SQLite driver is pure Go and needs no C toolchain.)

No Docker and no external database are required — metadata is stored in a local SQLite file.

## Run locally

```powershell
cd web
npm install
npm run build
cd ..
go run ./cmd/codeatlas serve
```

Open `http://127.0.0.1:7331` and paste an absolute local repository path or an HTTPS/SSH Git URL. You can also point CodeAtlas at a repository directly on startup, and it registers on first launch:

```powershell
go run ./cmd/codeatlas serve C:\absolute\path\to\repo
```

The header keeps repository lifecycle actions available after onboarding: add another repository, switch projects, reanalyze, remove derived metadata, or explicitly stop the local CodeAtlas process.

The SQLite database is created under the data directory (`<data-dir>/codeatlas.db`) on first run.

During frontend development, keep the Go server running and start Vite in a second terminal. Vite defaults to `127.0.0.1:5173` and proxies `/api` to the Go server:

```powershell
# Terminal 1
go run ./cmd/codeatlas serve

# Terminal 2
cd web
npm run dev
```

Register a folder from the terminal with:

```powershell
go run ./cmd/codeatlas add C:\absolute\path\to\repo
```

Configuration:

| Variable                  | Default                                    |
| ------------------------- | ------------------------------------------ |
| `CODEATLAS_DATABASE_PATH` | `<data-dir>/codeatlas.db`                  |
| `CODEATLAS_LISTEN_ADDR`   | `127.0.0.1:7331`                           |
| `CODEATLAS_DATA_DIR`      | OS user cache directory under `CodeAtlas`  |

## Frontend architecture

The frontend is a React 19 and TypeScript application built with Vite 7 and Tailwind CSS v4 through the official Vite integration.

- `web/src/styles/app.css` defines the CSS-first semantic token contract, component radii, graph colors, animations, and reduced-motion behavior.
- Reusable buttons, dialogs, icons, and brand elements live in `web/src/components` and consume static Tailwind variant maps.
- TanStack Query owns API requests, cancellation, polling, retries, and cache invalidation. Offline and mutation failures remain visible instead of falling through to empty states.
- Zustand stores only cross-workspace interaction state such as the selected project, graph mode, selected entity, and explicit signal filters.
- React Flow renders the bounded graph, while ELK layout runs outside the main thread in a worker and falls back to a deterministic grid if layout exceeds its time budget.
- Monaco and its local language support are lazy-loaded only when verified source evidence is requested.

`npm run build` writes the production frontend directly to `internal/webui/dist`. That directory is embedded by `internal/webui/webui.go`; rebuild it before compiling or distributing the Go executable after frontend changes.

## Analysis model

The analyzer inventories the repository, applies hard privacy exclusions, `.gitignore`, and `.codeatlasignore`, parses declarations and imports, resolves cross-file relationships, detects supported route patterns, and atomically promotes the completed graph. Failed or partial runs never replace the last valid graph. File hashes key a process-local metadata-only parse cache so unchanged files are reused during later analyses without retaining source or ASTs.

Supported route detectors:

- Express and Next.js route handlers for JavaScript/TypeScript
- `net/http` and Gin for Go
- FastAPI and Flask for Python

The skipped-file view explains every visited exclusion. Entire ignored directories are recorded once and pruned without enumerating their descendants. Unsupported files, generated output, dependencies, binaries, ignored paths, and privacy exclusions remain inventory metadata and never become graph nodes.

Tests are still analyzed because they are useful impact dependents, but test files, test symbols, and test-only modules are hidden from the canvas by default. They can be enabled with the **Include tests** filter.

## Graph experience and render budgets

The architecture explorer uses progressive disclosure instead of flattening every symbol into one canvas:

1. The overview aggregates source files, symbols, routes, and cross-module relationships into module cards.
2. Opening a module loads only its immediate file neighborhood.
3. Opening a file loads only its declarations and direct dependencies.
4. Flow and impact views start from a selected symbol and request at most 120 nodes by default.

The default canvas keeps only supported architectural entities and hides external or unresolved symbols behind the **Reference noise** filter. Edges are retained only when both endpoints remain visible, preventing hidden or unimportant records from leaving orphaned relationships.

React Flow mounts only visible cards, dense views suppress persistent edge labels and animation, and ELK layout runs in a worker with a bounded fallback layout. Graph layout identity includes edge topology so same-size graph updates cannot reuse stale positions. The skipped-file list uses `content-visibility` and a bounded API result. Monaco and the four supported syntax tokenizers are bundled locally and lazy-loaded only when source evidence is opened.

## Code layout

- `internal/api`: loopback security, project lifecycle, analysis events, graph handlers, source evidence, and process shutdown.
- `internal/analyzer`: explicit inventory, structural entity, declaration, relationship, and resolution phases.
- `internal/parser`: Tree-sitter adapters plus a no-CGO development fallback.
- `internal/repository`: safe Git acquisition, discovery, hashing, and ignore rules.
- `internal/store`: embedded SQLite migrations, durable jobs, atomic promotion, search, and graph traversal.
- `internal/webui`: generated frontend assets and the `go:embed` HTTP handler.
- `web/src/components`: accessible UI primitives and the in-repository SVG icon system.
- `web/src/features`: project, analysis, graph, search, file inventory, source, and workspace UI boundaries.
- `web/src/lib`: graph filtering, graph-layout identity, and shared utilities.
- `web/src/styles/app.css`: Tailwind v4 theme tokens, base styles, custom utilities, and React Flow integration styles.

## API

The local JSON API is rooted at `/api/v1`. Key endpoints include project creation and reanalysis, SSE progress, skipped-file inventory, symbol search, architecture graph, execution flow, impact traversal, entity details, live source evidence, and explicit local shutdown. Architecture requests accept a `scope` entity ID for drill-down. Graph responses retain a 500-node hard safety cap and traversal depth is capped at 12; the UI uses smaller 80/120-node budgets.

## Tests

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
cd web
npm run check
npm run build
npm audit
```

`npm run check` runs strict TypeScript, ESLint with React Hooks rules, Vitest, and Prettier verification. Frontend tests cover workspace-state resets, explicit filter persistence, low-signal graph filtering, test-only modules, edge cleanup, and topology-sensitive layout keys. The Go suite includes parser fixtures for all supported languages, privacy/ignore tests, API security and source-window tests, graph resolution tests, test-only module classification, and a cross-platform CGO CI build that exercises the official Tree-sitter grammars.

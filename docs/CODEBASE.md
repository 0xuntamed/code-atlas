# CodeAtlas — Codebase & Flow Reference

A function-level guide to the whole system: what every part does, how it works, and the
exact path a request travels — from **adding a repository** all the way to a **colour-coded
blast radius** on screen.

> This is the deep companion to [`ARCHITECTURE.md`](ARCHITECTURE.md) (which is the higher-level
> tour). Where this doc cites `file.go:NN`, that's a real, clickable location in the source.

---

## 0. Mental model in five sentences

1. CodeAtlas is a **single Go binary**. It runs a background **analyzer** and an **HTTP API**
   in one process; they never call each other directly — they communicate only through an
   **embedded SQLite** database.
2. The analyzer turns a repository into a **graph**: nodes are `entities` (module, file,
   function, route…), edges are `relationships` (contains, defines, imports, calls…).
3. Every graph belongs to one **analysis run**; a project points at its `active_run_id`, and a
   finished run is swapped in **atomically** — a failed run never replaces the last good graph.
4. The React frontend (built and **embedded** into the same binary) reads that graph over a
   loopback-only JSON API and draws it with React Flow.
5. Selecting any node asks the backend for its **impact map** — everything that depends on it
   and everything it depends on — which the UI paints over the base map.

---

## 1. Repository layout

```
cmd/codeatlas/main.go        Entry point: CLI (serve / add) + server wiring
internal/
  model/model.go             Shared domain types + the two controlled vocabularies
  id/id.go                   id.New() (random) and id.Stable() (deterministic) IDs
  repository/
    acquire.go               Git acquisition (clone/fetch) + URL validation
    discover.go              Filesystem walk, classification, hashing, hard exclusions
    ignore.go                .gitignore / .codeatlasignore matching
  parser/
    types.go                 ParseResult + the three "seed" types
    parser.go                Tree-sitter parser (//go:build cgo)
    parser_fallback.go       Regex structural parser (//go:build !cgo)
    javascript.go go.go python.go   Per-language Tree-sitter walkers (cgo)
  analyzer/
    service.go               Job loop + pipeline orchestration + parse worker pool
    graph.go                 graphBuilder + phase ordering
    declarations.go          Modules, file entities, declarations (contains / defines)
    relationships.go         imports / calls / route edges + synthetic nodes
    resolution.go            Symbol & import resolution heuristics
  store/
    schema.sql               SQLite DDL
    store.go                 Connection, migrations, projects, runs, persistence
    graph.go                 All graph read queries (architecture / flow / impact)
    store_test.go            End-to-end store tests over a temp SQLite file
  api/
    server.go                Router + Server struct
    middleware.go            Loopback security + security headers
    projects.go              Create / list / get / delete projects
    analysis.go              Queue re-analysis + SSE progress stream
    graph.go                 Graph, flow, impact, impact-map, search, entity handlers
    source.go                Hash-verified source evidence
    system.go                Local shutdown
    response.go paths.go     JSON helpers, param parsing, path containment
  webui/webui.go             go:embed of the built SPA + client-route fallback
web/                         React 19 + TypeScript frontend (built into internal/webui/dist)
```

---

## 2. Bootstrap & entry points — `cmd/codeatlas/main.go`

| Function | Role |
| --- | --- |
| `main` → `run(args)` | Dispatches the subcommand: no args or `serve` → `serve`; `add` → `add`; else usage. |
| `serve(args)` | The whole server. Parses flags, opens the store, launches the analyzer, starts HTTP. |
| `add(args)` | A thin CLI client: POSTs a local path to `/api/v1/projects` on a running server. |
| `registerOnStart` | If `serve` is given a positional repo path, waits for `/health` then registers it. |
| `defaultDataDir` | OS user-cache dir under `CodeAtlas` (holds the SQLite file + managed clones). |

**`serve` wiring** (`main.go:63`): resolve flags (`--listen`, `--database`, `--data-dir`;
DB path defaults to `<data-dir>/codeatlas.db`) → `store.Open` → `store.Migrate` →
`analyzer.New(db, logger).Run(ctx)` in a **goroutine** → mount `api.New(db, dataDir, logger,
cancel).Handler()` on an `http.Server` → a second goroutine calls `server.Shutdown` when the
signal context is cancelled. If a repo path was passed, `registerOnStart` runs in the
background so a first launch needs no separate `add`.

---

## 3. Data model & storage

### 3.1 Domain types — `internal/model/model.go`

`Project`, `AnalysisRun`, `FileRecord`, `Entity` (with `Range`, `Distance`, `Direction`,
`Metadata`), `Relationship`, `Graph`, `SourceEvidence`, `ProjectSource`,
`CreateProjectRequest/Response`.

**Two controlled vocabularies** everything keys off (`model.go:20`):

- `EntityKinds`: `package, module, file, function, method, class, struct, interface, route,
  external_symbol, unresolved_symbol`.
- `RelationshipKinds`: `contains, defines, imports, exports, calls, handles_route,
  uses_middleware, depends_on`.

> **Accuracy note:** the analyzer currently *emits* six of these edge kinds — `contains`,
> `defines`, `imports`, `calls`, `handles_route`, `uses_middleware`. `exports` and
> `depends_on` are defined in the vocabulary and referenced by some read queries, but the
> present builder does not produce them yet.

### 3.2 Schema — `internal/store/schema.sql`

Five tables: `projects`, `analysis_runs`, `files`, `entities`, `relationships`. Conventions
that matter:

- **Run-scoped rows.** `files`/`entities`/`relationships` all carry `run_id`. Reads join
  `JOIN projects p ON p.active_run_id = e.run_id`, so an in-progress run is invisible.
- **Timestamps are INTEGER Unix microseconds** (UTC); booleans are `0/1`; `metadata` is JSON
  in a `TEXT` column. (All SQLite-friendly — the driver is pure Go.)
- **Foreign keys are on** (`store.Open` sets the pragma), so deleting a project cascades to its
  runs, files, entities, and relationships.

### 3.3 Store plumbing — `internal/store/store.go`

- `Open(ctx, path)` — opens `modernc.org/sqlite` with
  `busy_timeout(5000)`, `journal_mode(WAL)`, `foreign_keys(on)`; `MaxOpenConns(8)`.
- `Migrate` — `splitStatements(schema.sql)` and executes each (idempotent `CREATE … IF NOT
  EXISTS`).
- Timestamp helpers `nowMicros` / `microsFromNow` / `microsToTime`; the `scanner` interface
  lets one `scanRun` / `scanEntity` / `scanProject` serve both `*sql.Row` and `*sql.Rows`.
- Query-building helpers reused across the read path: `placeholders(n)`, `stringArgs`,
  `scanIDs`, `nullString`, `marshalMeta`, `boolInt`.

---

## 4. The write path — from a repo to a stored graph

This is the heart of the system. Entry point: `analyzer.Service.analyze`
(`internal/analyzer/service.go:52`).

### 4.1 Step 0 — a repository is added

- **From the UI / API:** `POST /api/v1/projects` → `api/projects.go:createProject` →
  `newProject` → `newLocalProject` (abs-path, symlink-resolve, must be a directory) **or**
  `newGitProject` (validates the URL, marks `ManagedClone`, sets the clone target to
  `<data-dir>/repositories/<projectID>`) → `store.CreateProject` inserts the **project + a
  queued run** in one transaction. Returns `202 Accepted`.
- **From the CLI:** `codeatlas add <path>` POSTs the same request; `codeatlas serve <path>`
  registers on first launch via `registerOnStart`.

### 4.2 Step 1 — the job loop claims the run

`Service.Run` (`service.go:29`) ticks every **650 ms** and calls `store.ClaimRun`, which — in
a transaction — selects the oldest `queued` run (or a `running` run whose 45-second
`lease_until` has expired), flips it to `running`, and stamps a fresh lease. A single embedded
worker means no contention; the lease exists only so a crash-orphaned run is re-claimable on
restart. Each `store.UpdateRun` renews the lease as it reports progress.

### 4.3 Step 2 — `analyze()` orchestrates the pipeline

`analyze` (`service.go:52`) loads the project, installs a **deferred failure guard** (on any
error it calls `store.FailRun` with a **path-sanitized** message via `sanitizeError`, leaving
the previous graph intact), then walks the stages, calling `UpdateRun` before each:

```
acquiring → discovering → parsing → resolving → persisting → (promote) → ready
```

### 4.4 Step 3 — acquire — `internal/repository/acquire.go`

`Acquire(ctx, project)`:
- **local** → returns `""` (nothing to fetch).
- **git** → `ValidateGitURL` (rejects credential-bearing URLs and non-HTTPS/SSH schemes), then
  a shallow `git clone --depth=1` (or `fetch` + `checkout` if the clone dir already exists),
  with `GIT_TERMINAL_PROMPT=0` and `safeGitError` scrubbing tokens from any error.
- Ends with `git rev-parse HEAD`; the commit is saved via `store.UpdateProjectCommit`.

### 4.5 Step 4 — discover — `internal/repository/discover.go`

`Discover(ctx, projectID, runID, root)`:
1. `filepath.EvalSymlinks` + `Abs` to get the canonical root; load ignore rules
   (`loadIgnoreMatcher` reads `.gitignore` + `.codeatlasignore`).
2. `filepath.WalkDir`. For each entry, build a `FileRecord` (default classification
   `skipped`) and apply, **in order**:
   - **Hard safety** (`hardExcluded` + symlink + binary checks): symlinks aren't followed;
     secret-like files (`.env*`, `*.pem`, `*.key`, `id_rsa`, `credentials`, `.npmrc`…),
     binaries (extension or NUL-byte sniff via `looksBinary`), and files **> 1 MiB**
     (`MaxSourceFileSize`) are excluded with a reason. Excluded directories are `SkipDir`'d.
   - **Ignore rules** (`matcher.match`) next.
   - **Classification**: a known source extension → `source` (record language, `isTestFile`,
     and a **SHA-256** `content_hash` via `hashFile`); `package.json`/`go.mod`/… →
     `support`; anything else → `skipped` with "unsupported file type".
3. Returns `[]DiscoveredFile` (record + absolute path). `analyze` then enforces the
   **10 000 source-file** MVP cap.

### 4.6 Step 5 — parse — `Service.parseFiles` + `internal/parser/*`

`parseFiles` (`service.go:113`) runs a bounded **worker pool** (`min(NumCPU, 8)`):
- A process-local `sync.Map` **cache** keyed by `language\x00path\x00contentHash` reuses
  results for unchanged files across runs — **metadata only; the source buffer is read, parsed,
  then dropped** (`content = nil`).
- Progress is flushed to the run row at most every **500 ms**.

`parser.Parse(path, language, source)` has two build-tagged implementations:

- **With CGO — `parser.go`** drives the real **Tree-sitter** grammars (`.tsx` uses the TSX
  grammar), walks the syntax tree per language (`walkJS` / `walkGo` / `walkPython` in the
  per-language files), and records `tree.RootNode().HasError()` as `HasSyntaxErrors`.
- **Without CGO — `parser_fallback.go`** is a conservative **regex line-scanner**
  (`parseJSLines` / `parseGoLines` / `parsePythonLines`) that emits the same shapes at lower
  confidence — so cross-compiled, toolchain-free builds still work.

Both produce a `ParseResult` (`types.go`) of **seeds** — nothing resolved across files yet:
- `EntitySeed` — a declaration or route (`Key`, `Kind`, `Name`, `QualifiedName`, `Range`).
- `ImportSeed` — an import (`FromKey`, `Specifier`, `Range`).
- `ReferenceSeed` — a call / route-handler / middleware use (`FromKey`, `Target`, `Kind`,
  `Confidence`).

**Route detection** lives here: receiver heuristics (`isRouteReceiver` — `app`, `router`,
`*router`, `*engine`…), decorator patterns for Python, and a **Next.js App-Router** special
case (`addNextRoutes`) that synthesizes a `route` from an `app/**/route.ts` path plus its
exported HTTP-method functions.

### 4.7 Step 6 — build the graph — `internal/analyzer/graph.go` (+ declarations / relationships / resolution)

`buildGraph` (`graph.go:36`) constructs a `graphBuilder` and runs **five ordered phases**:

| Phase | File | Produces |
| --- | --- | --- |
| `indexFiles` | declarations.go | Registers file records + parse results by path. |
| `addStructuralEntities` | declarations.go | One `module` per directory (`ensureModule`) + one `file` entity per source file (`newFileEntity`), joined by **`contains`**. Marks a module `IsTest` if it has files but no production files. |
| `addDeclarations` | declarations.go | Promotes each `EntitySeed` to an `entity`, links it from its file by **`defines`**, and indexes it (`indexEntity` → `byName`, `byQualified`). |
| `addImports` | relationships.go | Resolves each import (`resolveImportedFile` → `resolveImport`) to a file entity (**`imports`**, `resolved`, conf 0.88) or synthesizes an `external_symbol` (`external`, 0.7); records `importsByFile`. |
| `addReferences` | relationships.go | Resolves each `ReferenceSeed` (`resolveReference`) to a target; unresolved targets become an `unresolved_symbol` node (`unresolvedEntity`, conf 0.35). Emits `calls` / `handles_route` / `uses_middleware`. |

**Resolution ladder** (`resolution.go:resolveReference`), highest confidence first:

```
exact qualified-name match ............ 1.00  (resolved)
same file / enclosing scope ........... 0.95  (resolved)
matches an imported module ............ 0.90  (resolved)
the only candidate with that name ..... 0.72  (inferred)
no match → synthetic unresolved node .. 0.35  (unresolved)
```

`normalizeTarget` is the deliberately fuzzy key: it strips `await`/`new`/call-args and
lower-cases the last dotted segment. `resolveImport` builds per-language candidate paths
(`javascriptImportCandidates`, `pythonImportCandidates`, `goImportCandidates`) and keeps the
first that exists in the file set.

**Determinism:** every id comes from `id.Stable(runID, parts…)` (a SHA-256 prefix), so identical
input yields an identical graph. `finish()` sorts entities by qualified name and
`dedupeRelationships` collapses duplicates by id.

### 4.8 Step 7 — persist — `store.ReplaceRunData`

Inside one transaction (`store.go`): delete this run's `relationships` → `entities` → `files`
(children first), then **batched prepared `INSERT`s** for files, entities, relationships.
`content_hash` is written here; `metadata` is JSON-encoded into TEXT.

### 4.9 Step 8 — promote — `store.PromoteRun`

Atomically flips the run to `ready` and sets `projects.active_run_id`; then GCs superseded
`ready` runs older than a minute (their rows cascade-delete). **This is the safety guarantee:**
a failed run only calls `FailRun`, so the last valid `active_run_id` is never disturbed.

### 4.10 Progress plumbing

`UpdateRun` writes `stage / completed / total / message` and renews the lease. The UI learns
about it through the SSE endpoint (§5.4).

---

## 5. The read path — serving the graph

### 5.1 Security — `internal/api/middleware.go`

Every request passes `Server.security`: reject non-loopback `Host` (`isLoopbackHost`) and
cross-origin `Origin` (`isLoopbackOrigin`); set `nosniff`, `no-referrer`, a strict CSP, and
`no-store` on `/api/` responses.

### 5.2 Routes — `internal/api/server.go`

| Method & path | Handler | Store call |
| --- | --- | --- |
| `GET /health` | `health` | — |
| `POST /system/shutdown` | `stopApplication` | cancels the server context |
| `GET /projects` | `listProjects` | `Projects` |
| `POST /projects` | `createProject` | `CreateProject` |
| `GET /projects/{id}` | `getProject` | `Project` |
| `DELETE /projects/{id}` | `deleteProject` | `DeleteProject` (+ remove managed clone) |
| `POST /projects/{id}/analyses` | `queueAnalysis` | `QueueAnalysis` |
| `GET /projects/{id}/events` | `events` | `LatestRun` (SSE) |
| `GET /projects/{id}/files` | `files` | `Files` |
| `GET /projects/{id}/search?q=` | `search` | `Search` |
| `GET /projects/{id}/graph/architecture?scope=` | `architecture` | `ArchitectureGraph` |
| `GET /projects/{id}/flow/{entityID}` | `flow` | `FlowGraph` |
| `GET /projects/{id}/impact/{entityID}?direction=` | `impact` | `ImpactGraph` |
| `GET /projects/{id}/impact-map/{entityID}` | `impactMap` | `ImpactMap` |
| `GET /projects/{id}/entities/{entityID}` | `entity` | `Entity` |
| `GET /projects/{id}/entities/{entityID}/source` | `source` | `Evidence` + disk read |
| `GET /` (catch-all) | `webui.Handler()` | serves the embedded SPA |

### 5.3 Graph queries — `internal/store/graph.go`

- **`ArchitectureGraph(scope, limit)`** — with no scope, returns **module aggregates** (each
  module carries `fileCount / symbolCount / routeCount / testFileCount` via a grouped join with
  `FILTER` clauses), plus **`moduleEdges`** which rolls file-level `imports`/`calls` up to
  module→module edges. With a scope id it delegates to `NeighborhoodGraph`.
- **`NeighborhoodGraph(root, limit)`** — one root + its immediate neighbours; the edge kinds
  depend on the root's kind (a module expands via `contains`; a file via
  `defines`/`imports`/…).
- **`traversalNodes(seeds[], direction, depth, limit, kinds)`** — the engine. A **recursive
  CTE** whose anchor is a `VALUES` list of seed ids (so it walks from *many* seeds at once),
  with a `/`-delimited-string **cycle guard** and direction-specific next-node logic
  (`upstream` / `downstream` / `both`). Returns nodes with their `Distance`. Safety caps:
  depth ≤ 12 (`clampDepth`), nodes ≤ 500 (`clampNodes`).
- **`traversalGraph`** wraps it for a single root; **`FlowGraph`** (downstream) and
  **`ImpactGraph`** (both/upstream/downstream) are thin callers.
- **`ImpactMap(root, depth, limit)`** — the blast radius (see §7B). Resolves seeds by kind,
  walks upstream and downstream, tags each node's `Direction`, and pulls edges with
  `edgesForNodes`.
- **`Search`** (`LIKE` + length ranking), **`Entity`**, **`Files`**, **`Evidence`**.

### 5.4 Live progress — `internal/api/analysis.go`

`events` sets SSE headers and every **700 ms** reads `LatestRun`, emitting a `progress` event
only when the JSON changed; it closes when the run reaches `ready` or `failed`.

### 5.5 Verified source — `internal/api/source.go`

`source` loads the entity's file via `store.Evidence`, confirms the path is **within** the repo
(`isWithin`), re-hashes the file and **constant-time compares** it to the hash stored at
analysis time:
- match → returns a bounded window (`±3` context lines, ≤ 400 lines) as `SourceEvidence`;
- mismatch → `409 source_stale`; missing → `404 source_unavailable`.

### 5.6 Embedded UI — `internal/webui/webui.go`

`//go:embed dist/*` serves the built SPA; unknown paths fall back to `index.html` for
client-side routing.

---

## 6. The frontend — `web/src/`

### 6.1 Bootstrap & state

- **`main.tsx`** creates the TanStack Query `QueryClient` (staleTime 15 s, retry 1).
- **`App.tsx`** polls `['projects']` every **3 s**, keeps a valid `projectId` selected, and
  owns the add / remove / reanalyze / **shutdown** dialogs. `AppContent` routes to
  loading → error → `Onboarding` → `Workspace`.
- **`store.ts`** (Zustand) holds *interaction* state only: `projectId`, `selectedEntityId`,
  `impactFilter` (`both`/`dependents`/`dependencies`), the `scopePath` breadcrumb, and the two
  signal filters — with actions `setProject`, `selectEntity`, `setImpactFilter`, `openScope`,
  `navigateScope`, `toggle*`, `resetSelection`. All **server** state lives in TanStack Query.

### 6.2 The workspace data hook — `features/workspace/useWorkspaceGraph.ts`

The single source of truth for what's on the canvas:
1. **Base map** query — `api.architecture(scopeId)`, keyed
   `['graph', projectId, activeRunId, 'structure', scopeId]`.
2. **Selected entity** — taken from the base map if present, else fetched (`api.entity`).
3. **Impact overlay** — whenever something is selected, `api.impactMap(selectedEntityId)` runs
   and is **merged onto the base map**; a `directionById` map records each node's role, honoring
   the `impactFilter`.
4. **`filterGraph`** hides tests / reference noise (references are force-shown while a blast
   radius is active).
5. **`rollupByFile`** collapses each affected file's symbols into one “N affected” file node
   (unless that file key is in the caller's `expandedFiles` set), and recomputes the in-radius
   edge set.

Returns `{ displayGraph, directionById, impactEdgeIds, impactActive, hiddenTotal,
selectedEntity, … }`.

### 6.3 Rendering — `features/graph/*`

- **`GraphCanvas.tsx`** computes an off-thread ELK layout in **`layout.worker.ts`** (a fresh
  worker per layout, 2 s timeout → deterministic grid fallback), memoizes React Flow nodes and
  edges, and colours by `directionById` (epicenter = accent, dependents = rose, dependencies =
  sky, both = violet), dimming everything outside the radius.
- **`AtlasNode.tsx`** draws each card — kind tone/icon from **`lib/entityKinds.ts`**, the
  direction ring, and the special “group” card (with an expand affordance). Clicking a group
  card calls `onToggleGroup`; clicking any other node selects it.
- **`RelationshipEdge.tsx`**, **`ImpactLegend.tsx`**, **`GraphToolbar.tsx`** (breadcrumb +
  legend), **`WorkspaceSidebar.tsx`** (search, blast-radius filter, signal filters, skipped
  files).

### 6.4 Shared libs

- **`lib/graph.ts`** — `filterGraph`, `graphLayoutKey` (includes edge topology so a same-size
  graph re-lays-out), `entityKindLabel`.
- **`lib/entityKinds.ts`** — one `KIND_META` table (tone, icon, meaningful, reference) plus the
  derived `MEANINGFUL_KINDS` / `REFERENCE_KINDS` / `TRACEABLE_KINDS` sets and `isTraceable`.
- **`lib/graphConfig.ts`** — `NODE_WIDTH/HEIGHT` and the `DENSITY` thresholds, shared by the
  canvas and the layout worker so they can't drift.

### 6.5 Supporting features

- **`useAnalysisEvents.ts`** opens an `EventSource` on the SSE endpoint and mirrors run
  progress into `AnalysisProgress.tsx`.
- **`SymbolSearch.tsx`** (debounced `api.search`), **`EntityInspector.tsx`** + lazy-loaded
  **`LocalSourceEditor.tsx`** (Monaco) for verified source, **`SkippedFiles.tsx`**.

---

## 7. End-to-end walkthroughs

### 7A. Add a local repo → see the map

```
UI AddProjectDialog → api.createProject
  → POST /api/v1/projects  (api/projects.go createProject → newLocalProject → store.CreateProject)
  → 202, project + queued run stored
analyzer.Service.Run tick → store.ClaimRun → analyze()
  → Acquire (local: validate) → Discover (walk/classify/hash) → parseFiles (tree-sitter)
  → buildGraph (5 phases) → ReplaceRunData → PromoteRun → status "ready"
meanwhile: App polls /projects (3s) and useAnalysisEvents streams stages over SSE
on ready: useWorkspaceGraph → api.architecture("") → module cards + module→module edges
  → GraphCanvas lays out with ELK → the map renders
```

### 7B. Click a function → blast radius

```
click node → GraphCanvas onNodeClick → store.selectEntity(id)
useWorkspaceGraph → api.impactMap(id)  →  GET /impact-map/{id}
  store.ImpactMap:
    impactSeeds(root)  → symbol: [itself] · file: itself + its defined symbols
                         · module: itself + its files + their symbols
    traversalNodes(seeds, "upstream")   → dependents   (tag: dependent)
    traversalNodes(seeds, "downstream") → dependencies (tag: dependency)
    merge + tag each node root / dependent / dependency / both  → edgesForNodes
  → hook merges the overlay onto the base map, builds directionById,
    filterGraph, rollupByFile
  → GraphCanvas colours nodes/edges by direction, dims the rest
  → toolbar shows the ripple source + legend; sidebar shows the direction filter
```

### 7C. Open verified source

```
EntityInspector "Load source evidence" → api.source(id) → GET /entities/{id}/source
  store.Evidence → path-containment check → re-hash file → constant-time compare
    match → bounded window returned → Monaco renders it
    changed → 409 source_stale ("reanalyze")
```

---

## 8. Feature catalogue (what it does · how it's done)

| Feature | What you get | How it works (entry points) |
| --- | --- | --- |
| **Whole-repo map** | Files/functions/imports as one graph, grouped by module | Tree-sitter parse → 5-phase `buildGraph` → `ArchitectureGraph` + `moduleEdges` |
| **Progressive disclosure** | Overview → module → file, never everything at once | `scopePath` + scoped `NeighborhoodGraph`, 80/120/160 node budgets |
| **Blast radius** | Click a node → colour-coded dependents & dependencies | `store.ImpactMap` (seed-by-kind + bidirectional `traversalNodes` + `Direction`) |
| **File rollup + expand** | “N affected” file nodes you can expand to functions | `useWorkspaceGraph.rollupByFile` + `expandedFiles` state |
| **Route awareness** | HTTP endpoints as first-class nodes | route detectors in `parser.go` / fallback; `route` + `handles_route` |
| **Signal filters** | Hide tests & external/unresolved noise | `lib/graph.ts filterGraph` + `entityKinds` sets |
| **Verified source** | Real code, only if it still matches analysis | `api/source.go` SHA-256 constant-time compare |
| **Live progress** | Stage-by-stage analysis feedback | `UpdateRun` → SSE `events` → `useAnalysisEvents` |
| **Privacy boundary** | Source never stored; loopback only | discard buffers after parse; `middleware.security` |
| **Single binary** | One file, no Docker/DB to install | pure-Go SQLite + `go:embed` SPA |

---

## 9. Security & privacy (summary)

- **Loopback only** — non-local `Host`/`Origin` rejected; strict CSP; `no-store` on the API.
- **No source in the database** — only paths, hashes, ranges, symbols, and relationships;
  buffers are discarded right after parsing and re-read on demand, hash-verified.
- **Hard exclusions** for secrets, binaries, oversized files, and symlinks — before ignore
  rules and independent of them.
- **Git** — credential-bearing URLs rejected; shallow clones; errors scrubbed of tokens.
- **Sanitized failures** — repository paths stripped and length-capped before storage.
- **No external anything** — no AI provider, analytics, remote fonts, or telemetry.

---

## 10. Build, run, test, and known gotchas

```powershell
# frontend bundle (embedded into the binary) — required before building the exe
cd web; npm ci; npm run build; cd ..

# run (opens http://127.0.0.1:7331) — optionally point it at a repo
go run ./cmd/codeatlas serve
go run ./cmd/codeatlas serve C:\path\to\repo

# tests
go test ./cmd/... ./internal/...      # includes the SQLite store tests
cd web; npm run check                  # tsc + eslint + vitest + prettier
```

**Gotchas worth knowing:**
- **CGO vs fallback.** Real Tree-sitter needs a C toolchain (`CGO_ENABLED=1`). With
  `CGO_ENABLED=0` the binary still builds and runs using the **regex fallback parser** (lower
  fidelity) — handy for toolchain-free cross-compiles.
- **Windows long paths.** Managed git clones live under `<data-dir>/repositories/<id>`; a very
  deep data dir can push `.git` pack paths past the 260-char limit (`fatal: cannot write keep
  file`). Use a short data dir, or add a local folder instead of a Git URL.
- **Stale copy.** The shutdown dialog in `web/src/App.tsx` still says metadata "remains in
  PostgreSQL" — a leftover string from before the SQLite migration; harmless, worth updating.

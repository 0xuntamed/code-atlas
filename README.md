# CodeAtlas

**See your codebase as a map — and the blast radius of any change.**

CodeAtlas parses a JavaScript, TypeScript, Go, or Python repository into a graph of files,
functions, imports, calls, and web routes, then lets you **click any node to light up what a
change to it would affect** — everything that depends on it, and everything it depends on.

It runs entirely on your machine, stores **only structural metadata** (never your source),
uses **no AI**, and ships as **one self-contained binary**: metadata lives in an embedded
SQLite database (no Docker, no external database), and the Go server serves the built React UI
as embedded assets.

- 🗺️ **Whole-repo map** — modules, files, functions, routes, and how they connect.
- 💥 **Blast radius** — select a file, module, or function; dependents (what breaks) turn
  **rose**, dependencies (what it relies on) turn **sky**, the rest dims.
- 🔒 **Local-first & private** — loopback only; source is read, parsed, and discarded.
- 📦 **Single binary** — download (or build) once, point it at a folder, open the page.

---

## Requirements

| Tool | Version | Needed for |
| --- | --- | --- |
| **Go** | 1.26+ | building / running the server |
| **Node.js + npm** | 22.12+ (or 20.19+) | building the frontend bundle once |
| **Git** | any recent | analyzing Git-URL repositories (not needed for local folders) |
| **C toolchain** | optional | full Tree-sitter parsing (see [Parsing modes](#parsing-modes)) |

No Docker, no database server. The SQLite driver is pure Go.

---

## Quick start

From the repository root:

```bash
# 1. Build the frontend once (this writes internal/webui/dist, which the binary embeds)
cd web
npm install
npm run build
cd ..

# 2. Run the server
go run ./cmd/codeatlas serve
```

Then open **http://127.0.0.1:7331** and add a repository (see below).

> **No C compiler?** Build the parser's regex fallback instead — same command, one env var:
> ```bash
> CGO_ENABLED=0 go run ./cmd/codeatlas serve
> ```
> See [Parsing modes](#parsing-modes) for the trade-off.

### Build a standalone binary

```bash
cd web && npm install && npm run build && cd ..
go build -o codeatlas ./cmd/codeatlas      # add CGO_ENABLED=0 for the toolchain-free build
./codeatlas serve
```

---

## Adding a repository

Three ways, all equivalent:

**A. In the browser** — open http://127.0.0.1:7331 and paste either:
- an **absolute local folder path** (recommended — reads in place, nothing is copied), or
- an **HTTPS/SSH Git URL** (CodeAtlas makes a shallow managed clone).

**B. On startup** — point the server at a repo; it registers on first launch:
```bash
go run ./cmd/codeatlas serve C:\absolute\path\to\repo
```

**C. From another terminal** while the server runs:
```bash
go run ./cmd/codeatlas add C:\absolute\path\to\repo
```

Analysis starts immediately and streams live progress. When it finishes, the map appears.

---

## Using CodeAtlas

1. **Read the shape.** The overview shows modules sized by what they hold, wired by the calls
   and imports between them.
2. **Drill in.** Double-click a **module** to see its files; double-click a **file** to see the
   functions and routes it declares. The breadcrumb (top-left) walks you back out.
3. **See a blast radius.** Click **any** node. Its ripple lights up on the same map:
   - **the node itself** — the change epicenter,
   - **rose = breaks** — things that depend on it (upstream),
   - **sky = relies on** — things it depends on (downstream),
   - everything unrelated dims.
4. **Focus the ripple.** Use the **Both / Breaks / Relies on** filter in the sidebar to isolate
   one side.
5. **Expand the detail.** Affected files group their symbols into one “N affected” node — click
   it to reveal the exact functions inside, or **Collapse** to fold them back.
6. **Confirm with the source.** Open a node's inspector to read the real, **hash-verified** code
   without leaving the map.

Two signal filters keep the view honest: **Include tests** and **Reference noise**
(external/unresolved symbols) are off by default.

The header keeps repository actions handy: add another repo, switch projects, reanalyze, remove
derived metadata, or stop the local server.

---

## Configuration

Flags (or environment variables) accepted by `serve`:

| Flag | Env var | Default |
| --- | --- | --- |
| `--listen` | `CODEATLAS_LISTEN_ADDR` | `127.0.0.1:7331` |
| `--data-dir` | `CODEATLAS_DATA_DIR` | OS user-cache dir under `CodeAtlas` |
| `--database` | `CODEATLAS_DATABASE_PATH` | `<data-dir>/codeatlas.db` |

The data directory holds the SQLite file and any managed Git clones. It's created on first run.

---

## Parsing modes

CodeAtlas has two parsers behind a build tag:

- **Tree-sitter (default, needs CGO + a C toolchain)** — accurate syntax-tree parsing of
  JS/TS/Go/Python. This is what a normal `go build` / `go run` uses.
- **Structural fallback (`CGO_ENABLED=0`)** — a conservative regex line-scanner that produces
  the same kind of graph at lower fidelity, with **no C toolchain required**. Ideal for quick
  local runs and toolchain-free cross-compiles.

If a plain build fails with a C-compiler error, either install a C toolchain (e.g. MSYS2/MinGW
on Windows) or prefix the command with `CGO_ENABLED=0`.

---

## Troubleshooting

- **`fatal: cannot write keep file` when adding a Git URL (Windows).** The managed clone path
  crossed Windows' 260-character limit. Use a **short** `--data-dir` (e.g. `C:\atlas`), or add
  the repo as a **local folder path** instead of a Git URL (no clone needed).
- **“Analysis stopped / kept your last valid graph.”** A run failed; the previous graph is
  preserved by design. Check the server log for the reason, fix it, and reanalyze.
- **C-compiler error on build.** See [Parsing modes](#parsing-modes) — use `CGO_ENABLED=0`.
- **Repository too large.** The MVP caps analysis at 10,000 source files.

---

## Development

Run the Go server and Vite side by side; Vite proxies `/api` to the Go server:

```bash
# Terminal 1 — API + analyzer
go run ./cmd/codeatlas serve

# Terminal 2 — frontend with hot reload at http://127.0.0.1:5173
cd web
npm run dev
```

`npm run build` writes the production frontend to `internal/webui/dist`, which
`internal/webui/webui.go` embeds — **rebuild it before compiling the binary** after any
frontend change.

### Tests

```bash
go test ./cmd/... ./internal/...       # Go: parser, analyzer, store (SQLite), API
go vet ./cmd/... ./internal/...
cd web && npm run check                 # tsc + ESLint + Vitest + Prettier
```

---

## Privacy boundary

- Source buffers are read, parsed, and **immediately discarded**. The database stores only
  paths, hashes, source ranges, symbols, relationships, and analysis status.
- Source evidence is re-read from disk on demand and **rejected if its hash changed** since
  analysis.
- The server binds to **loopback** and rejects non-loopback `Host`/`Origin`.
- Git credentials are never accepted in URLs or stored; managed clones defer to the local
  credential helper / SSH agent.
- **No AI provider, analytics, external fonts, or telemetry** — the CSP forbids off-origin
  requests.

---

## How it works (short version)

**Analyze:** discover & classify files → parse with Tree-sitter → build a graph of `entities`
(module/file/function/route…) and `relationships` (contains/defines/imports/calls/
handles_route) → store in SQLite and promote the finished run **atomically** (a failed run
never replaces the last good graph).

**Explore:** the React app reads the graph over the loopback API and draws it with React Flow;
selecting a node requests its **impact map** — a bounded, bidirectional graph walk that tags
each reached node as a dependent or dependency — which the UI paints over the base map.

Supported route detectors: **Express & Next.js** (JS/TS), **net/http & Gin** (Go),
**FastAPI & Flask** (Python).

### Deeper docs

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — architecture & data-flow tour.
- [`docs/CODEBASE.md`](docs/CODEBASE.md) — function-level reference: every module and the full
  flow from adding a repo to the blast radius.

### Project layout

```
cmd/codeatlas     CLI + server entry point
internal/api      loopback security, project lifecycle, graph/impact handlers, SSE, source
internal/analyzer job loop + the analysis pipeline (discover → parse → build → persist)
internal/parser   Tree-sitter adapters (+ no-CGO fallback)
internal/repository  Git acquisition, filesystem discovery, hashing, ignore rules
internal/store    embedded SQLite: migrations, durable jobs, atomic promotion, graph queries
internal/webui    go:embed of the built SPA
web/              React 19 + TypeScript frontend (built into internal/webui/dist)
```

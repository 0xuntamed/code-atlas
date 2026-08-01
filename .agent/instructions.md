# CodeAtlas Agent Instructions

## Project context

CodeAtlas is a local-first code intelligence graph. The backend is written in Go and the frontend is a React/TypeScript application under `web/`.

Preserve these core guarantees:

- Never persist repository source code, credentials, or Git URLs containing credentials.
- Keep the server bound to loopback and retain Host/Origin validation.
- Treat repository paths, source evidence, Git operations, and database queries as security-sensitive.
- Preserve atomic graph promotion: failed or partial analysis must not replace the last valid graph.
- Keep graph traversal and rendering bounded.

## Working rules

1. Read the relevant implementation, tests, and nearby conventions before editing.
2. Keep changes focused; do not modify unrelated files or overwrite existing user work.
3. Add or update tests for behavior changes and regressions.
4. Prefer clear, idiomatic Go and strict TypeScript over clever abstractions.
5. Do not add external services, telemetry, remote fonts, or AI providers.
6. Do not weaken privacy, loopback, path-containment, ignore-rule, or graph-budget protections.
7. Never commit secrets, generated build output, dependency directories, or local database data.

## Repository map

- `cmd/codeatlas`: CLI and server entry points.
- `internal/api`: HTTP API, loopback security, lifecycle, graph, and source evidence.
- `internal/analyzer`: inventory and graph-analysis phases.
- `internal/parser`: Tree-sitter adapters and the no-CGO fallback parser.
- `internal/repository`: repository discovery, safe Git acquisition, hashing, and ignore rules.
- `internal/store`: PostgreSQL persistence, migrations, jobs, search, and traversal.
- `web/src/features`: frontend feature boundaries.
- `web/src/styles`: design-system and feature styles.
- `testdata`: parser and repository fixtures.

## Verification

Run checks proportional to the change. Before committing, use the complete review in `workflows/pre-commit-review.md`.

Backend:

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
```

Frontend:

```powershell
cd web
npm run check
npm run build
```

## Review guidance

- Use `skills/backend-review.md` for Go, API, analyzer, parser, repository, or store changes.
- Use `skills/security-review.md` whenever a change touches trust boundaries, local files, Git, HTTP, source evidence, database access, or user-controlled input.
- Report review findings by severity with file and line references. Do not claim checks passed unless they were run successfully.

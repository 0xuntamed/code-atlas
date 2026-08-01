# Backend Review Skill

Use this review for changes under `cmd/` or `internal/`.

## Review procedure

1. Read the complete diff and the surrounding code paths it affects.
2. Trace inputs from the CLI or HTTP boundary through parsing, business logic, storage, and output.
3. Check behavioral compatibility, error handling, cancellation, cleanup, and concurrency.
4. Verify that tests cover the successful path, invalid input, failures, and important boundary conditions.
5. Run the narrowest relevant tests first, then the full backend checks when practical.

## Checklist

### Correctness

- Errors are returned or wrapped with useful context; none are silently discarded.
- Context cancellation and deadlines propagate through I/O and database calls.
- Resources such as files, response bodies, rows, transactions, and goroutines are closed or terminated.
- Transactions commit only complete state and roll back on every failure path.
- Analysis promotion remains atomic and deterministic.
- Nil, empty, duplicate, stale, partial, and oversized inputs have defined behavior.

### Go quality

- APIs and names are idiomatic and no broader than necessary.
- Interfaces are introduced at the consumer boundary only when they improve testing or substitution.
- Shared mutable state is synchronized and goroutines cannot leak or deadlock.
- Allocation-heavy work is bounded in repository scans and graph traversal.
- Logs are actionable and exclude source text, secrets, and credentials.

### API and persistence

- HTTP status codes and response shapes remain consistent.
- Request bodies, path/query values, and pagination/traversal bounds are validated.
- Database queries are parameterized and preserve project/repository isolation.
- Schema or query changes include migration and compatibility considerations.
- Source evidence is hash-checked before it is returned.

### Verification

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
```

When CGO or parser behavior is affected, exercise both the official Tree-sitter path and the conservative no-CGO fallback where the environment permits.

## Output format

List findings first, ordered by severity (`critical`, `high`, `medium`, `low`). For each finding include:

- the affected file and line;
- the concrete failure or risk;
- how it can be triggered;
- a focused remediation.

Then state tests run, any checks not run, and residual risks. If no findings remain, say so explicitly.

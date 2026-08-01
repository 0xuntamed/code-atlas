# Pre-Commit Review Workflow

Run this workflow after implementation and before creating a commit. It reviews the working tree; it does not stage or commit files.

## 1. Inspect the change set

```powershell
git status --short
git diff --check
git diff --stat
git diff
git diff --cached
```

Confirm that every changed file is intentional. Look for generated output, dependency directories, debug code, temporary files, secrets, credentials, and unrelated edits.

## 2. Review behavior and risk

- Trace each behavior change end to end.
- Confirm tests fail without the fix or otherwise demonstrate the new behavior.
- Apply `../skills/backend-review.md` to backend changes.
- Apply `../skills/security-review.md` to security-sensitive changes.
- Check documentation and examples when commands, configuration, APIs, or user-visible behavior changed.

## 3. Run focused checks

Run the closest tests for the packages or frontend features changed. Fix failures before continuing; do not suppress tests or weaken assertions merely to make them pass.

## 4. Run repository checks

Backend:

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
```

Frontend, when `web/` or shared API behavior changed:

```powershell
cd web
npm run check
npm run build
cd ..
```

If a required tool or service is unavailable, record exactly which check was skipped and why.

## 5. Final diff audit

```powershell
git diff --check
git status --short
```

Re-read the final diff for:

- accidental API or schema changes;
- privacy and security regressions;
- unbounded work or resource leaks;
- missing error handling;
- stale comments or documentation;
- missing tests;
- formatting-only churn outside the task.

Finish with a concise summary containing the changed behavior, checks run and their results, skipped checks, and any remaining risks. Stage or commit only when explicitly requested.

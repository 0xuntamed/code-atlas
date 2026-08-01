# Security Review Skill

Use this review for changes involving HTTP handlers, repository paths, Git, source evidence, database queries, configuration, process control, or any user-controlled input.

## Threat model

Assume an attacker can supply crafted HTTP requests, repository contents, file names, symlinks, Git URLs, ignore rules, symbols, and graph parameters. The application is local-first, but browser-origin attacks and malicious repositories remain in scope.

## Checklist

### Network boundary

- The service remains loopback-only.
- Host and Origin validation cannot be bypassed by alternate casing, ports, proxy headers, IPv6 forms, or missing headers.
- State-changing endpoints enforce the intended method and origin policy.
- Errors do not expose source, credentials, absolute paths unnecessarily, SQL details, or stack traces.
- Request sizes, traversal depths, result counts, and streaming lifetimes are bounded.

### Filesystem and source privacy

- Canonical path and containment checks happen before reads, writes, deletes, or Git operations.
- Symlinks, junctions, traversal segments, case differences, and TOCTOU changes cannot escape the registered repository.
- Ignore and privacy exclusions are applied before source parsing or evidence access.
- PostgreSQL, logs, events, caches, and errors never retain source buffers or snippets.
- Source evidence is served only on demand and only after its current hash matches analyzed metadata.

### Git and process execution

- Credentials embedded in Git URLs are rejected and never logged or stored.
- Commands do not invoke a shell with user-controlled strings.
- Arguments are separated, timeouts/cancellation are enforced, and child processes are cleaned up.
- Clone destinations are unique, contained, and safely cleaned without following links.

### Data and application logic

- SQL is parameterized and every query is scoped to the correct project/repository.
- Authorization-by-identifier assumptions cannot expose another project's metadata or source.
- Untrusted repository content cannot trigger HTML/script injection in the frontend.
- Denial-of-service controls cover repository size, file size, parser work, graph fan-out, and rendered nodes.
- Secrets and sensitive local paths are not introduced into fixtures, snapshots, or configuration defaults.

### Dependencies and configuration

- New dependencies are necessary, maintained, pinned through lockfiles, and do not add unwanted network behavior.
- Production defaults remain private and secure; development conveniences do not weaken them.
- `.env.example` contains placeholders only.

## Verification

Run targeted security and regression tests for every affected trust boundary, followed by the applicable backend and frontend checks. Treat a missing negative test for a newly handled hostile input as a review finding.

## Output format

Report findings first, ordered by exploitability and impact. Include the affected file and line, attack scenario, impact, and recommended fix. Clearly distinguish confirmed vulnerabilities from defense-in-depth suggestions, then list tests run and residual risks.

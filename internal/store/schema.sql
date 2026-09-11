-- SQLite schema. Timestamps are stored as INTEGER Unix microseconds (UTC),
-- booleans as INTEGER 0/1, and JSON metadata as TEXT. Foreign keys are enforced
-- via the foreign_keys pragma set on every pooled connection (see store.Open).

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('local', 'git')),
    root_path TEXT NOT NULL,
    remote_url TEXT NOT NULL DEFAULT '',
    git_ref TEXT NOT NULL DEFAULT '',
    current_commit TEXT NOT NULL DEFAULT '',
    managed_clone INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'queued',
    active_run_id TEXT REFERENCES analysis_runs(id) ON DELETE SET NULL,
    created_at INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS analysis_runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued',
    stage TEXT NOT NULL DEFAULT 'queued',
    completed INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    entities INTEGER NOT NULL DEFAULT 0,
    relationships INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    lease_until INTEGER,
    started_at INTEGER,
    completed_at INTEGER,
    created_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL REFERENCES analysis_runs(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    classification TEXT NOT NULL,
    ignore_reason TEXT NOT NULL DEFAULT '',
    is_directory INTEGER NOT NULL DEFAULT 0,
    is_test INTEGER NOT NULL DEFAULT 0,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    content_hash TEXT NOT NULL DEFAULT '',
    UNIQUE (run_id, path)
);

CREATE TABLE IF NOT EXISTS entities (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL REFERENCES analysis_runs(id) ON DELETE CASCADE,
    file_id TEXT REFERENCES files(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    start_line INTEGER NOT NULL DEFAULT 0,
    start_column INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_column INTEGER NOT NULL DEFAULT 0,
    is_test INTEGER NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS relationships (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL REFERENCES analysis_runs(id) ON DELETE CASCADE,
    source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 1,
    evidence_file_id TEXT REFERENCES files(id) ON DELETE SET NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_column INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_column INTEGER NOT NULL DEFAULT 0,
    resolution_state TEXT NOT NULL DEFAULT 'resolved',
    metadata TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS analysis_runs_claim_idx
    ON analysis_runs (status, lease_until, created_at);
CREATE INDEX IF NOT EXISTS files_project_run_idx
    ON files (project_id, run_id, classification);
CREATE INDEX IF NOT EXISTS entities_project_run_idx
    ON entities (project_id, run_id, kind);
CREATE INDEX IF NOT EXISTS entities_qualified_name_idx
    ON entities (qualified_name);
CREATE INDEX IF NOT EXISTS entities_name_idx
    ON entities (name);
CREATE INDEX IF NOT EXISTS relationships_source_idx
    ON relationships (project_id, run_id, source_entity_id, relationship_type);
CREATE INDEX IF NOT EXISTS relationships_target_idx
    ON relationships (project_id, run_id, target_entity_id, relationship_type);

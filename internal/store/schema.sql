CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('local', 'git')),
    root_path TEXT NOT NULL,
    remote_url TEXT NOT NULL DEFAULT '',
    git_ref TEXT NOT NULL DEFAULT '',
    current_commit TEXT NOT NULL DEFAULT '',
    managed_clone BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'queued',
    active_run_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS analysis_runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued',
    stage TEXT NOT NULL DEFAULT 'queued',
    completed INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    lease_until TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_active_run_id_fkey;
ALTER TABLE projects
    ADD CONSTRAINT projects_active_run_id_fkey
    FOREIGN KEY (active_run_id) REFERENCES analysis_runs(id) ON DELETE SET NULL
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL REFERENCES analysis_runs(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    classification TEXT NOT NULL,
    ignore_reason TEXT NOT NULL DEFAULT '',
    is_directory BOOLEAN NOT NULL DEFAULT FALSE,
    is_test BOOLEAN NOT NULL DEFAULT FALSE,
    size_bytes BIGINT NOT NULL DEFAULT 0,
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
    is_test BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS relationships (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL REFERENCES analysis_runs(id) ON DELETE CASCADE,
    source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1,
    evidence_file_id TEXT REFERENCES files(id) ON DELETE SET NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_column INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_column INTEGER NOT NULL DEFAULT 0,
    resolution_state TEXT NOT NULL DEFAULT 'resolved',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS analysis_runs_claim_idx
    ON analysis_runs (status, lease_until, created_at);
CREATE INDEX IF NOT EXISTS files_project_run_idx
    ON files (project_id, run_id, classification);
CREATE INDEX IF NOT EXISTS entities_project_run_idx
    ON entities (project_id, run_id, kind);
CREATE INDEX IF NOT EXISTS entities_qualified_name_trgm_idx
    ON entities USING gin (qualified_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS relationships_source_idx
    ON relationships (project_id, run_id, source_entity_id, relationship_type);
CREATE INDEX IF NOT EXISTS relationships_target_idx
    ON relationships (project_id, run_id, target_entity_id, relationship_type);


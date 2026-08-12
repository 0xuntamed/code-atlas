package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/codeatlas/codeatlas/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var ErrNotFound = errors.New("not found")

// leaseDuration is how long a claimed run stays leased before another worker
// (or the same worker after a crash) may reclaim it.
const leaseDuration = 45 * time.Second

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path. The bundled
// driver is pure Go, so no external database process or C toolchain is needed.
func Open(ctx context.Context, path string) (*Store, error) {
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(8)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() { _ = s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	for _, statement := range splitStatements(schemaSQL) {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply database schema: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateProject(ctx context.Context, project model.Project, run model.AnalysisRun) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := nowMicros()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO projects (id, name, source_type, root_path, remote_url, git_ref, managed_clone, status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		project.ID, project.Name, project.SourceType, project.RootPath, project.RemoteURL,
		project.GitRef, boolInt(project.ManagedClone), model.ProjectQueued, now, now)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO analysis_runs (id, project_id, status, stage, message, created_at)
		VALUES (?,?,?,'queued','Waiting for the local analyzer',?)`, run.ID, project.ID, model.RunQueued, now)
	if err != nil {
		return fmt.Errorf("insert analysis run: %w", err)
	}
	return tx.Commit()
}

func (s *Store) QueueAnalysis(ctx context.Context, projectID, runID string) error {
	now := nowMicros()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO analysis_runs (id, project_id, status, stage, message, created_at)
		SELECT ?, id, 'queued', 'queued', 'Waiting for the local analyzer', ?
		FROM projects WHERE id=?`, runID, now, projectID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	_, err = s.db.ExecContext(ctx, `UPDATE projects SET status='queued', updated_at=? WHERE id=?`, now, projectID)
	return err
}

func scanRun(row scanner) (*model.AnalysisRun, error) {
	var run model.AnalysisRun
	var started, completed sql.NullInt64
	var created int64
	err := row.Scan(&run.ID, &run.ProjectID, &run.Status, &run.Stage, &run.Completed,
		&run.Total, &run.Message, &run.ErrorMessage, &started, &completed, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	run.StartedAt = nullTime(started)
	run.CompletedAt = nullTime(completed)
	run.CreatedAt = microsToTime(created)
	return &run, nil
}

const runColumns = `id, project_id, status, stage, completed, total, message, error_message, started_at, completed_at, created_at`

func (s *Store) LatestRun(ctx context.Context, projectID string) (*model.AnalysisRun, error) {
	return scanRun(s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM analysis_runs WHERE project_id=? ORDER BY created_at DESC LIMIT 1`, projectID))
}

func (s *Store) ClaimRun(ctx context.Context) (*model.AnalysisRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := nowMicros()
	row := tx.QueryRowContext(ctx, `
		SELECT `+runColumns+` FROM analysis_runs
		WHERE status='queued' OR (status='running' AND lease_until < ?)
		ORDER BY created_at LIMIT 1`, now)
	run, err := scanRun(row)
	if err != nil {
		return nil, err
	}
	lease := microsFromNow(leaseDuration)
	_, err = tx.ExecContext(ctx, `
		UPDATE analysis_runs SET status='running', stage='acquiring', started_at=COALESCE(started_at,?),
			lease_until=?, message='Preparing repository'
		WHERE id=?`, now, lease, run.ID)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE projects SET status='analyzing', updated_at=? WHERE id=?`, now, run.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	run.Status = model.RunRunning
	run.Stage = "acquiring"
	return run, nil
}

func (s *Store) UpdateRun(ctx context.Context, runID, stage string, completed, total int, message string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE analysis_runs SET stage=?, completed=?, total=?, message=?, lease_until=? WHERE id=?`,
		stage, completed, total, message, microsFromNow(leaseDuration), runID)
	return err
}

func (s *Store) FailRun(ctx context.Context, runID, projectID, safeMessage string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		UPDATE analysis_runs SET status='failed', stage='failed', error_message=?,
			message='Analysis failed', lease_until=NULL, completed_at=? WHERE id=?`, safeMessage, nowMicros(), runID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE projects SET status='failed', updated_at=? WHERE id=?`, nowMicros(), projectID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Project(ctx context.Context, id string) (*model.Project, error) {
	project, err := scanProject(s.db.QueryRowContext(ctx, `
		SELECT id,name,source_type,root_path,remote_url,git_ref,current_commit,managed_clone,status,
			active_run_id,created_at,updated_at FROM projects WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	if run, err := s.LatestRun(ctx, id); err == nil {
		project.LatestRun = run
	}
	return project, nil
}

func (s *Store) Projects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,name,source_type,root_path,remote_url,git_ref,current_commit,managed_clone,status,
			active_run_id,created_at,updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	projects := make([]model.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		projects = append(projects, *project)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range projects {
		if run, err := s.LatestRun(ctx, projects[index].ID); err == nil {
			projects[index].LatestRun = run
		}
	}
	return projects, nil
}

func scanProject(row scanner) (*model.Project, error) {
	var p model.Project
	var managed int
	var active sql.NullString
	var created, updated int64
	err := row.Scan(&p.ID, &p.Name, &p.SourceType, &p.RootPath, &p.RemoteURL, &p.GitRef,
		&p.CurrentCommit, &managed, &p.Status, &active, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ManagedClone = managed != 0
	p.ActiveRunID = active.String
	p.CreatedAt = microsToTime(created)
	p.UpdatedAt = microsToTime(updated)
	return &p, nil
}

func (s *Store) UpdateProjectCommit(ctx context.Context, projectID, commit string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE projects SET current_commit=?, updated_at=? WHERE id=?`, commit, nowMicros(), projectID)
	return err
}

func (s *Store) ReplaceRunData(ctx context.Context, runID string, files []model.FileRecord, entities []model.Entity, relationships []model.Relationship) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Delete children before parents so foreign keys stay satisfied.
	for _, table := range []string{"relationships", "entities", "files"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE run_id=?`, runID); err != nil {
			return err
		}
	}

	fileStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO files (id, project_id, run_id, path, language, classification, ignore_reason, is_directory, is_test, size_bytes, content_hash)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer fileStmt.Close()
	for _, f := range files {
		if _, err := fileStmt.ExecContext(ctx, f.ID, f.ProjectID, f.RunID, f.Path, f.Language, f.Classification,
			f.IgnoreReason, boolInt(f.IsDirectory), boolInt(f.IsTest), f.SizeBytes, f.ContentHash); err != nil {
			return fmt.Errorf("insert file metadata: %w", err)
		}
	}

	entityStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO entities (id, project_id, run_id, file_id, kind, name, qualified_name, language, start_line, start_column, end_line, end_column, is_test, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer entityStmt.Close()
	for _, e := range entities {
		if _, err := entityStmt.ExecContext(ctx, e.ID, e.ProjectID, e.RunID, nullString(e.FileID), e.Kind, e.Name,
			e.QualifiedName, e.Language, e.Range.StartLine, e.Range.StartColumn, e.Range.EndLine, e.Range.EndColumn,
			boolInt(e.IsTest), marshalMeta(e.Metadata)); err != nil {
			return fmt.Errorf("insert entity metadata: %w", err)
		}
	}

	relationshipStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO relationships (id, project_id, run_id, source_entity_id, target_entity_id, relationship_type, confidence, evidence_file_id, start_line, start_column, end_line, end_column, resolution_state, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer relationshipStmt.Close()
	for _, r := range relationships {
		if _, err := relationshipStmt.ExecContext(ctx, r.ID, r.ProjectID, r.RunID, r.SourceID, r.TargetID, r.Kind,
			r.Confidence, nullString(r.EvidenceFileID), r.Range.StartLine, r.Range.StartColumn, r.Range.EndLine,
			r.Range.EndColumn, r.Resolution, marshalMeta(r.Metadata)); err != nil {
			return fmt.Errorf("insert relationship metadata: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) PromoteRun(ctx context.Context, runID, projectID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE analysis_runs SET status='ready',stage='ready',message='Analysis complete',lease_until=NULL,completed_at=? WHERE id=?`, nowMicros(), runID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE projects SET active_run_id=?,status='ready',updated_at=? WHERE id=?`, runID, nowMicros(), projectID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	cutoff := microsFromNow(-time.Minute)
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM analysis_runs WHERE project_id=? AND id<>? AND status='ready' AND created_at < ?`, projectID, runID, cutoff)
	return nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Files(ctx context.Context, projectID, classification string, limit int) ([]model.FileRecord, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id,f.project_id,f.run_id,f.path,f.language,f.classification,f.ignore_reason,f.is_directory,f.size_bytes,f.is_test
		FROM files f JOIN projects p ON p.active_run_id=f.run_id
		WHERE f.project_id=? AND (?='' OR f.classification=?)
		ORDER BY f.path LIMIT ?`, projectID, classification, classification, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.FileRecord, 0)
	for rows.Next() {
		var f model.FileRecord
		var isDirectory, isTest int
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.RunID, &f.Path, &f.Language, &f.Classification, &f.IgnoreReason,
			&isDirectory, &f.SizeBytes, &isTest); err != nil {
			return nil, err
		}
		f.IsDirectory = isDirectory != 0
		f.IsTest = isTest != 0
		result = append(result, f)
	}
	return result, rows.Err()
}

func scanEntity(row scanner) (*model.Entity, error) {
	var e model.Entity
	var metadata []byte
	var isTest int
	err := row.Scan(&e.ID, &e.ProjectID, &e.RunID, &e.FileID, &e.Kind, &e.Name, &e.QualifiedName, &e.Language,
		&e.Range.StartLine, &e.Range.StartColumn, &e.Range.EndLine, &e.Range.EndColumn, &isTest, &metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	e.IsTest = isTest != 0
	_ = json.Unmarshal(metadata, &e.Metadata)
	return &e, nil
}

const entityColumns = `e.id,e.project_id,e.run_id,COALESCE(e.file_id,''),e.kind,e.name,e.qualified_name,e.language,
	e.start_line,e.start_column,e.end_line,e.end_column,e.is_test,e.metadata`

func (s *Store) Entity(ctx context.Context, projectID, entityID string) (*model.Entity, error) {
	return scanEntity(s.db.QueryRowContext(ctx, `SELECT `+entityColumns+` FROM entities e JOIN projects p ON p.active_run_id=e.run_id WHERE e.project_id=? AND e.id=?`, projectID, entityID))
}

func (s *Store) Search(ctx context.Context, projectID, query string, limit int) ([]model.Entity, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `SELECT `+entityColumns+` FROM entities e JOIN projects p ON p.active_run_id=e.run_id
		WHERE e.project_id=? AND (e.qualified_name LIKE ? OR e.name LIKE ?)
		ORDER BY CASE WHEN lower(e.name)=lower(?) THEN 0 ELSE 1 END, length(e.qualified_name), e.qualified_name LIMIT ?`,
		projectID, pattern, pattern, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Entity, 0)
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *e)
	}
	return result, rows.Err()
}

type EvidenceRecord struct {
	Entity       model.Entity
	RootPath     string
	RelativePath string
	ContentHash  string
}

func (s *Store) Evidence(ctx context.Context, projectID, entityID string) (*EvidenceRecord, error) {
	var record EvidenceRecord
	var metadata []byte
	var isTest int
	err := s.db.QueryRowContext(ctx, `
		SELECT `+entityColumns+`,p.root_path,f.path,f.content_hash
		FROM entities e JOIN projects p ON p.active_run_id=e.run_id JOIN files f ON f.id=e.file_id
		WHERE e.project_id=? AND e.id=?`, projectID, entityID).
		Scan(&record.Entity.ID, &record.Entity.ProjectID, &record.Entity.RunID, &record.Entity.FileID, &record.Entity.Kind,
			&record.Entity.Name, &record.Entity.QualifiedName, &record.Entity.Language, &record.Entity.Range.StartLine,
			&record.Entity.Range.StartColumn, &record.Entity.Range.EndLine, &record.Entity.Range.EndColumn, &isTest,
			&metadata, &record.RootPath, &record.RelativePath, &record.ContentHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	record.Entity.IsTest = isTest != 0
	_ = json.Unmarshal(metadata, &record.Entity.Metadata)
	return &record, nil
}

// --- helpers ---

func nowMicros() int64 { return time.Now().UTC().UnixMicro() }

func microsFromNow(d time.Duration) int64 { return time.Now().Add(d).UTC().UnixMicro() }

func microsToTime(v int64) time.Time { return time.UnixMicro(v).UTC() }

func nullTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := microsToTime(v.Int64)
	return &t
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func marshalMeta(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func splitStatements(script string) []string {
	parts := strings.Split(script, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		statements = append(statements, part)
	}
	return statements
}

// placeholders returns "?,?,..." with n placeholders for building IN clauses.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// stringArgs converts a []string to []any for variadic query arguments.
func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}
	return args
}

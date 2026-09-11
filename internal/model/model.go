package model

import "time"

const (
	SourceLocal = "local"
	SourceGit   = "git"

	ProjectQueued    = "queued"
	ProjectAnalyzing = "analyzing"
	ProjectReady     = "ready"
	ProjectFailed    = "failed"

	RunQueued  = "queued"
	RunRunning = "running"
	RunReady   = "ready"
	RunFailed  = "failed"
)

var EntityKinds = []string{
	"package", "module", "file", "function", "method", "class", "struct",
	"interface", "route", "external_symbol", "unresolved_symbol",
}

var RelationshipKinds = []string{
	"contains", "defines", "imports", "exports", "calls", "handles_route",
	"uses_middleware", "depends_on",
}

type Project struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	SourceType    string       `json:"sourceType"`
	RootPath      string       `json:"rootPath"`
	RemoteURL     string       `json:"remoteUrl,omitempty"`
	GitRef        string       `json:"gitRef,omitempty"`
	CurrentCommit string       `json:"currentCommit,omitempty"`
	ManagedClone  bool         `json:"managedClone"`
	Status        string       `json:"status"`
	ActiveRunID   string       `json:"activeRunId,omitempty"`
	LatestRun     *AnalysisRun `json:"latestRun,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

type AnalysisRun struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"projectId"`
	Status       string     `json:"status"`
	Stage        string     `json:"stage"`
	Completed    int        `json:"completed"`
	Total        int        `json:"total"`
	Entities     int        `json:"entities"`
	Relationships int       `json:"relationships"`
	Message      string     `json:"message,omitempty"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type FileRecord struct {
	ID             string `json:"id"`
	ProjectID      string `json:"projectId"`
	RunID          string `json:"runId"`
	Path           string `json:"path"`
	Language       string `json:"language,omitempty"`
	Classification string `json:"classification"`
	IgnoreReason   string `json:"ignoreReason,omitempty"`
	IsDirectory    bool   `json:"isDirectory"`
	SizeBytes      int64  `json:"sizeBytes"`
	ContentHash    string `json:"-"`
	IsTest         bool   `json:"isTest"`
}

type Range struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

type Entity struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"projectId"`
	RunID         string         `json:"runId"`
	FileID        string         `json:"fileId,omitempty"`
	Kind          string         `json:"kind"`
	Name          string         `json:"name"`
	QualifiedName string         `json:"qualifiedName"`
	Language      string         `json:"language,omitempty"`
	Range         Range          `json:"range"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Distance      int            `json:"distance,omitempty"`
	// Direction relative to an impact root: root, dependent (upstream, breaks if
	// the root changes), dependency (downstream, relied upon), or both.
	Direction string `json:"direction,omitempty"`
	IsTest    bool   `json:"isTest,omitempty"`
}

type Relationship struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	RunID          string         `json:"runId"`
	SourceID       string         `json:"source"`
	TargetID       string         `json:"target"`
	Kind           string         `json:"kind"`
	Confidence     float64        `json:"confidence"`
	EvidenceFileID string         `json:"evidenceFileId,omitempty"`
	Range          Range          `json:"range"`
	Resolution     string         `json:"resolution"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type Graph struct {
	Nodes     []Entity       `json:"nodes"`
	Edges     []Relationship `json:"edges"`
	RootID    string         `json:"rootId,omitempty"`
	Truncated bool           `json:"truncated"`
	Limit     int            `json:"limit"`
	MaxDepth  int            `json:"maxDepth,omitempty"`
}

type SourceEvidence struct {
	EntityID  string `json:"entityId"`
	FilePath  string `json:"filePath"`
	Language  string `json:"language"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Code      string `json:"code"`
}

type ProjectSource struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	URL  string `json:"url,omitempty"`
	Ref  string `json:"ref,omitempty"`
}

type CreateProjectRequest struct {
	Source ProjectSource `json:"source"`
}

type CreateProjectResponse struct {
	ProjectID     string `json:"projectId"`
	AnalysisRunID string `json:"analysisRunId"`
	Status        string `json:"status"`
}

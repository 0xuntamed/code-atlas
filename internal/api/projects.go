package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeatlas/codeatlas/internal/id"
	"github.com/codeatlas/codeatlas/internal/model"
	"github.com/codeatlas/codeatlas/internal/repository"
)

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.Projects(r.Context())
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.store.Project(r.Context(), r.PathValue("projectID"))
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var request model.CreateProjectRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	projectID := id.New()
	runID := id.New()
	project, err := s.newProject(projectID, request.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Code, err.Message)
		return
	}

	run := model.AnalysisRun{ID: runID, ProjectID: projectID}
	if err := s.store.CreateProject(r.Context(), project, run); err != nil {
		s.logger.Error("create project", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Could not create project")
		return
	}

	writeJSON(w, http.StatusAccepted, model.CreateProjectResponse{
		ProjectID:     projectID,
		AnalysisRunID: runID,
		Status:        model.RunQueued,
	})
}

type projectInputError struct {
	Code    string
	Message string
}

func (s *Server) newProject(projectID string, source model.ProjectSource) (model.Project, *projectInputError) {
	project := model.Project{
		ID:         projectID,
		SourceType: source.Type,
		GitRef:     strings.TrimSpace(source.Ref),
		Status:     model.ProjectQueued,
	}

	switch source.Type {
	case model.SourceLocal:
		return s.newLocalProject(project, source.Path)
	case model.SourceGit:
		return s.newGitProject(project, source.URL)
	default:
		return model.Project{}, &projectInputError{
			Code:    "invalid_source",
			Message: "source.type must be local or git",
		}
	}
}

func (s *Server) newLocalProject(project model.Project, inputPath string) (model.Project, *projectInputError) {
	absolute, err := filepath.Abs(strings.TrimSpace(inputPath))
	if err != nil {
		return model.Project{}, &projectInputError{Code: "invalid_path", Message: "Local path is invalid"}
	}

	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return model.Project{}, &projectInputError{Code: "invalid_path", Message: "Local path does not exist"}
	}

	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return model.Project{}, &projectInputError{Code: "invalid_path", Message: "Local path must be a directory"}
	}

	project.RootPath = canonical
	project.Name = filepath.Base(canonical)
	return project, nil
}

func (s *Server) newGitProject(project model.Project, rawURL string) (model.Project, *projectInputError) {
	if err := repository.ValidateGitURL(rawURL); err != nil {
		return model.Project{}, &projectInputError{Code: "invalid_git_url", Message: err.Error()}
	}

	project.RemoteURL = strings.TrimSpace(rawURL)
	project.ManagedClone = true
	project.RootPath = filepath.Join(s.dataDir, "repositories", project.ID)
	project.Name = gitProjectName(project.RemoteURL)
	return project, nil
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.store.Project(r.Context(), r.PathValue("projectID"))
	if handleStoreError(w, err) {
		return
	}
	if err := s.store.DeleteProject(r.Context(), project.ID); handleStoreError(w, err) {
		return
	}

	if project.ManagedClone && isWithin(s.dataDir, project.RootPath) {
		if err := os.RemoveAll(project.RootPath); err != nil {
			s.logger.Warn("remove managed clone", "project_id", project.ID, "error", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

package repository

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codeatlas/codeatlas/internal/model"
)

func ValidateGitURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("Git URL is required")
	}
	if strings.ContainsAny(raw, "\r\n\x00") {
		return fmt.Errorf("Git URL contains invalid characters")
	}
	if strings.HasPrefix(raw, "git@") {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid Git URL")
	}
	if u.User != nil {
		return fmt.Errorf("credential-bearing Git URLs are not allowed")
	}
	if u.Scheme != "https" && u.Scheme != "ssh" {
		return fmt.Errorf("Git URL must use HTTPS or SSH")
	}
	if u.Host == "" {
		return fmt.Errorf("Git URL host is required")
	}
	return nil
}

func Acquire(ctx context.Context, project *model.Project) (string, error) {
	if project.SourceType == model.SourceLocal {
		return "", nil
	}
	if err := ValidateGitURL(project.RemoteURL); err != nil {
		return "", err
	}
	if info, err := os.Stat(filepath.Join(project.RootPath, ".git")); err == nil && info.IsDir() {
		if err := runGit(ctx, project.RootPath, "fetch", "--depth=1", "origin"); err != nil {
			return "", err
		}
		ref := project.GitRef
		if ref == "" {
			ref = "origin/HEAD"
		}
		if err := runGit(ctx, project.RootPath, "checkout", "--force", ref); err != nil {
			return "", err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(project.RootPath), 0o700); err != nil {
			return "", fmt.Errorf("prepare managed repository directory: %w", err)
		}
		args := []string{"clone", "--depth=1"}
		if project.GitRef != "" {
			args = append(args, "--branch", project.GitRef)
		}
		args = append(args, "--", project.RemoteURL, project.RootPath)
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("clone repository: %s", safeGitError(output))
		}
	}
	cmd := exec.CommandContext(ctx, "git", "-C", project.RootPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read repository commit")
	}
	return strings.TrimSpace(string(output)), nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	all := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", all...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Git operation failed: %s", safeGitError(output))
	}
	return nil
}

func safeGitError(output []byte) string {
	message := strings.TrimSpace(string(output))
	if len(message) > 240 {
		message = message[:240]
	}
	if message == "" {
		return "Git command failed"
	}
	if strings.Contains(strings.ToLower(message), "password") || strings.Contains(strings.ToLower(message), "token") {
		return "authentication failed; use the local Git credential helper or SSH agent"
	}
	return message
}

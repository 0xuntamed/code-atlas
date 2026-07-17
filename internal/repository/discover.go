package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeatlas/codeatlas/internal/id"
	"github.com/codeatlas/codeatlas/internal/model"
)

const MaxSourceFileSize int64 = 1 << 20

type DiscoveredFile struct {
	Record       model.FileRecord
	AbsolutePath string
}

var sourceExtensions = map[string]string{
	".js": "javascript", ".jsx": "javascript", ".mjs": "javascript", ".cjs": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".mts": "typescript", ".cts": "typescript",
	".go": "go", ".py": "python", ".pyi": "python",
}

var supportFiles = map[string]bool{
	"package.json": true, "tsconfig.json": true, "jsconfig.json": true, "go.mod": true, "go.work": true,
	"pyproject.toml": true, "requirements.txt": true, "setup.py": true,
}

var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true, ".pdf": true,
	".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true, ".exe": true, ".dll": true, ".so": true,
	".dylib": true, ".a": true, ".o": true, ".class": true, ".jar": true, ".woff": true, ".woff2": true, ".ttf": true,
	".mp3": true, ".mp4": true, ".mov": true, ".avi": true, ".db": true, ".sqlite": true,
}

func Discover(ctx context.Context, projectID, runID, root string) ([]DiscoveredFile, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, err
	}
	matcher := loadIgnoreMatcher(canonical)
	result := make([]DiscoveredFile, 0, 512)
	err = filepath.WalkDir(canonical, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("inspect repository entry: %w", walkErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if current == canonical {
			return nil
		}
		rel, err := filepath.Rel(canonical, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read repository metadata: %w", err)
		}
		record := model.FileRecord{ID: id.Stable(runID, "file", rel), ProjectID: projectID, RunID: runID, Path: rel,
			IsDirectory: entry.IsDir(), SizeBytes: info.Size(), Classification: "skipped"}
		if info.Mode()&os.ModeSymlink != 0 {
			record.IgnoreReason = "hard safety: symlink not followed"
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if hard, reason := hardExcluded(rel, entry.IsDir(), info.Size()); hard {
			record.IgnoreReason = reason
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if ignored, reason := matcher.match(rel, entry.IsDir()); ignored {
			record.IgnoreReason = reason
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if binaryExtensions[ext] || looksBinary(current) {
			record.IgnoreReason = "hard safety: binary file"
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			return nil
		}
		if language, ok := sourceExtensions[ext]; ok {
			record.Classification = "source"
			record.Language = language
			record.IsTest = isTestFile(rel)
			hash, err := hashFile(current)
			if err != nil {
				return err
			}
			record.ContentHash = hash
		} else if supportFiles[entry.Name()] {
			record.Classification = "support"
			record.IgnoreReason = "read transiently for module resolution"
		} else {
			record.IgnoreReason = "unsupported file type"
		}
		result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
		return nil
	})
	return result, err
}

func hardExcluded(relative string, isDir bool, size int64) (bool, string) {
	base := strings.ToLower(filepath.Base(relative))
	ext := strings.ToLower(filepath.Ext(base))
	if !isDir && (strings.HasPrefix(base, ".env") || ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx" ||
		base == "id_rsa" || base == "id_ed25519" || base == "credentials" || base == ".npmrc" || base == ".pypirc") {
		return true, "hard safety: secret-like file"
	}
	if !isDir && size > MaxSourceFileSize {
		return true, "hard safety: file exceeds 1 MiB"
	}
	return false, ""
}

func looksBinary(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return false
	}
	return bytes.IndexByte(buf[:n], 0) >= 0
}

func hashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("hash source file: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash source file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashFile(filePath string) (string, error) { return hashFile(filePath) }

func isTestFile(relative string) bool {
	p := strings.ToLower(filepath.ToSlash(relative))
	base := strings.ToLower(filepath.Base(relative))
	return strings.Contains(p, "/test/") || strings.Contains(p, "/tests/") || strings.Contains(p, "/__tests__/") ||
		strings.HasSuffix(base, "_test.go") || strings.HasPrefix(base, "test_") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.")
}

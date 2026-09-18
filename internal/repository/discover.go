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
	"runtime"
	"strings"
	"sync"

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
	var tasks []ioTask
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
		if binaryExtensions[ext] {
			record.IgnoreReason = "hard safety: binary file"
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			return nil
		}
		if language, ok := sourceExtensions[ext]; ok {
			// A source file needs its content hashed; that I/O is deferred to the
			// parallel pass below (hashing every file serially dominates discovery).
			record.Classification = "source"
			record.Language = language
			record.IsTest = isTestFile(rel)
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			tasks = append(tasks, ioTask{index: len(result) - 1, kind: taskHash})
			return nil
		}
		if supportFiles[entry.Name()] {
			record.Classification = "support"
			record.IgnoreReason = "read transiently for module resolution"
			result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
			return nil
		}
		// Unknown type: sniff for binary content (also deferred I/O). Defaults to
		// "unsupported file type" unless the sniff finds it binary.
		record.IgnoreReason = "unsupported file type"
		result = append(result, DiscoveredFile{Record: record, AbsolutePath: current})
		tasks = append(tasks, ioTask{index: len(result) - 1, kind: taskSniff})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := runIOTasks(ctx, result, tasks); err != nil {
		return nil, err
	}
	return result, nil
}

const (
	taskHash  = 1 // hash a source file's contents
	taskSniff = 2 // sniff an unknown file for binary content
)

type ioTask struct {
	index int
	kind  int
}

// runIOTasks performs discovery's per-file I/O (content hashing and binary
// sniffing) across a worker pool. Each task writes a distinct result element,
// so no locking is needed on the slice itself. This turns discovery from a
// serial file-by-file read into a parallel one — the dominant cost on large
// repositories, especially on Windows where each file open is expensive.
func runIOTasks(ctx context.Context, result []DiscoveredFile, tasks []ioTask) error {
	if len(tasks) == 0 {
		return ctx.Err()
	}
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan ioTask)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				path := result[task.index].AbsolutePath
				switch task.kind {
				case taskHash:
					hash, err := hashFile(path)
					if err != nil {
						mu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						mu.Unlock()
						continue
					}
					result[task.index].Record.ContentHash = hash
				case taskSniff:
					if looksBinary(path) {
						result[task.index].Record.IgnoreReason = "hard safety: binary file"
					}
				}
			}
		}()
	}
feed:
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			break feed
		case jobs <- task:
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
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

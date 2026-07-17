package api

import (
	"path/filepath"
	"strings"
)

func gitProjectName(rawURL string) string {
	value := strings.TrimSuffix(strings.TrimSpace(rawURL), "/")
	value = strings.TrimSuffix(value, ".git")
	if index := strings.LastIndexAny(value, "/:"); index >= 0 {
		value = value[index+1:]
	}
	if value == "" {
		return "repository"
	}
	return value
}

func isWithin(root, target string) bool {
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbsolute, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootAbsolute, targetAbsolute)
	return err == nil &&
		relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

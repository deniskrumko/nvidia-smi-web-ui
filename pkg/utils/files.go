package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ReadVersionFile reads and trims a version file, returning fallback on errors or empty content.
func ReadVersionFile(path string, fallback string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return TextOrDefault(strings.TrimSpace(string(content)), fallback)
}

// DefaultAssetDir returns the first existing directory candidate, otherwise fallback.
func DefaultAssetDir(fallback string, candidates ...string) string {
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return fallback
}

// PackageDir returns the directory of the caller's source file.
func PackageDir() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

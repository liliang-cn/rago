package agent

import (
	"path/filepath"
	"slices"
	"strings"
)

var defaultRepositoryIgnoreNames = []string{
	".git",
	".hg",
	".svn",
	".next",
	".nuxt",
	".turbo",
	".cache",
	".venv",
	"__pycache__",
	"node_modules",
	"vendor",
	"target",
	"dist",
	"build",
	"coverage",
	"tmp",
	"temp",
	"out",
	"bin",
}

func DefaultRepositoryIgnoreNames() []string {
	return append([]string(nil), defaultRepositoryIgnoreNames...)
}

func FormatRepositoryIgnoreList() string {
	return strings.Join(defaultRepositoryIgnoreNames, ", ")
}

func IsBlacklistedRepositoryPath(path string) bool {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return false
	}

	parts := strings.Split(filepath.ToSlash(cleaned), "/")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if slices.Contains(defaultRepositoryIgnoreNames, part) {
			return true
		}
	}
	return false
}

package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
)

func sanitizeRelativePath(input string, allowEmpty bool) (string, error) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimLeft(trimmed, "/\\")

	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("path is required")
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("path is required")
	}

	if filepath.IsAbs(cleaned) || filepath.VolumeName(cleaned) != "" {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	return cleaned, nil
}

func resolvePathWithinRoot(root, relativePath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, relativePath))
	if err != nil {
		return "", err
	}

	relToRoot, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}

	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relToRoot) {
		return "", fmt.Errorf("path escapes the configured root")
	}

	return targetAbs, nil
}

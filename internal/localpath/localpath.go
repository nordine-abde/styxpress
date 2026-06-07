package localpath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanRequired(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("path is required")
	}
	if strings.Contains(value, "\x00") {
		return "", errors.New("path contains NUL byte")
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func EnsureSeparateRoots(firstName string, firstPath string, secondName string, secondPath string) error {
	firstCandidates, err := pathCandidates(firstPath)
	if err != nil {
		return fmt.Errorf("%s: %w", firstName, err)
	}
	secondCandidates, err := pathCandidates(secondPath)
	if err != nil {
		return fmt.Errorf("%s: %w", secondName, err)
	}
	for _, first := range firstCandidates {
		for _, second := range secondCandidates {
			if sameOrNested(first, second) || sameOrNested(second, first) {
				return fmt.Errorf("%s and %s must not be the same directory or nested", firstName, secondName)
			}
		}
	}
	return nil
}

func pathCandidates(path string) ([]string, error) {
	clean, err := CleanRequired(path)
	if err != nil {
		return nil, err
	}
	candidates := []string{clean}
	resolved, err := ResolveExistingOrAncestor(clean)
	if err != nil {
		return nil, err
	}
	if resolved != clean {
		candidates = append(candidates, resolved)
	}
	return candidates, nil
}

func ResolveExistingOrAncestor(path string) (string, error) {
	clean, err := CleanRequired(path)
	if err != nil {
		return "", err
	}
	current := clean
	missing := []string{}
	for {
		_, err := os.Lstat(current)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return clean, nil
		}
		missing = append([]string{filepath.Base(current)}, missing...)
		current = parent
	}
	resolved, err := filepath.EvalSymlinks(current)
	if err != nil {
		return "", err
	}
	parts := append([]string{resolved}, missing...)
	return filepath.Clean(filepath.Join(parts...)), nil
}

func sameOrNested(root string, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	if root == candidate {
		return true
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

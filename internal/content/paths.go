package content

import (
	"errors"
	"path"
	"strings"
)

var (
	ErrInvalidSlug      = errors.New("invalid slug")
	ErrInvalidAssetPath = errors.New("invalid asset path")
)

func ValidateSlug(slug string) error {
	if slug == "" {
		return ErrInvalidSlug
	}
	for _, r := range slug {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' {
			continue
		}
		return ErrInvalidSlug
	}
	return nil
}

func ValidateAssetPath(assetPath string) error {
	_, err := CleanAssetPath(assetPath)
	return err
}

func CleanAssetPath(assetPath string) (string, error) {
	if assetPath == "" || strings.HasPrefix(assetPath, "/") || strings.HasPrefix(assetPath, "\\") {
		return "", ErrInvalidAssetPath
	}

	normalized := strings.ReplaceAll(assetPath, "\\", "/")
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", ErrInvalidAssetPath
		}
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", ErrInvalidAssetPath
	}
	return cleaned, nil
}

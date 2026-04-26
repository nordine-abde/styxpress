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
	if assetPath == "" || strings.HasPrefix(assetPath, "/") || strings.HasPrefix(assetPath, "\\") {
		return ErrInvalidAssetPath
	}

	parts := strings.FieldsFunc(assetPath, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return ErrInvalidAssetPath
		}
	}

	cleaned := path.Clean(strings.ReplaceAll(assetPath, "\\", "/"))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return ErrInvalidAssetPath
	}
	return nil
}

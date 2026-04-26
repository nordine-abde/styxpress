package content

import "testing"

func TestValidateSlug(t *testing.T) {
	tests := map[string]bool{
		"hello-world": true,
		"go-http-2":   true,
		"":            false,
		"Hello":       false,
		"hello_world": false,
		"hello/world": false,
		"../secret":   false,
	}

	for slug, valid := range tests {
		err := ValidateSlug(slug)
		if valid && err != nil {
			t.Fatalf("ValidateSlug(%q) returned %v, want nil", slug, err)
		}
		if !valid && err == nil {
			t.Fatalf("ValidateSlug(%q) returned nil, want error", slug)
		}
	}
}

func TestValidateAssetPath(t *testing.T) {
	tests := map[string]bool{
		"image.jpg":          true,
		"gallery/image.jpg":  true,
		"":                   false,
		"/image.jpg":         false,
		"..":                 false,
		"../secret.txt":      false,
		"gallery/../x.jpg":   false,
		"gallery\\..\\x.jpg": false,
	}

	for assetPath, valid := range tests {
		err := ValidateAssetPath(assetPath)
		if valid && err != nil {
			t.Fatalf("ValidateAssetPath(%q) returned %v, want nil", assetPath, err)
		}
		if !valid && err == nil {
			t.Fatalf("ValidateAssetPath(%q) returned nil, want error", assetPath)
		}
	}
}

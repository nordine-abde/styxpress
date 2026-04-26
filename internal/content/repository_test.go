package content

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRepositoryCreateLoadUpdatePost(t *testing.T) {
	repo := NewRepository(t.TempDir())
	firstPublished := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	updated := time.Date(2026, 4, 2, 10, 45, 0, 0, time.UTC)

	created, err := repo.WritePost(Post{
		Slug:        "hello-world",
		Title:       " Hello World ",
		Description: " First post ",
		Source:      "# Hello\n",
	}, WritePostOptions{Now: firstPublished})
	if err != nil {
		t.Fatalf("WritePost create returned error: %v", err)
	}
	if !created.PublishedAt.Equal(firstPublished) {
		t.Fatalf("PublishedAt = %v, want %v", created.PublishedAt, firstPublished)
	}
	if !created.UpdatedAt.Equal(firstPublished) {
		t.Fatalf("UpdatedAt = %v, want %v", created.UpdatedAt, firstPublished)
	}
	if created.Title != "Hello World" || created.Description != "First post" {
		t.Fatalf("metadata = %#v, want trimmed title and description", created)
	}

	got, err := repo.WritePost(Post{
		Slug:        "hello-world",
		Title:       "Hello Again",
		Description: "",
		Source:      "# Changed\n",
	}, WritePostOptions{Now: updated})
	if err != nil {
		t.Fatalf("WritePost update returned error: %v", err)
	}
	if !got.PublishedAt.Equal(firstPublished) {
		t.Fatalf("PublishedAt changed to %v, want %v", got.PublishedAt, firstPublished)
	}
	if !got.UpdatedAt.Equal(updated) {
		t.Fatalf("UpdatedAt = %v, want %v", got.UpdatedAt, updated)
	}
	if got.Description != "" {
		t.Fatalf("Description = %q, want empty after removing description", got.Description)
	}

	loaded, err := repo.LoadPost("hello-world")
	if err != nil {
		t.Fatalf("LoadPost returned error: %v", err)
	}
	if loaded.Source != "# Changed\n" {
		t.Fatalf("Source = %q, want updated source", loaded.Source)
	}
}

func TestRepositoryCreatePostRejectsExistingDirectory(t *testing.T) {
	repo := NewRepository(t.TempDir())
	post := Post{Slug: "hello-world", Title: "Title", Source: "Body"}

	if _, err := repo.CreatePost(post); err != nil {
		t.Fatalf("CreatePost returned error: %v", err)
	}
	if _, err := repo.CreatePost(post); !errors.Is(err, ErrPostExists) {
		t.Fatalf("second CreatePost error = %v, want ErrPostExists", err)
	}
}

func TestRepositoryRejectsInvalidPostMetadata(t *testing.T) {
	repo := NewRepository(t.TempDir())

	tests := []Post{
		{Slug: "Hello", Title: "Title", Source: "Body"},
		{Slug: "hello", Title: " ", Source: "Body"},
		{Slug: "hello", Title: "Title", Source: "Body", Assets: []string{"../secret.txt"}},
		{Slug: "hello", Title: "Title", Source: "Body", Cover: "cover.gif"},
	}

	for _, post := range tests {
		if _, err := repo.WritePost(post, WritePostOptions{Now: time.Now()}); err == nil {
			t.Fatalf("WritePost(%#v) returned nil, want error", post)
		}
	}
}

func TestRepositoryRejectsInvalidStoredMetadata(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "posts", "hello-world")
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	files := map[string]string{
		"title.txt":        "Title\n",
		"description.txt":  "Description\n",
		"source.md":        "Body\n",
		"published_at.txt": "not-a-time\n",
		"updated_at.txt":   "2026-04-01T00:00:00Z\n",
	}
	for name, value := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), fileMode); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	repo := NewRepository(root)
	if _, err := repo.LoadPost("hello-world"); !errors.Is(err, ErrInvalidPost) {
		t.Fatalf("LoadPost error = %v, want ErrInvalidPost", err)
	}
}

func TestRepositoryCoverOperations(t *testing.T) {
	root := t.TempDir()
	repo := NewRepository(root)

	if err := repo.WriteCover("hello-world", "cover.jpg", strings.NewReader("jpg")); err != nil {
		t.Fatalf("WriteCover returned error: %v", err)
	}
	if err := repo.WriteCover("hello-world", "cover.webp", strings.NewReader("webp")); err != nil {
		t.Fatalf("WriteCover replacement returned error: %v", err)
	}

	dir := filepath.Join(root, "posts", "hello-world")
	if _, err := os.Stat(filepath.Join(dir, "cover.jpg")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old cover stat error = %v, want not exist", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "cover.webp"))
	if err != nil {
		t.Fatalf("ReadFile cover.webp returned error: %v", err)
	}
	if string(data) != "webp" {
		t.Fatalf("cover.webp = %q, want webp", data)
	}

	if err := os.WriteFile(filepath.Join(dir, "cover.png"), []byte("png"), fileMode); err != nil {
		t.Fatalf("WriteFile duplicate cover returned error: %v", err)
	}
	if _, err := findCover(dir); !errors.Is(err, ErrDuplicateCover) {
		t.Fatalf("findCover error = %v, want ErrDuplicateCover", err)
	}

	if err := repo.DeleteCover("hello-world"); err != nil {
		t.Fatalf("DeleteCover returned error: %v", err)
	}
	if cover, err := findCover(dir); err != nil || cover != "" {
		t.Fatalf("findCover after delete = %q, %v; want empty nil", cover, err)
	}
}

func TestRepositoryAssetOperations(t *testing.T) {
	repo := NewRepository(t.TempDir())

	if err := repo.WriteAsset("hello-world", "gallery/image.jpg", strings.NewReader("image")); err != nil {
		t.Fatalf("WriteAsset returned error: %v", err)
	}
	if err := repo.WriteAsset("hello-world", "../secret.txt", strings.NewReader("secret")); !errors.Is(err, ErrInvalidAssetPath) {
		t.Fatalf("WriteAsset traversal error = %v, want ErrInvalidAssetPath", err)
	}

	post := Post{Slug: "hello-world", Title: "Title", Source: "Body"}
	if _, err := repo.WritePost(post, WritePostOptions{Now: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("WritePost returned error: %v", err)
	}
	loaded, err := repo.LoadPost("hello-world")
	if err != nil {
		t.Fatalf("LoadPost returned error: %v", err)
	}
	if len(loaded.Assets) != 1 || loaded.Assets[0] != "gallery/image.jpg" {
		t.Fatalf("Assets = %#v, want gallery/image.jpg", loaded.Assets)
	}

	if err := repo.DeleteAsset("hello-world", "gallery/image.jpg"); err != nil {
		t.Fatalf("DeleteAsset returned error: %v", err)
	}
	loaded, err = repo.LoadPost("hello-world")
	if err != nil {
		t.Fatalf("LoadPost after delete returned error: %v", err)
	}
	if len(loaded.Assets) != 0 {
		t.Fatalf("Assets after delete = %#v, want empty", loaded.Assets)
	}
}

func TestRepositoryRejectsAssetSymlink(t *testing.T) {
	root := t.TempDir()
	repo := NewRepository(root)
	post := Post{Slug: "hello-world", Title: "Title", Source: "Body"}
	if _, err := repo.WritePost(post, WritePostOptions{Now: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("WritePost returned error: %v", err)
	}

	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("target"), fileMode); err != nil {
		t.Fatalf("WriteFile target returned error: %v", err)
	}
	link := filepath.Join(root, "posts", "hello-world", "assets", "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := repo.LoadPost("hello-world"); !errors.Is(err, ErrInvalidAsset) {
		t.Fatalf("LoadPost error = %v, want ErrInvalidAsset", err)
	}
	if err := repo.WriteAsset("hello-world", "link.txt", strings.NewReader("replace")); !errors.Is(err, ErrInvalidAsset) {
		t.Fatalf("WriteAsset symlink error = %v, want ErrInvalidAsset", err)
	}
}

func TestRepositoryFeaturedRoundTrip(t *testing.T) {
	repo := NewRepository(t.TempDir())

	if err := repo.WriteFeatured([]string{"hello-world", "go-http-2"}); err != nil {
		t.Fatalf("WriteFeatured returned error: %v", err)
	}
	got, err := repo.ReadFeatured()
	if err != nil {
		t.Fatalf("ReadFeatured returned error: %v", err)
	}
	want := []string{"hello-world", "go-http-2"}
	if len(got) != len(want) {
		t.Fatalf("ReadFeatured length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ReadFeatured[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if err := repo.WriteFeatured([]string{"Hello"}); !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("WriteFeatured invalid slug error = %v, want ErrInvalidSlug", err)
	}
}

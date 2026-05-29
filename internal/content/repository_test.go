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
	root := t.TempDir()
	repo := NewRepository(root)
	firstSaved := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	updated := time.Date(2026, 4, 2, 10, 45, 0, 0, time.UTC)
	published := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	synced := time.Date(2026, 4, 3, 11, 5, 0, 0, time.UTC)

	created, err := repo.WritePost(Post{
		Slug:        "hello-world",
		Title:       " Hello World ",
		Description: " First post ",
		Source:      "# Hello\n",
	}, WritePostOptions{Now: firstSaved})
	if err != nil {
		t.Fatalf("WritePost create returned error: %v", err)
	}
	if !created.PublishedAt.IsZero() {
		t.Fatalf("PublishedAt = %v, want zero for draft", created.PublishedAt)
	}
	if !created.UpdatedAt.Equal(firstSaved) {
		t.Fatalf("UpdatedAt = %v, want %v", created.UpdatedAt, firstSaved)
	}
	if created.PublishStatus() != PublishStatusDraft {
		t.Fatalf("PublishStatus = %q, want draft", created.PublishStatus())
	}
	if _, err := os.Stat(filepath.Join(root, "posts", "hello-world", "published_at.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published_at.txt stat error = %v, want not exist", err)
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
	if !got.PublishedAt.IsZero() {
		t.Fatalf("PublishedAt = %v, want zero after draft update", got.PublishedAt)
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

	marked, err := repo.MarkPostPublished("hello-world", TimestampOptions{Now: published})
	if err != nil {
		t.Fatalf("MarkPostPublished returned error: %v", err)
	}
	if !marked.PublishedAt.Equal(published) || !marked.UpdatedAt.Equal(updated) {
		t.Fatalf("marked post times = published %v updated %v, want %v and %v", marked.PublishedAt, marked.UpdatedAt, published, updated)
	}
	if marked.PublishStatus() != PublishStatusPendingPublish {
		t.Fatalf("PublishStatus after marking published = %q, want pending_publish", marked.PublishStatus())
	}

	syncedPost, err := repo.MarkPostSynced("hello-world", TimestampOptions{Now: synced})
	if err != nil {
		t.Fatalf("MarkPostSynced returned error: %v", err)
	}
	if !syncedPost.SyncedAt.Equal(synced) {
		t.Fatalf("SyncedAt = %v, want %v", syncedPost.SyncedAt, synced)
	}
	if syncedPost.PublishStatus() != PublishStatusPublished {
		t.Fatalf("PublishStatus after sync = %q, want published", syncedPost.PublishStatus())
	}

	clearedPost, err := repo.ClearPostSynced("hello-world")
	if err != nil {
		t.Fatalf("ClearPostSynced returned error: %v", err)
	}
	if !clearedPost.SyncedAt.IsZero() {
		t.Fatalf("SyncedAt after clear = %v, want zero", clearedPost.SyncedAt)
	}
	if clearedPost.PublishStatus() != PublishStatusPendingPublish {
		t.Fatalf("PublishStatus after clearing sync = %q, want pending_publish", clearedPost.PublishStatus())
	}
	if _, err := os.Stat(filepath.Join(root, "posts", "hello-world", "remote_synced_at.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remote_synced_at.txt stat error = %v, want not exist", err)
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

func TestRepositoryMediaChangesTouchUpdatedAt(t *testing.T) {
	root := t.TempDir()
	repo := NewRepository(root)
	published := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	synced := time.Date(2026, 4, 1, 9, 35, 0, 0, time.UTC)
	touched := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	if _, err := repo.WritePost(Post{
		Slug:        "hello-world",
		Title:       "Hello World",
		Source:      "Body",
		PublishedAt: published,
		UpdatedAt:   published,
		SyncedAt:    synced,
	}, WritePostOptions{}); err != nil {
		t.Fatalf("WritePost returned error: %v", err)
	}
	repo.now = func() time.Time { return touched }

	if err := repo.WriteCover("hello-world", "cover.jpg", strings.NewReader("jpg")); err != nil {
		t.Fatalf("WriteCover returned error: %v", err)
	}
	loaded, err := repo.LoadPost("hello-world")
	if err != nil {
		t.Fatalf("LoadPost returned error: %v", err)
	}
	if !loaded.UpdatedAt.Equal(touched) {
		t.Fatalf("UpdatedAt = %v, want touched time %v", loaded.UpdatedAt, touched)
	}
	if loaded.PublishStatus() != PublishStatusPendingPublish {
		t.Fatalf("PublishStatus = %q, want pending_publish after media change", loaded.PublishStatus())
	}

	nextTouch := touched.Add(time.Hour)
	repo.now = func() time.Time { return nextTouch }
	if err := repo.WriteAsset("hello-world", "diagram.txt", strings.NewReader("diagram")); err != nil {
		t.Fatalf("WriteAsset returned error: %v", err)
	}
	loaded, err = repo.LoadPost("hello-world")
	if err != nil {
		t.Fatalf("LoadPost after asset returned error: %v", err)
	}
	if !loaded.UpdatedAt.Equal(nextTouch) {
		t.Fatalf("UpdatedAt after asset = %v, want %v", loaded.UpdatedAt, nextTouch)
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

func TestRepositoryListPostsOrdersByPublishedAtThenSlug(t *testing.T) {
	repo := NewRepository(t.TempDir())
	older := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)
	for _, post := range []Post{
		{Slug: "z-post", Title: "Z", Source: "Z", PublishedAt: newer, UpdatedAt: newer},
		{Slug: "a-post", Title: "A", Source: "A", PublishedAt: newer, UpdatedAt: newer},
		{Slug: "old-post", Title: "Old", Source: "Old", PublishedAt: older, UpdatedAt: older},
	} {
		if _, err := repo.WritePost(post, WritePostOptions{}); err != nil {
			t.Fatalf("WritePost(%s) returned error: %v", post.Slug, err)
		}
	}

	posts, err := repo.ListPosts()
	if err != nil {
		t.Fatalf("ListPosts returned error: %v", err)
	}
	got := make([]string, len(posts))
	for i, post := range posts {
		got[i] = post.Slug
	}
	want := []string{"a-post", "z-post", "old-post"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListPosts order = %#v, want %#v", got, want)
		}
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
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for _, post := range []Post{
		{Slug: "hello-world", Title: "Hello World", Source: "Body"},
		{Slug: "go-http-2", Title: "Go HTTP 2", Source: "Body"},
	} {
		if _, err := repo.WritePost(post, WritePostOptions{Now: now}); err != nil {
			t.Fatalf("WritePost %q returned error: %v", post.Slug, err)
		}
	}

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
	if err := repo.WriteFeatured([]string{"missing-post"}); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("WriteFeatured missing post error = %v, want ErrPostNotFound", err)
	}
}

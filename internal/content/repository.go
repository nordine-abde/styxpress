package content

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	postsDirName        = "posts"
	assetsDirName       = "assets"
	sourceFileName      = "source.md"
	titleFileName       = "title.txt"
	descriptionFileName = "description.txt"
	publishedFileName   = "published_at.txt"
	updatedFileName     = "updated_at.txt"
	featuredFileName    = "featured.txt"
	directoryMode       = 0o755
	fileMode            = 0o644
)

var (
	ErrInvalidPost      = errors.New("invalid post")
	ErrPostNotFound     = errors.New("post not found")
	ErrPostExists       = errors.New("post already exists")
	ErrDuplicateCover   = errors.New("duplicate cover files")
	ErrUnsupportedCover = errors.New("unsupported cover file")
	ErrInvalidAsset     = errors.New("invalid asset")
)

type Repository struct {
	root string
	now  func() time.Time
}

type Post struct {
	Slug        string
	Title       string
	Description string
	Source      string
	PublishedAt time.Time
	UpdatedAt   time.Time
	Cover       string
	Assets      []string
}

type WritePostOptions struct {
	Now time.Time
}

func NewRepository(root string) *Repository {
	return &Repository{
		root: root,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (r *Repository) CreatePost(post Post) (Post, error) {
	if err := ValidateSlug(post.Slug); err != nil {
		return Post{}, err
	}
	if _, err := os.Stat(r.postDir(post.Slug)); err == nil {
		return Post{}, ErrPostExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return Post{}, err
	}
	return r.writePost(post, false, WritePostOptions{})
}

func (r *Repository) UpdatePost(post Post) (Post, error) {
	return r.writePost(post, true, WritePostOptions{})
}

func (r *Repository) WritePost(post Post, opts WritePostOptions) (Post, error) {
	if err := ValidateSlug(post.Slug); err != nil {
		return Post{}, err
	}

	_, err := os.Stat(filepath.Join(r.postDir(post.Slug), publishedFileName))
	update := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Post{}, err
	}
	return r.writePost(post, update, opts)
}

func (r *Repository) LoadPost(slug string) (Post, error) {
	if err := ValidateSlug(slug); err != nil {
		return Post{}, err
	}

	dir := r.postDir(slug)
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Post{}, ErrPostNotFound
		}
		return Post{}, err
	}
	if !info.IsDir() {
		return Post{}, fmt.Errorf("%w: post path is not a directory", ErrInvalidPost)
	}

	cover, err := findCover(dir)
	if err != nil {
		return Post{}, err
	}
	assets, err := listAssets(filepath.Join(dir, assetsDirName))
	if err != nil {
		return Post{}, err
	}

	title, err := readRequiredTextFile(filepath.Join(dir, titleFileName))
	if err != nil {
		return Post{}, err
	}
	description, err := readOptionalTextFile(filepath.Join(dir, descriptionFileName))
	if err != nil {
		return Post{}, err
	}
	source, err := readRequiredTextFile(filepath.Join(dir, sourceFileName))
	if err != nil {
		return Post{}, err
	}
	publishedAt, err := readRequiredTimeFile(filepath.Join(dir, publishedFileName))
	if err != nil {
		return Post{}, err
	}
	updatedAt, err := readRequiredTimeFile(filepath.Join(dir, updatedFileName))
	if err != nil {
		return Post{}, err
	}

	post := Post{
		Slug:        slug,
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		Source:      source,
		PublishedAt: publishedAt,
		UpdatedAt:   updatedAt,
		Cover:       cover,
		Assets:      assets,
	}
	if err := validatePost(post); err != nil {
		return Post{}, err
	}
	return post, nil
}

func (r *Repository) ListPosts() ([]Post, error) {
	root := filepath.Join(r.root, postsDirName)
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var posts []Post
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		if err := ValidateSlug(slug); err != nil {
			return nil, fmt.Errorf("%w: stored post directory %q", err, slug)
		}
		post, err := r.LoadPost(slug)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	sortPosts(posts)
	return posts, nil
}

func (r *Repository) ReadFeatured() ([]string, error) {
	path := filepath.Join(r.root, featuredFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var slugs []string
	for _, line := range strings.Split(string(data), "\n") {
		slug := strings.TrimSpace(line)
		if slug == "" {
			continue
		}
		if err := ValidateSlug(slug); err != nil {
			return nil, fmt.Errorf("%w: featured slug %q", err, slug)
		}
		slugs = append(slugs, slug)
	}
	return slugs, nil
}

func (r *Repository) WriteFeatured(slugs []string) error {
	var builder strings.Builder
	for _, slug := range slugs {
		if err := ValidateSlug(slug); err != nil {
			return fmt.Errorf("%w: featured slug %q", err, slug)
		}
		if _, err := r.LoadPost(slug); err != nil {
			if errors.Is(err, ErrPostNotFound) {
				return fmt.Errorf("%w: featured slug %q", ErrPostNotFound, slug)
			}
			return err
		}
		builder.WriteString(slug)
		builder.WriteByte('\n')
	}

	if err := os.MkdirAll(r.root, directoryMode); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.root, featuredFileName), []byte(builder.String()), fileMode)
}

func (r *Repository) WriteCover(slug string, name string, reader io.Reader) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	if !isCoverFile(name) {
		return ErrUnsupportedCover
	}

	dir := r.postDir(slug)
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !isCoverFile(entry.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return writeReader(filepath.Join(dir, name), reader)
}

func (r *Repository) DeleteCover(slug string) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}

	dir := r.postDir(slug)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !isCoverFile(entry.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) WriteAsset(slug string, assetPath string, reader io.Reader) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	cleaned, err := CleanAssetPath(assetPath)
	if err != nil {
		return err
	}
	return writeReader(filepath.Join(r.postDir(slug), assetsDirName, filepath.FromSlash(cleaned)), reader)
}

func (r *Repository) DeleteAsset(slug string, assetPath string) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	cleaned, err := CleanAssetPath(assetPath)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(r.postDir(slug), assetsDirName, filepath.FromSlash(cleaned))); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (r *Repository) writePost(post Post, update bool, opts WritePostOptions) (Post, error) {
	if err := validatePost(post); err != nil {
		return Post{}, err
	}

	now := opts.Now
	if now.IsZero() {
		now = r.now()
	}
	now = now.UTC().Truncate(0)

	dir := r.postDir(post.Slug)
	if update {
		existing, err := r.LoadPost(post.Slug)
		if err != nil {
			return Post{}, err
		}
		post.PublishedAt = existing.PublishedAt
		post.UpdatedAt = now
	} else {
		if post.PublishedAt.IsZero() {
			post.PublishedAt = now
		}
		if post.UpdatedAt.IsZero() {
			post.UpdatedAt = post.PublishedAt
		}
	}

	if err := os.MkdirAll(filepath.Join(dir, assetsDirName), directoryMode); err != nil {
		return Post{}, err
	}

	files := map[string]string{
		sourceFileName:    post.Source,
		titleFileName:     strings.TrimSpace(post.Title) + "\n",
		publishedFileName: post.PublishedAt.UTC().Format(time.RFC3339) + "\n",
		updatedFileName:   post.UpdatedAt.UTC().Format(time.RFC3339) + "\n",
	}
	if strings.TrimSpace(post.Description) != "" {
		files[descriptionFileName] = strings.TrimSpace(post.Description) + "\n"
	} else {
		if err := os.Remove(filepath.Join(dir, descriptionFileName)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Post{}, err
		}
	}

	for name, value := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), fileMode); err != nil {
			return Post{}, err
		}
	}
	return r.LoadPost(post.Slug)
}

func (r *Repository) postDir(slug string) string {
	return filepath.Join(r.root, postsDirName, slug)
}

func validatePost(post Post) error {
	if err := ValidateSlug(post.Slug); err != nil {
		return err
	}
	if strings.TrimSpace(post.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidPost)
	}
	for _, asset := range post.Assets {
		if err := ValidateAssetPath(asset); err != nil {
			return err
		}
	}
	if post.Cover != "" && !isCoverFile(post.Cover) {
		return ErrUnsupportedCover
	}
	return nil
}

func sortPosts(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		if !posts[i].PublishedAt.Equal(posts[j].PublishedAt) {
			return posts[i].PublishedAt.After(posts[j].PublishedAt)
		}
		return posts[i].Slug < posts[j].Slug
	})
}

func findCover(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var covers []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isCoverFile(name) {
			covers = append(covers, name)
		}
	}
	sort.Strings(covers)
	if len(covers) > 1 {
		return "", fmt.Errorf("%w: %s", ErrDuplicateCover, strings.Join(covers, ", "))
	}
	if len(covers) == 0 {
		return "", nil
	}
	return covers[0], nil
}

func isCoverFile(name string) bool {
	switch strings.ToLower(name) {
	case "cover.jpg", "cover.jpeg", "cover.png", "cover.webp", "cover.avif":
		return true
	default:
		return false
	}
}

func listAssets(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: assets path is not a directory", ErrInvalidPost)
	}

	var assets []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", ErrInvalidAsset, filepath.ToSlash(path))
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		assetPath := filepath.ToSlash(rel)
		if err := ValidateAssetPath(assetPath); err != nil {
			return err
		}
		assets = append(assets, assetPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(assets)
	return assets, nil
}

func readRequiredTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: missing %s", ErrInvalidPost, filepath.Base(path))
		}
		return "", err
	}
	return string(data), nil
}

func readOptionalTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func readRequiredTimeFile(path string) (time.Time, error) {
	value, err := readRequiredTextFile(path)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s must be RFC3339", ErrInvalidPost, filepath.Base(path))
	}
	return parsed, nil
}

func writeReader(path string, reader io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: symlink %s", ErrInvalidAsset, path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return err
	}
	return file.Chmod(fileMode)
}

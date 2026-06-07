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

type TimestampOptions struct {
	Now time.Time
}

type PublishStatus string

const (
	PublishStatusDraft     PublishStatus = "draft"
	PublishStatusPublished PublishStatus = "published"
)

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
	if err := requireDirectoryNoSymlink(r.root, r.postDir(post.Slug), ErrInvalidPost); err == nil {
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

	update, err := r.hasPostMetadata(post.Slug)
	if err != nil {
		return Post{}, err
	}
	return r.writePost(post, update, opts)
}

func (r *Repository) LoadPost(slug string) (Post, error) {
	if err := ValidateSlug(slug); err != nil {
		return Post{}, err
	}

	dir := r.postDir(slug)
	if err := requireDirectoryNoSymlink(r.root, dir, ErrInvalidPost); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Post{}, ErrPostNotFound
		}
		return Post{}, err
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
	publishedAt, err := readOptionalTimeFile(filepath.Join(dir, publishedFileName))
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
	entries, err := readDirectoryNoSymlink(r.root, root, ErrInvalidPost)
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

func (r *Repository) ListPublishedPosts() ([]Post, error) {
	posts, err := r.ListPosts()
	if err != nil {
		return nil, err
	}
	published := make([]Post, 0, len(posts))
	for _, post := range posts {
		if post.IsPublished() {
			published = append(published, post)
		}
	}
	return published, nil
}

func (r *Repository) MarkPostPublished(slug string, opts TimestampOptions) (Post, error) {
	if err := ValidateSlug(slug); err != nil {
		return Post{}, err
	}
	post, err := r.LoadPost(slug)
	if err != nil {
		return Post{}, err
	}
	if post.IsPublished() {
		return post, nil
	}

	now := r.timestamp(opts.Now)
	if err := writeFileNoSymlink(r.root, filepath.Join(r.postDir(slug), publishedFileName), []byte(formatTime(now)), ErrInvalidPost); err != nil {
		return Post{}, err
	}
	return r.LoadPost(slug)
}

func (r *Repository) WriteCover(slug string, name string, reader io.Reader) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	if !isCoverFile(name) {
		return ErrUnsupportedCover
	}

	dir := r.postDir(slug)
	if err := mkdirAllNoSymlink(r.root, dir, directoryMode, ErrInvalidPost); err != nil {
		return err
	}
	entries, err := readDirectoryNoSymlink(r.root, dir, ErrInvalidPost)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !isCoverFile(entry.Name()) {
			continue
		}
		if err := removeFileNoSymlink(r.root, filepath.Join(dir, entry.Name()), ErrInvalidPost); err != nil {
			return err
		}
	}
	if err := writeReader(r.root, filepath.Join(dir, name), reader, ErrInvalidPost); err != nil {
		return err
	}
	return r.touchPostUpdated(slug)
}

func (r *Repository) DeleteCover(slug string) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}

	dir := r.postDir(slug)
	entries, err := readDirectoryNoSymlink(r.root, dir, ErrInvalidPost)
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
		if err := removeFileNoSymlink(r.root, filepath.Join(dir, entry.Name()), ErrInvalidPost); err != nil {
			return err
		}
	}
	return r.touchPostUpdated(slug)
}

func (r *Repository) WriteAsset(slug string, assetPath string, reader io.Reader) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	cleaned, err := CleanAssetPath(assetPath)
	if err != nil {
		return err
	}
	if err := writeReader(r.root, filepath.Join(r.postDir(slug), assetsDirName, filepath.FromSlash(cleaned)), reader, ErrInvalidAsset); err != nil {
		return err
	}
	return r.touchPostUpdated(slug)
}

func (r *Repository) DeleteAsset(slug string, assetPath string) error {
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	cleaned, err := CleanAssetPath(assetPath)
	if err != nil {
		return err
	}
	if err := removeFileNoSymlink(r.root, filepath.Join(r.postDir(slug), assetsDirName, filepath.FromSlash(cleaned)), ErrInvalidAsset); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return r.touchPostUpdated(slug)
}

func (r *Repository) writePost(post Post, update bool, opts WritePostOptions) (Post, error) {
	if err := validatePost(post); err != nil {
		return Post{}, err
	}

	now := r.timestamp(opts.Now)

	dir := r.postDir(post.Slug)
	if update {
		existing, err := r.LoadPost(post.Slug)
		if err != nil {
			return Post{}, err
		}
		post.PublishedAt = existing.PublishedAt
		post.UpdatedAt = now
	} else {
		if post.UpdatedAt.IsZero() {
			if post.PublishedAt.IsZero() {
				post.UpdatedAt = now
			} else {
				post.UpdatedAt = post.PublishedAt
			}
		}
	}

	if err := mkdirAllNoSymlink(r.root, filepath.Join(dir, assetsDirName), directoryMode, ErrInvalidPost); err != nil {
		return Post{}, err
	}

	files := map[string]string{
		sourceFileName:  post.Source,
		titleFileName:   strings.TrimSpace(post.Title) + "\n",
		updatedFileName: formatTime(post.UpdatedAt),
	}
	if post.IsPublished() {
		files[publishedFileName] = formatTime(post.PublishedAt)
	} else if err := removeFileNoSymlink(r.root, filepath.Join(dir, publishedFileName), ErrInvalidPost); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Post{}, err
	}
	if strings.TrimSpace(post.Description) != "" {
		files[descriptionFileName] = strings.TrimSpace(post.Description) + "\n"
	} else {
		if err := removeFileNoSymlink(r.root, filepath.Join(dir, descriptionFileName), ErrInvalidPost); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Post{}, err
		}
	}

	for name, value := range files {
		if err := writeFileNoSymlink(r.root, filepath.Join(dir, name), []byte(value), ErrInvalidPost); err != nil {
			return Post{}, err
		}
	}
	return r.LoadPost(post.Slug)
}

func (r *Repository) hasPostMetadata(slug string) (bool, error) {
	dir := r.postDir(slug)
	if err := requireDirectoryNoSymlink(r.root, dir, ErrInvalidPost); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	for _, name := range []string{sourceFileName, titleFileName, descriptionFileName, publishedFileName, updatedFileName} {
		_, err := os.Stat(filepath.Join(dir, name))
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	return false, nil
}

func (r *Repository) timestamp(value time.Time) time.Time {
	if value.IsZero() {
		value = r.now()
	}
	return value.UTC()
}

func (r *Repository) touchPostUpdated(slug string) error {
	hasMetadata, err := r.hasPostMetadata(slug)
	if err != nil || !hasMetadata {
		return err
	}
	return writeFileNoSymlink(r.root, filepath.Join(r.postDir(slug), updatedFileName), []byte(formatTime(r.timestamp(time.Time{}))), ErrInvalidPost)
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

func (p Post) IsPublished() bool {
	return !p.PublishedAt.IsZero()
}

func (p Post) PublishStatus() PublishStatus {
	if !p.IsPublished() {
		return PublishStatusDraft
	}
	return PublishStatusPublished
}

func sortPosts(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		iPublished := posts[i].IsPublished()
		jPublished := posts[j].IsPublished()
		if iPublished != jPublished {
			return iPublished
		}
		if iPublished && !posts[i].PublishedAt.Equal(posts[j].PublishedAt) {
			return posts[i].PublishedAt.After(posts[j].PublishedAt)
		}
		if !iPublished && !posts[i].UpdatedAt.Equal(posts[j].UpdatedAt) {
			return posts[i].UpdatedAt.After(posts[j].UpdatedAt)
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
	info, err := os.Lstat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: symlink %s", ErrInvalidAsset, root)
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
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s must be RFC3339", ErrInvalidPost, filepath.Base(path))
	}
	return parsed, nil
}

func readOptionalTimeFile(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s must be RFC3339", ErrInvalidPost, filepath.Base(path))
	}
	return parsed, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano) + "\n"
}

func writeReader(root string, path string, reader io.Reader, errKind error) error {
	if err := mkdirAllNoSymlink(root, filepath.Dir(path), directoryMode, errKind); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: symlink %s", errKind, path)
	} else if err == nil && info.IsDir() {
		return fmt.Errorf("%w: path is a directory %s", errKind, path)
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

func readDirectoryNoSymlink(root string, dir string, errKind error) ([]os.DirEntry, error) {
	if err := requireDirectoryNoSymlink(root, dir, errKind); err != nil {
		return nil, err
	}
	return os.ReadDir(dir)
}

func writeFileNoSymlink(root string, path string, data []byte, errKind error) error {
	if err := mkdirAllNoSymlink(root, filepath.Dir(path), directoryMode, errKind); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: symlink %s", errKind, path)
	} else if err == nil && info.IsDir() {
		return fmt.Errorf("%w: path is a directory %s", errKind, path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, data, fileMode)
}

func removeFileNoSymlink(root string, path string, errKind error) error {
	if err := requireDirectoryNoSymlink(root, filepath.Dir(path), errKind); err != nil {
		return err
	}
	return os.Remove(path)
}

func mkdirAllNoSymlink(root string, dir string, mode fs.FileMode, errKind error) error {
	cleanRoot, rel, err := cleanPathUnderRoot(root, dir, errKind)
	if err != nil {
		return err
	}
	if err := ensureRootDirectory(cleanRoot, mode, errKind); err != nil {
		return err
	}

	current := cleanRoot
	for _, part := range relativeParts(rel) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: symlink %s", errKind, current)
			}
			if !info.IsDir() {
				return fmt.Errorf("%w: path is not a directory %s", errKind, current)
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Mkdir(current, mode); err != nil {
			return err
		}
	}
	return nil
}

func requireDirectoryNoSymlink(root string, dir string, errKind error) error {
	cleanRoot, rel, err := cleanPathUnderRoot(root, dir, errKind)
	if err != nil {
		return err
	}
	if err := requireRootDirectory(cleanRoot, errKind); err != nil {
		return err
	}

	current := cleanRoot
	for _, part := range relativeParts(rel) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", errKind, current)
		}
		if !info.IsDir() {
			return fmt.Errorf("%w: path is not a directory %s", errKind, current)
		}
	}
	return nil
}

func ensureRootDirectory(root string, mode fs.FileMode, errKind error) error {
	if err := os.MkdirAll(root, mode); err != nil {
		return err
	}
	return requireRootDirectory(root, errKind)
}

func requireRootDirectory(root string, errKind error) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: root is not a directory %s", errKind, root)
	}
	return nil
}

func cleanPathUnderRoot(root string, path string, errKind error) (string, string, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	cleanRoot = filepath.Clean(cleanRoot)
	cleanPath = filepath.Clean(cleanPath)

	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("%w: path escapes root %s", errKind, cleanPath)
	}
	return cleanRoot, rel, nil
}

func relativeParts(rel string) []string {
	if rel == "." || rel == "" {
		return nil
	}
	return strings.Split(rel, string(filepath.Separator))
}

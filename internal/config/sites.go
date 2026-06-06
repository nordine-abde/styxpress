package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	sitesDirName       = "sites"
	activeSiteFileName = "active-site.txt"
	defaultSiteID      = "default"
)

var (
	ErrInvalidSiteID = errors.New("invalid site id")
	ErrSiteNotFound  = errors.New("site not found")
	ErrNoActiveSite  = errors.New("no active site selected")
)

type Site struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Config Config `json:"config"`
}

type SiteStore struct {
	root string
}

func DefaultSiteStore() (*SiteStore, error) {
	root, err := DefaultDir()
	if err != nil {
		return nil, err
	}
	return NewSiteStore(root), nil
}

func NewSiteStore(root string) *SiteStore {
	return &SiteStore{root: filepath.Clean(root)}
}

func (s *SiteStore) List() ([]Site, string, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, "", err
	}
	sites, err := s.loadSites()
	if err != nil {
		return nil, "", err
	}
	if len(sites) == 0 {
		return sites, "", nil
	}
	activeID, err := s.activeSiteID()
	if err != nil && !errors.Is(err, ErrInvalidSiteID) && !errors.Is(err, os.ErrNotExist) {
		return nil, "", err
	}
	if err != nil || !containsSite(sites, activeID) {
		activeID = sites[0].ID
		if err := s.writeActiveSiteID(activeID); err != nil {
			return nil, "", err
		}
	}
	return sites, activeID, nil
}

func (s *SiteStore) Active() (Site, error) {
	sites, activeID, err := s.List()
	if err != nil {
		return Site{}, err
	}
	if activeID == "" {
		return Site{}, ErrNoActiveSite
	}
	for _, site := range sites {
		if site.ID == activeID {
			return site, nil
		}
	}
	return Site{}, fmt.Errorf("%w: %q", ErrSiteNotFound, activeID)
}

func (s *SiteStore) Create(cfg Config) (Site, error) {
	if err := s.ensureInitialized(); err != nil {
		return Site{}, err
	}
	cfg = WithDefaults(cfg)
	if cfg.Name == "" {
		cfg.Name = "Untitled site"
	}
	id, err := s.nextSiteID(cfg.Name)
	if err != nil {
		return Site{}, err
	}
	if err := s.saveSite(id, cfg); err != nil {
		return Site{}, err
	}
	if err := s.writeActiveSiteID(id); err != nil {
		return Site{}, err
	}
	return s.site(id)
}

func (s *SiteStore) Select(id string) (Site, error) {
	id, err := normalizeSiteID(id)
	if err != nil {
		return Site{}, err
	}
	site, err := s.site(id)
	if err != nil {
		return Site{}, err
	}
	if err := s.writeActiveSiteID(id); err != nil {
		return Site{}, err
	}
	return site, nil
}

func (s *SiteStore) SaveActive(cfg Config) (Site, error) {
	active, err := s.Active()
	if err != nil {
		return Site{}, err
	}
	if cfg.Name == "" {
		cfg.Name = active.Name
	}
	if err := s.saveSite(active.ID, cfg); err != nil {
		return Site{}, err
	}
	return s.site(active.ID)
}

func (s *SiteStore) Delete(id string) (string, error) {
	id, err := normalizeSiteID(id)
	if err != nil {
		return "", err
	}
	sites, activeID, err := s.List()
	if err != nil {
		return "", err
	}
	if !containsSite(sites, id) {
		return "", fmt.Errorf("%w: %q", ErrSiteNotFound, id)
	}
	if err := os.Remove(s.sitePath(id)); err != nil {
		return "", err
	}
	if activeID != id {
		return activeID, nil
	}
	remaining, err := s.loadSites()
	if err != nil {
		return "", err
	}
	if len(remaining) == 0 {
		if err := s.writeNoActiveSiteID(); err != nil {
			return "", err
		}
		return "", nil
	}
	nextID := remaining[0].ID
	if err := s.writeActiveSiteID(nextID); err != nil {
		return "", err
	}
	return nextID, nil
}

func (s *SiteStore) ensureInitialized() error {
	if err := os.MkdirAll(s.sitesDir(), directoryPermission); err != nil {
		return err
	}
	sites, err := s.loadSites()
	if err != nil {
		return err
	}
	if len(sites) == 0 {
		if _, err := os.Stat(s.activePath()); err == nil {
			return nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		migrated, err := s.migrateLegacyConfig()
		if err != nil {
			return err
		}
		if migrated {
			return s.writeActiveSiteID(defaultSiteID)
		}
		return s.writeNoActiveSiteID()
	}
	if _, err := os.Stat(s.activePath()); errors.Is(err, os.ErrNotExist) {
		return s.writeActiveSiteID(sites[0].ID)
	} else if err != nil {
		return err
	}
	return nil
}

func (s *SiteStore) migrateLegacyConfig() (bool, error) {
	legacyPath := filepath.Join(s.root, configFileName)
	cfg, err := Load(legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if cfg.Name == "" {
		cfg.Name = siteName(defaultSiteID, cfg)
	}
	if err := s.saveSite(defaultSiteID, cfg); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SiteStore) loadSites() ([]Site, error) {
	entries, err := os.ReadDir(s.sitesDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sites := make([]Site, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".toml")
		if _, err := normalizeSiteID(id); err != nil {
			continue
		}
		site, err := s.site(id)
		if err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	sort.Slice(sites, func(i, j int) bool {
		left := strings.ToLower(sites[i].Name)
		right := strings.ToLower(sites[j].Name)
		if left == right {
			return sites[i].ID < sites[j].ID
		}
		return left < right
	})
	return sites, nil
}

func (s *SiteStore) site(id string) (Site, error) {
	id, err := normalizeSiteID(id)
	if err != nil {
		return Site{}, err
	}
	cfg, err := Load(s.sitePath(id))
	if errors.Is(err, os.ErrNotExist) {
		return Site{}, fmt.Errorf("%w: %q", ErrSiteNotFound, id)
	}
	if err != nil {
		return Site{}, err
	}
	name := siteName(id, cfg)
	cfg.Name = name
	return Site{ID: id, Name: name, Config: cfg}, nil
}

func (s *SiteStore) saveSite(id string, cfg Config) error {
	id, err := normalizeSiteID(id)
	if err != nil {
		return err
	}
	cfg = WithDefaults(cfg)
	if cfg.Name == "" {
		cfg.Name = siteName(id, cfg)
	}
	return Save(s.sitePath(id), cfg)
}

func (s *SiteStore) nextSiteID(name string) (string, error) {
	base := slugifySiteID(name)
	if base == "" {
		base = "site"
	}
	for i := 1; ; i++ {
		id := base
		if i > 1 {
			id = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := os.Stat(s.sitePath(id)); errors.Is(err, os.ErrNotExist) {
			return id, nil
		} else if err != nil {
			return "", err
		}
	}
}

func (s *SiteStore) activeSiteID() (string, error) {
	data, err := os.ReadFile(s.activePath())
	if err != nil {
		return "", err
	}
	return normalizeSiteID(strings.TrimSpace(string(data)))
}

func (s *SiteStore) writeActiveSiteID(id string) error {
	id, err := normalizeSiteID(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, directoryPermission); err != nil {
		return err
	}
	return os.WriteFile(s.activePath(), []byte(id+"\n"), filePermission)
}

func (s *SiteStore) writeNoActiveSiteID() error {
	if err := os.MkdirAll(s.root, directoryPermission); err != nil {
		return err
	}
	return os.WriteFile(s.activePath(), []byte("\n"), filePermission)
}

func (s *SiteStore) sitePath(id string) string {
	return filepath.Join(s.sitesDir(), id+".toml")
}

func (s *SiteStore) activePath() string {
	return filepath.Join(s.root, activeSiteFileName)
}

func (s *SiteStore) sitesDir() string {
	return filepath.Join(s.root, sitesDirName)
}

func normalizeSiteID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" || id == "." || id == ".." || strings.Contains(id, "\x00") {
		return "", ErrInvalidSiteID
	}
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("%w: %q", ErrInvalidSiteID, id)
	}
	return id, nil
}

func slugifySiteID(name string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' {
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func siteName(id string, cfg Config) string {
	if strings.TrimSpace(cfg.Name) != "" {
		return strings.TrimSpace(cfg.Name)
	}
	if strings.TrimSpace(cfg.ContentDir) != "" {
		return filepath.Base(filepath.Clean(cfg.ContentDir))
	}
	return id
}

func containsSite(sites []Site, id string) bool {
	for _, site := range sites {
		if site.ID == id {
			return true
		}
	}
	return false
}

package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"portside/core/porthome"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type Prefix struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

type Profile struct {
	Schema    string            `json:"schema"`
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	AppID     int               `json:"appid"`
	Prefix    string            `json:"prefix"`
	Launcher  string            `json:"launcher"`
	Run       RunConfig         `json:"run"`
	Graphics  GraphicsConfig    `json:"graphics"`
	Steam     SteamConfig       `json:"steam"`
	Process   ProcessConfig     `json:"process"`
	Env       map[string]string `json:"env"`
	Notes     []string          `json:"notes"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

type RunConfig struct {
	EXE  string   `json:"exe"`
	Args []string `json:"args"`
	CWD  string   `json:"cwd,omitempty"`
}

type GraphicsConfig struct {
	Backend        string `json:"backend"`
	MetalHUD       bool   `json:"metal_hud"`
	ResolutionHint string `json:"resolution_hint,omitempty"`
}

type SteamConfig struct {
	AppManifest string `json:"appmanifest,omitempty"`
	LibraryHint string `json:"library_hint,omitempty"`
}

type ProcessConfig struct {
	WaitFor        []string `json:"wait_for,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type AddOptions struct {
	ID     string
	Name   string
	AppID  int
	Prefix string
}

func CreatePrefix(id string) (Prefix, error) {
	if err := validateID(id); err != nil {
		return Prefix{}, err
	}

	home, err := porthome.Resolve()
	if err != nil {
		return Prefix{}, err
	}

	path := filepath.Join(porthome.PrefixesDir(home.Path), id)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return Prefix{}, err
	}

	return Prefix{
		ID:        id,
		Path:      path,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func ListPrefixes() ([]Prefix, error) {
	home, err := porthome.Resolve()
	if err != nil {
		return nil, err
	}

	dir := porthome.PrefixesDir(home.Path)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Prefix{}, nil
	}
	if err != nil {
		return nil, err
	}

	prefixes := []Prefix{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, _ := entry.Info()
		created := ""
		if info != nil {
			created = info.ModTime().UTC().Format(time.RFC3339)
		}
		prefixes = append(prefixes, Prefix{
			ID:        entry.Name(),
			Path:      filepath.Join(dir, entry.Name()),
			CreatedAt: created,
		})
	}

	sort.Slice(prefixes, func(i, j int) bool {
		return prefixes[i].ID < prefixes[j].ID
	})

	return prefixes, nil
}

func AddProfile(options AddOptions) (Profile, error) {
	if err := validateID(options.ID); err != nil {
		return Profile{}, err
	}
	if options.AppID <= 0 {
		return Profile{}, fmt.Errorf("appid must be a positive integer")
	}
	if options.Prefix == "" {
		return Profile{}, fmt.Errorf("prefix is required")
	}
	if options.Name == "" {
		options.Name = options.ID
	}

	home, err := porthome.Resolve()
	if err != nil {
		return Profile{}, err
	}

	if _, err := os.Stat(filepath.Join(porthome.PrefixesDir(home.Path), options.Prefix)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Profile{}, fmt.Errorf("prefix %s does not exist", options.Prefix)
		}
		return Profile{}, err
	}

	path := profilePath(home.Path, options.ID)
	if _, err := os.Stat(path); err == nil {
		return Profile{}, fmt.Errorf("profile %s already exists", options.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Profile{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	profile := Profile{
		Schema:   "portside.profile/v1",
		ID:       options.ID,
		Name:     options.Name,
		AppID:    options.AppID,
		Prefix:   options.Prefix,
		Launcher: "steam",
		Run: RunConfig{
			EXE:  "C:/Program Files (x86)/Steam/steam.exe",
			Args: []string{"-applaunch", strconv.Itoa(options.AppID)},
			CWD:  "C:/Program Files (x86)/Steam",
		},
		Graphics: GraphicsConfig{
			Backend:  "d3dmetal",
			MetalHUD: false,
		},
		Steam: SteamConfig{
			AppManifest: fmt.Sprintf("steamapps/appmanifest_%d.acf", options.AppID),
			LibraryHint: "C:/Program Files (x86)/Steam/steamapps",
		},
		Process: ProcessConfig{TimeoutSeconds: 60},
		Env: map[string]string{
			"STEAM_COMPAT_LOG": "0",
		},
		Notes:     []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := porthome.WriteJSONAtomic(path, profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func ListProfiles() ([]Profile, error) {
	home, err := porthome.Resolve()
	if err != nil {
		return nil, err
	}

	dir := porthome.ProfilesDir(home.Path)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Profile{}, nil
	}
	if err != nil {
		return nil, err
	}

	result := []Profile{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		profile, err := ReadProfile(entry.Name()[:len(entry.Name())-len(".json")])
		if err != nil {
			return nil, err
		}
		result = append(result, profile)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

func ReadProfile(id string) (Profile, error) {
	if err := validateID(id); err != nil {
		return Profile{}, err
	}

	home, err := porthome.Resolve()
	if err != nil {
		return Profile{}, err
	}

	data, err := os.ReadFile(profilePath(home.Path, id))
	if err != nil {
		return Profile{}, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("parse profile %s: %w", id, err)
	}

	return profile, nil
}

func profilePath(home, id string) string {
	return filepath.Join(porthome.ProfilesDir(home), id+".json")
}

func validateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid id %q: use lowercase letters, digits, and hyphen", id)
	}
	return nil
}

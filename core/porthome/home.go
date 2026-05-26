package porthome

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	DefaultRelativeHome = "Games/portside"
	ConfigFileName      = "config.json"
)

var StateDirs = []string{
	"prefixes",
	"profiles",
	"runners",
	"installers",
	"logs",
	"snapshots",
	"apps",
	"state",
	"state/tasks",
	"state/locks",
	"cache",
	"cache/covers",
	"cache/steam-metadata",
	"cache/updates",
}

type Home struct {
	Path string `json:"path"`
}

type Config struct {
	Schema    string       `json:"schema"`
	Home      string       `json:"home"`
	CreatedAt string       `json:"created_at"`
	Runner    RunnerConfig `json:"runner"`
	GPTK      GPTKConfig   `json:"gptk"`
	Update    UpdateConfig `json:"update"`
}

type RunnerConfig struct {
	Default string                 `json:"default"`
	Runners map[string]RunnerEntry `json:"runners"`
}

type RunnerEntry struct {
	Command       string `json:"command"`
	NoHUDCommand  string `json:"no_hud_command,omitempty"`
	ServerCommand string `json:"server_command,omitempty"`
	PrefixMode    string `json:"prefix_mode"`
	EXEPathStyle  string `json:"exe_path_style"`
	Kind          string `json:"kind,omitempty"`
	Source        string `json:"source,omitempty"`
	Version       string `json:"version,omitempty"`
}

type GPTKConfig struct {
	RuntimePackage GPTKPackage `json:"runtime_package"`
}

type GPTKPackage struct {
	Provider string `json:"provider,omitempty"`
	File     string `json:"file,omitempty"`
	Version  string `json:"version,omitempty"`
}

type UpdateConfig struct {
	Channel   string           `json:"channel"`
	Source    string           `json:"source"`
	Endpoints []UpdateEndpoint `json:"endpoints"`
}

type UpdateEndpoint struct {
	Name     string `json:"name"`
	Manifest string `json:"manifest"`
}

type InitResult struct {
	Home       string   `json:"home"`
	ConfigPath string   `json:"config_path"`
	Created    []string `json:"created"`
	Existing   []string `json:"existing"`
}

func Resolve() (Home, error) {
	if value := os.Getenv("PORTSIDE_HOME"); value != "" {
		path, err := expandPath(value)
		if err != nil {
			return Home{}, err
		}
		return Home{Path: filepath.Clean(path)}, nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return Home{}, fmt.Errorf("resolve user home: %w", err)
	}

	return Home{Path: filepath.Join(userHome, DefaultRelativeHome)}, nil
}

func Init() (InitResult, error) {
	home, err := Resolve()
	if err != nil {
		return InitResult{}, err
	}

	result := InitResult{
		Home:       home.Path,
		ConfigPath: filepath.Join(home.Path, ConfigFileName),
		Created:    []string{},
		Existing:   []string{},
	}

	for _, dir := range append([]string{""}, StateDirs...) {
		path := filepath.Join(home.Path, dir)
		created, err := ensureDir(path)
		if err != nil {
			return InitResult{}, err
		}
		if created {
			result.Created = append(result.Created, path)
		} else {
			result.Existing = append(result.Existing, path)
		}
	}

	if _, err := os.Stat(result.ConfigPath); err == nil {
		result.Existing = append(result.Existing, result.ConfigPath)
		return result, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return InitResult{}, fmt.Errorf("stat config: %w", err)
	}

	config := DefaultConfig(home.Path)
	if err := WriteJSONAtomic(result.ConfigPath, config); err != nil {
		return InitResult{}, err
	}
	result.Created = append(result.Created, result.ConfigPath)

	return result, nil
}

func DefaultConfig(home string) Config {
	return Config{
		Schema:    "portside.config/v1",
		Home:      home,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Runner: RunnerConfig{
			Default: "gptk",
			Runners: map[string]RunnerEntry{
				"gptk": {
					Command:      "",
					NoHUDCommand: "",
					PrefixMode:   "env",
					EXEPathStyle: "windows",
					Source:       "unconfigured",
				},
			},
		},
		GPTK: GPTKConfig{
			RuntimePackage: GPTKPackage{
				Provider: "official-file",
			},
		},
		Update: UpdateConfig{
			Channel: "stable",
			Source:  "auto",
			Endpoints: []UpdateEndpoint{
				{Name: "global", Manifest: "https://updates.portside.dev/cli/stable/manifest.json"},
				{Name: "cn", Manifest: "https://download-cn.portside.dev/cli/stable/manifest.json"},
			},
		},
	}
}

func LoadConfig() (Config, error) {
	home, err := Resolve()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(filepath.Join(home.Path, ConfigFileName))
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return config, nil
}

func SaveConfig(config Config) error {
	home, err := Resolve()
	if err != nil {
		return err
	}
	if config.Home == "" {
		config.Home = home.Path
	}
	if config.Schema == "" {
		config.Schema = "portside.config/v1"
	}
	return WriteJSONAtomic(filepath.Join(home.Path, ConfigFileName), config)
}

func ConfigureRunner(name string, entry RunnerEntry) (Config, error) {
	if name == "" {
		return Config{}, fmt.Errorf("runner name is required")
	}
	if entry.Command == "" {
		return Config{}, fmt.Errorf("runner command is required")
	}

	config, err := LoadConfig()
	if errors.Is(err, os.ErrNotExist) {
		if _, initErr := Init(); initErr != nil {
			return Config{}, initErr
		}
		config, err = LoadConfig()
	}
	if err != nil {
		return Config{}, err
	}

	if config.Runner.Runners == nil {
		config.Runner.Runners = map[string]RunnerEntry{}
	}
	if entry.PrefixMode == "" {
		entry.PrefixMode = "env"
	}
	if entry.EXEPathStyle == "" {
		entry.EXEPathStyle = "windows"
	}
	if entry.Source == "" {
		entry.Source = "external"
	}
	config.Runner.Default = name
	config.Runner.Runners[name] = entry

	if err := SaveConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func ConfigureGPTKRuntimePackage(entry GPTKPackage) (Config, error) {
	if entry.File == "" {
		return Config{}, fmt.Errorf("gptk runtime package file is required")
	}
	if entry.Provider == "" {
		entry.Provider = "official-file"
	}

	config, err := LoadConfig()
	if errors.Is(err, os.ErrNotExist) {
		if _, initErr := Init(); initErr != nil {
			return Config{}, initErr
		}
		config, err = LoadConfig()
	}
	if err != nil {
		return Config{}, err
	}

	config.GPTK.RuntimePackage = entry

	if err := SaveConfig(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func ConfigExists() (bool, string, error) {
	home, err := Resolve()
	if err != nil {
		return false, "", err
	}

	path := filepath.Join(home.Path, ConfigFileName)
	if _, err := os.Stat(path); err == nil {
		return true, path, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, path, nil
	} else {
		return false, path, err
	}
}

func PrefixesDir(home string) string {
	return filepath.Join(home, "prefixes")
}

func ProfilesDir(home string) string {
	return filepath.Join(home, "profiles")
}

func LogsDir(home string) string {
	return filepath.Join(home, "logs")
}

func InstallersDir(home string) string {
	return filepath.Join(home, "installers")
}

func RunnersDir(home string) string {
	return filepath.Join(home, "runners")
}

func RuntimeOS() string {
	return runtime.GOOS
}

func RuntimeArch() string {
	return runtime.GOARCH
}

func WriteJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return nil
}

func ExpandPath(path string) (string, error) {
	return expandPath(path)
}

func ensureDir(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false, err
	}
	return true, nil
}

func expandPath(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if len(path) > 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

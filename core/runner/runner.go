package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"portside/core/porthome"
)

type ConfiguredRunner struct {
	Name          string `json:"name"`
	Command       string `json:"command"`
	NoHUDCommand  string `json:"no_hud_command,omitempty"`
	ServerCommand string `json:"server_command,omitempty"`
	PrefixMode    string `json:"prefix_mode"`
	EXEPathStyle  string `json:"exe_path_style"`
	Kind          string `json:"kind,omitempty"`
	Source        string `json:"source,omitempty"`
	Version       string `json:"version,omitempty"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
}

type UseOptions struct {
	Name          string
	Command       string
	NoHUDCommand  string
	ServerCommand string
	Kind          string
	Version       string
	Source        string
}

type RunnerPlan struct {
	Runner       string            `json:"runner"`
	Provider     string            `json:"provider"`
	File         string            `json:"file,omitempty"`
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	Configured   *ConfiguredRunner `json:"configured,omitempty"`
	PlannedSteps []string          `json:"planned_steps"`
	NextCommands []string          `json:"next_commands"`
}

func List() ([]ConfiguredRunner, error) {
	config, err := porthome.LoadConfig()
	if errors.Is(err, os.ErrNotExist) {
		return []ConfiguredRunner{}, nil
	}
	if err != nil {
		return nil, err
	}

	result := []ConfiguredRunner{}
	for name, entry := range config.Runner.Runners {
		item := ConfiguredRunner{
			Name:          name,
			Command:       entry.Command,
			NoHUDCommand:  entry.NoHUDCommand,
			ServerCommand: entry.ServerCommand,
			PrefixMode:    entry.PrefixMode,
			EXEPathStyle:  entry.EXEPathStyle,
			Kind:          entry.Kind,
			Source:        entry.Source,
			Version:       entry.Version,
			Status:        "configured",
		}
		if name == config.Runner.Default {
			item.Status = "default"
		}
		if entry.Command == "" {
			item.Status = "missing"
			item.Message = "runner command is not configured"
		}
		result = append(result, item)
	}
	return result, nil
}

func Use(options UseOptions) (ConfiguredRunner, error) {
	if options.Name == "" {
		options.Name = "gptk"
	}
	commandValue := options.Command
	if commandValue == "" && options.Name == "gptk" {
		if discovered, ok := discoverGPTK(); ok {
			commandValue = discovered.Command
			if options.NoHUDCommand == "" {
				options.NoHUDCommand = discovered.NoHUDCommand
			}
			if options.ServerCommand == "" {
				options.ServerCommand = discovered.ServerCommand
			}
			if options.Kind == "" {
				options.Kind = discovered.Kind
			}
			if options.Source == "" {
				options.Source = discovered.Source
			}
		} else {
			commandValue = "gameportingtoolkit"
		}
	}
	command, err := resolveCommand(commandValue)
	if err != nil {
		if options.Command == "" && options.Name == "gptk" {
			return ConfiguredRunner{}, fmt.Errorf("no GPTK runner was found; run portside runner setup gptk after installing a runner, or pass --command <path>")
		}
		return ConfiguredRunner{}, err
	}
	noHUD := ""
	if options.NoHUDCommand != "" {
		noHUD, err = resolveCommand(options.NoHUDCommand)
		if err != nil {
			return ConfiguredRunner{}, err
		}
	} else if options.Name == "gptk" {
		if path, err := exec.LookPath("gameportingtoolkit-no-hud"); err == nil {
			noHUD = path
		}
	}
	server := ""
	if options.ServerCommand != "" {
		server, err = resolveCommand(options.ServerCommand)
		if err != nil {
			return ConfiguredRunner{}, err
		}
	}

	entry := porthome.RunnerEntry{
		Command:       command,
		NoHUDCommand:  noHUD,
		ServerCommand: server,
		PrefixMode:    "env",
		EXEPathStyle:  "windows",
		Kind:          options.Kind,
		Source:        options.Source,
		Version:       options.Version,
	}
	if entry.Kind == "" {
		entry.Kind = "wine"
	}
	if entry.Source == "" {
		entry.Source = "external"
	}

	if _, err := porthome.ConfigureRunner(options.Name, entry); err != nil {
		return ConfiguredRunner{}, err
	}

	return ConfiguredRunner{
		Name:          options.Name,
		Command:       entry.Command,
		NoHUDCommand:  entry.NoHUDCommand,
		ServerCommand: entry.ServerCommand,
		PrefixMode:    entry.PrefixMode,
		EXEPathStyle:  entry.EXEPathStyle,
		Kind:          entry.Kind,
		Source:        entry.Source,
		Version:       entry.Version,
		Status:        "default",
	}, nil
}

func SetupGPTK() (RunnerPlan, error) {
	configured, err := Use(UseOptions{
		Name: "gptk",
	})
	if err == nil {
		return RunnerPlan{
			Runner:       "gptk",
			Provider:     configured.Source,
			Status:       "configured",
			Message:      "Found a GPTK-compatible runner and wrote it to Portside config.",
			Configured:   &configured,
			PlannedSteps: []string{},
			NextCommands: []string{
				"portside doctor",
				"portside prefix create steam-main",
				"portside steam install --prefix steam-main",
			},
		}, nil
	}

	return RunnerPlan{
		Runner:   "gptk",
		Provider: "homebrew-cask",
		Status:   "needs_runner",
		Message:  "No GPTK-compatible runner was found. Install the prebuilt Homebrew runner, then run portside runner setup gptk.",
		PlannedSteps: []string{
			"Install the Gcenx Homebrew tap.",
			"Install the game-porting-toolkit cask.",
			"Run portside runner setup gptk to discover and register wine64.",
		},
		NextCommands: []string{
			"brew tap gcenx/wine",
			"brew install --cask gcenx/wine/game-porting-toolkit",
			"portside runner setup gptk",
		},
	}, nil
}

func ImportGPTKPackagePlan(file string) (RunnerPlan, error) {
	if file == "" {
		return RunnerPlan{
			Runner:   "gptk",
			Provider: "official-file",
			Status:   "needs_file",
			Message:  "GPTK package import requires a local Apple GPTK dmg/pkg. To discover an installed runner, use portside runner setup gptk.",
			PlannedSteps: []string{
				"Use setup to discover a prebuilt runner.",
				"Use import --file to register the local Apple GPTK redist/runtime package.",
			},
			NextCommands: []string{
				"portside runner setup gptk",
				"portside runner import gptk --file ~/Downloads/Game_Porting_Toolkit_3.0.dmg",
			},
		}, nil
	}

	expanded, err := porthome.ExpandPath(file)
	if err != nil {
		return RunnerPlan{}, err
	}
	info, err := os.Stat(expanded)
	if err != nil {
		return RunnerPlan{}, err
	}
	if info.IsDir() {
		return RunnerPlan{}, fmt.Errorf("runner import file must not be a directory: %s", expanded)
	}

	extension := strings.ToLower(filepath.Ext(expanded))
	status := "registered"
	registeredFile := expanded
	message := "File is available in Portside installers and registered as the local Apple GPTK runtime package. Automatic dmg/pkg import is not implemented yet; use runner setup gptk to register an installed prebuilt runner."
	if extension != ".dmg" && extension != ".pkg" && extension != ".zip" {
		status = "unsupported_file_type"
		message = "File exists, but expected a .dmg, .pkg, or .zip from the official GPTK download."
	} else {
		cached, err := cacheGPTKRuntimePackage(expanded)
		if err != nil {
			return RunnerPlan{}, err
		}
		registeredFile = cached
		if _, err := porthome.ConfigureGPTKRuntimePackage(porthome.GPTKPackage{
			Provider: "official-file",
			File:     registeredFile,
		}); err != nil {
			return RunnerPlan{}, err
		}
	}

	return RunnerPlan{
		Runner:   "gptk",
		Provider: "official-file",
		File:     registeredFile,
		Status:   status,
		Message:  message,
		PlannedSteps: []string{
			"Verify Apple-signed or user-provided GPTK package.",
			"Mount or extract the local file.",
			"Locate redist/lib in the Evaluation environment.",
			"Update a registered prebuilt runner's D3DMetal/evaluation environment libraries.",
		},
		NextCommands: []string{
			"portside runner setup gptk",
			"portside doctor",
		},
	}, nil
}

func cacheGPTKRuntimePackage(file string) (string, error) {
	if _, err := porthome.Init(); err != nil {
		return "", err
	}
	home, err := porthome.Resolve()
	if err != nil {
		return "", err
	}
	target := filepath.Join(porthome.InstallersDir(home.Path), filepath.Base(file))
	if sameFile(file, target) {
		return target, nil
	}
	if err := copyFileAtomic(file, target); err != nil {
		return "", err
	}
	return target, nil
}

func copyFileAtomic(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(temp, input); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, target)
}

func sameFile(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

type discoveredRunner struct {
	Command       string
	NoHUDCommand  string
	ServerCommand string
	Kind          string
	Source        string
}

func DiscoverGPTK() (ConfiguredRunner, bool) {
	discovered, ok := discoverGPTK()
	if !ok {
		return ConfiguredRunner{}, false
	}
	return ConfiguredRunner{
		Name:          "gptk",
		Command:       discovered.Command,
		NoHUDCommand:  discovered.NoHUDCommand,
		ServerCommand: discovered.ServerCommand,
		PrefixMode:    "env",
		EXEPathStyle:  "windows",
		Kind:          discovered.Kind,
		Source:        discovered.Source,
		Status:        "discovered",
	}, true
}

func discoverGPTK() (discoveredRunner, bool) {
	if path, err := exec.LookPath("gameportingtoolkit"); err == nil {
		noHUD := ""
		if noHUDPath, err := exec.LookPath("gameportingtoolkit-no-hud"); err == nil {
			noHUD = noHUDPath
		}
		return discoveredRunner{Command: path, NoHUDCommand: noHUD, Kind: "gameportingtoolkit", Source: "path"}, true
	}
	if path, err := exec.LookPath("wine64"); err == nil && looksLikeGPTKWine(path) {
		server := ""
		if serverPath, err := exec.LookPath("wineserver"); err == nil {
			server = serverPath
		}
		return discoveredRunner{Command: path, ServerCommand: server, Kind: "gptk-wine", Source: "path"}, true
	}

	if os.Getenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY") == "1" {
		return discoveredRunner{}, false
	}

	appWine := "/Applications/Game Porting Toolkit.app/Contents/Resources/wine/bin/wine64"
	if fileExists(appWine) {
		server := "/Applications/Game Porting Toolkit.app/Contents/Resources/wine/bin/wineserver"
		if !fileExists(server) {
			server = ""
		}
		return discoveredRunner{Command: appWine, ServerCommand: server, Kind: "gptk-wine", Source: "homebrew-cask"}, true
	}

	for _, pattern := range []string{
		"/opt/homebrew/Caskroom/game-porting-toolkit/*/Game Porting Toolkit.app/Contents/Resources/wine/bin/wine64",
		"/usr/local/Caskroom/game-porting-toolkit/*/Game Porting Toolkit.app/Contents/Resources/wine/bin/wine64",
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, caskWine := range matches {
			if fileExists(caskWine) {
				server := filepath.Join(filepath.Dir(caskWine), "wineserver")
				if !fileExists(server) {
					server = ""
				}
				return discoveredRunner{Command: caskWine, ServerCommand: server, Kind: "gptk-wine", Source: "homebrew-cask"}, true
			}
		}
	}

	return discoveredRunner{}, false
}

func looksLikeGPTKWine(path string) bool {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	return strings.Contains(resolved, "Game Porting Toolkit.app")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func resolveCommand(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("--command is required")
	}
	if !strings.ContainsRune(value, os.PathSeparator) {
		path, err := exec.LookPath(value)
		if err != nil {
			return "", err
		}
		return path, nil
	}
	expanded, err := porthome.ExpandPath(value)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(expanded)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("runner command is a directory: %s", expanded)
	}
	return expanded, nil
}

package doctor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"portside/core/porthome"
	"portside/core/runner"
)

type Report struct {
	Home            string  `json:"home"`
	Config          Check   `json:"config"`
	Directories     []Check `json:"directories"`
	MacOS           Check   `json:"macos"`
	Arch            Check   `json:"arch"`
	Rosetta         Check   `json:"rosetta"`
	GPTKRunner      Check   `json:"gptk_runner"`
	GPTKNoHUDRunner Check   `json:"gptk_no_hud_runner"`
	GPTKRuntime     Check   `json:"gptk_runtime"`
}

type Summary struct {
	Home   string `json:"home"`
	Status string `json:"status"`
	Needs  []Need `json:"needs"`
}

type Need struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Message string   `json:"message"`
	Actions []string `json:"actions"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"`
}

func Summarize(report Report) Summary {
	needs := []Need{}

	if report.Config.Status != "ok" {
		needs = append(needs, Need{
			ID:      "init",
			Title:   "初始化 Portside 数据目录",
			Message: "Portside 还没有完整的数据目录和配置文件。",
			Actions: []string{
				"portside init",
				"portside doctor",
			},
		})
		return Summary{Home: report.Home, Status: "needs_action", Needs: needs}
	}

	if missingDirectory(report.Directories) {
		needs = append(needs, Need{
			ID:      "repair_home",
			Title:   "修复 Portside 数据目录",
			Message: "有状态目录缺失。",
			Actions: []string{
				"portside init",
				"portside doctor",
			},
		})
	}

	if report.MacOS.Status != "ok" || report.Arch.Status != "ok" {
		needs = append(needs, Need{
			ID:      "unsupported_platform",
			Title:   "确认运行平台",
			Message: "Portside 当前优先支持 Apple Silicon Mac。",
			Actions: []string{
				"在 Apple Silicon Mac 上使用 Portside",
			},
		})
	}

	if report.GPTKRunner.Status == "discovered" {
		needs = append(needs, Need{
			ID:      "gptk_runner",
			Title:   "写入 GPTK runner 配置",
			Message: "已经发现本机 GPTK/Wine runner，但还没有写入 Portside 配置。",
			Actions: []string{
				"portside runner setup gptk",
				"portside doctor",
			},
		})
	} else if report.GPTKRunner.Status != "ok" {
		needs = append(needs, Need{
			ID:      "gptk_runner",
			Title:   "配置 GPTK runner",
			Message: "还没有完整的 GPTK/Wine runner，不能真正运行 Windows Steam 或游戏。Apple GPTK 3 dmg 里的 Evaluation environment 是 runtime 组件，不是完整 runner。",
			Actions: []string{
				"添加 Homebrew tap：brew tap gcenx/wine",
				"安装预构建 runner：brew install --cask gcenx/wine/game-porting-toolkit",
				"安装后运行：portside runner setup gptk",
			},
		})
	}

	if report.GPTKRuntime.Status == "discovered" {
		needs = append(needs, Need{
			ID:      "gptk_runtime",
			Title:   "写入 GPTK 官方 runtime 包配置",
			Message: "已经发现本机 Apple GPTK dmg，但还没有写入 Portside 配置。提前登记后，后续导入 D3DMetal/evaluation environment 时不用再找文件。",
			Actions: []string{
				"portside runner import gptk --file " + report.GPTKRuntime.Path,
				"portside doctor",
			},
		})
	} else if report.GPTKRuntime.Status != "ok" {
		needs = append(needs, Need{
			ID:      "gptk_runtime",
			Title:   "准备 GPTK 官方 runtime 包",
			Message: "Apple GPTK 3 dmg 不是 runner，但 Portside 后续导入或更新 D3DMetal/evaluation environment 需要这个本地官方包。建议现在下载并登记到 Portside 配置。",
			Actions: []string{
				"从 Apple Developer 下载 GPTK dmg：https://developer.apple.com/games/game-porting-toolkit/",
				"portside runner import gptk --file ~/Downloads/Game_Porting_Toolkit_3.0.dmg",
				"portside doctor",
			},
		})
	}

	status := "ok"
	if len(needs) > 0 {
		status = "needs_action"
	}

	return Summary{Home: report.Home, Status: status, Needs: needs}
}

func Run() (Report, error) {
	home, err := porthome.Resolve()
	if err != nil {
		return Report{}, err
	}

	configOK, configPath, err := porthome.ConfigExists()
	if err != nil {
		return Report{}, err
	}

	report := Report{
		Home: home.Path,
		Config: Check{
			Name:   "config",
			Status: status(configOK),
			Path:   configPath,
		},
		Directories:     checkDirectories(home.Path),
		MacOS:           checkMacOS(),
		Arch:            checkArch(),
		Rosetta:         checkRosetta(),
		GPTKRunner:      checkRunnerCommand("gptk_runner", false),
		GPTKNoHUDRunner: checkRunnerCommand("gptk_no_hud_runner", true),
		GPTKRuntime:     checkGPTKRuntimePackage(),
	}

	if !configOK {
		report.Config.Message = "Run: portside init"
	}

	return report, nil
}

func checkGPTKRuntimePackage() Check {
	if config, err := porthome.LoadConfig(); err == nil {
		file := config.GPTK.RuntimePackage.File
		if file != "" {
			return checkGPTKRuntimePackageFile(file)
		}
	}

	if discovered := discoverGPTKRuntimePackage(); discovered != "" {
		return Check{Name: "gptk_runtime", Status: "discovered", Path: discovered, Message: "official GPTK runtime package found but not configured"}
	}

	return Check{Name: "gptk_runtime", Status: "missing", Message: "official GPTK runtime package is not configured"}
}

func checkGPTKRuntimePackageFile(file string) Check {
	expanded, err := porthome.ExpandPath(file)
	if err != nil {
		return Check{Name: "gptk_runtime", Status: "error", Path: file, Message: err.Error()}
	}
	info, err := os.Stat(expanded)
	if err != nil {
		return Check{Name: "gptk_runtime", Status: "missing", Path: expanded, Message: err.Error()}
	}
	if info.IsDir() {
		return Check{Name: "gptk_runtime", Status: "error", Path: expanded, Message: "path is a directory"}
	}
	if !isSupportedGPTKRuntimePackage(expanded) {
		return Check{Name: "gptk_runtime", Status: "error", Path: expanded, Message: "expected a .dmg, .pkg, or .zip"}
	}
	return Check{Name: "gptk_runtime", Status: "ok", Path: expanded}
}

func discoverGPTKRuntimePackage() string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	matches, err := filepath.Glob(filepath.Join(userHome, "Downloads", "Game_Porting_Toolkit*.dmg"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	for i := len(matches) - 1; i >= 0; i-- {
		if fileExists(matches[i]) && isSupportedGPTKRuntimePackage(matches[i]) {
			return matches[i]
		}
	}
	return ""
}

func isSupportedGPTKRuntimePackage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dmg", ".pkg", ".zip":
		return true
	default:
		return false
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func checkDirectories(home string) []Check {
	var checks []Check
	for _, dir := range porthome.StateDirs {
		path := filepath.Join(home, dir)
		check := Check{Name: dir, Path: path}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			check.Status = "ok"
		} else if errors.Is(err, os.ErrNotExist) {
			check.Status = "missing"
			check.Message = "Run: portside init"
		} else if err != nil {
			check.Status = "error"
			check.Message = err.Error()
		} else {
			check.Status = "error"
			check.Message = "path exists but is not a directory"
		}
		checks = append(checks, check)
	}
	return checks
}

func checkMacOS() Check {
	if porthome.RuntimeOS() == "darwin" {
		return Check{Name: "macos", Status: "ok", Message: "darwin"}
	}
	return Check{Name: "macos", Status: "warn", Message: "Portside P0 targets macOS"}
}

func checkArch() Check {
	if porthome.RuntimeArch() == "arm64" {
		return Check{Name: "arch", Status: "ok", Message: "arm64"}
	}
	return Check{Name: "arch", Status: "warn", Message: "Apple Silicon arm64 is the primary target"}
}

func checkRosetta() Check {
	if porthome.RuntimeOS() != "darwin" {
		return Check{Name: "rosetta", Status: "warn", Message: "Rosetta check is macOS-only"}
	}
	cmd := exec.Command("/usr/bin/pgrep", "oahd")
	if err := cmd.Run(); err == nil {
		return Check{Name: "rosetta", Status: "ok", Message: "oahd running"}
	}
	return Check{Name: "rosetta", Status: "warn", Message: "Rosetta service not detected"}
}

func checkCommand(name, command string) Check {
	path, err := exec.LookPath(command)
	if err != nil {
		return Check{Name: name, Status: "missing", Message: command + " not found in PATH"}
	}
	return Check{Name: name, Status: "ok", Path: path}
}

func checkRunnerCommand(name string, noHUD bool) Check {
	if config, err := porthome.LoadConfig(); err == nil {
		if entry, ok := config.Runner.Runners["gptk"]; ok {
			command := entry.Command
			if noHUD {
				command = entry.NoHUDCommand
			}
			if command != "" {
				if info, err := os.Stat(command); err == nil && !info.IsDir() {
					return Check{Name: name, Status: "ok", Path: command}
				} else if err != nil {
					return Check{Name: name, Status: "missing", Path: command, Message: err.Error()}
				}
				return Check{Name: name, Status: "error", Path: command, Message: "path is a directory"}
			}
		}
	}
	if discovered, ok := runner.DiscoverGPTK(); ok {
		if noHUD {
			if discovered.NoHUDCommand == "" {
				return Check{Name: name, Status: "missing", Message: "optional no-HUD runner not configured"}
			}
			return Check{Name: name, Status: "discovered", Path: discovered.NoHUDCommand}
		}
		return Check{Name: name, Status: "discovered", Path: discovered.Command}
	}
	if noHUD {
		return checkCommand(name, "gameportingtoolkit-no-hud")
	}
	return checkCommand(name, "gameportingtoolkit")
}

func status(ok bool) string {
	if ok {
		return "ok"
	}
	return "missing"
}

func missingDirectory(checks []Check) bool {
	for _, check := range checks {
		if check.Status != "ok" {
			return true
		}
	}
	return false
}

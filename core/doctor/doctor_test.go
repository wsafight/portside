package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"portside/core/porthome"
)

func TestSummarizePrioritizesInitWhenConfigMissing(t *testing.T) {
	summary := Summarize(Report{
		Home:   "/tmp/portside",
		Config: Check{Name: "config", Status: "missing"},
	})

	if summary.Status != "needs_action" {
		t.Fatalf("status = %s", summary.Status)
	}
	if len(summary.Needs) != 1 || summary.Needs[0].ID != "init" {
		t.Fatalf("needs = %#v", summary.Needs)
	}
}

func TestSummarizePromptsSetupWhenRunnerDiscovered(t *testing.T) {
	summary := Summarize(Report{
		Config:      Check{Name: "config", Status: "ok"},
		MacOS:       Check{Name: "macos", Status: "ok"},
		Arch:        Check{Name: "arch", Status: "ok"},
		GPTKRunner:  Check{Name: "gptk_runner", Status: "discovered"},
		GPTKRuntime: Check{Name: "gptk_runtime", Status: "ok"},
	})

	need := findNeed(summary, "gptk_runner")
	if need == nil {
		t.Fatalf("gptk need missing: %#v", summary.Needs)
	}
	if len(need.Actions) != 2 || need.Actions[0] != "portside runner setup gptk" {
		t.Fatalf("actions = %#v", need.Actions)
	}
}

func TestSummarizePromptsInstallWhenRunnerMissing(t *testing.T) {
	summary := Summarize(Report{
		Config:      Check{Name: "config", Status: "ok"},
		MacOS:       Check{Name: "macos", Status: "ok"},
		Arch:        Check{Name: "arch", Status: "ok"},
		GPTKRunner:  Check{Name: "gptk_runner", Status: "missing"},
		GPTKRuntime: Check{Name: "gptk_runtime", Status: "ok"},
	})

	need := findNeed(summary, "gptk_runner")
	if need == nil {
		t.Fatalf("gptk need missing: %#v", summary.Needs)
	}
	if !contains(need.Actions, "安装预构建 runner：brew install --cask gcenx/wine/game-porting-toolkit") {
		t.Fatalf("actions = %#v", need.Actions)
	}
	if !contains(need.Actions, "安装后运行：portside runner setup gptk") {
		t.Fatalf("actions = %#v", need.Actions)
	}
}

func TestSummarizePromptsImportWhenRuntimePackageDiscovered(t *testing.T) {
	summary := Summarize(Report{
		Config:      Check{Name: "config", Status: "ok"},
		MacOS:       Check{Name: "macos", Status: "ok"},
		Arch:        Check{Name: "arch", Status: "ok"},
		GPTKRunner:  Check{Name: "gptk_runner", Status: "ok"},
		GPTKRuntime: Check{Name: "gptk_runtime", Status: "discovered", Path: "/tmp/Game_Porting_Toolkit_3.0.dmg"},
	})

	need := findNeed(summary, "gptk_runtime")
	if need == nil {
		t.Fatalf("gptk runtime need missing: %#v", summary.Needs)
	}
	if !contains(need.Actions, "portside runner import gptk --file /tmp/Game_Porting_Toolkit_3.0.dmg") {
		t.Fatalf("actions = %#v", need.Actions)
	}
}

func TestSummarizePromptsDownloadWhenRuntimePackageMissing(t *testing.T) {
	summary := Summarize(Report{
		Config:      Check{Name: "config", Status: "ok"},
		MacOS:       Check{Name: "macos", Status: "ok"},
		Arch:        Check{Name: "arch", Status: "ok"},
		GPTKRunner:  Check{Name: "gptk_runner", Status: "ok"},
		GPTKRuntime: Check{Name: "gptk_runtime", Status: "missing"},
	})

	need := findNeed(summary, "gptk_runtime")
	if need == nil {
		t.Fatalf("gptk runtime need missing: %#v", summary.Needs)
	}
	if !contains(need.Actions, "portside runner import gptk --file ~/Downloads/Game_Porting_Toolkit_3.0.dmg") {
		t.Fatalf("actions = %#v", need.Actions)
	}
}

func TestRunReportsDiscoveredRunnerBeforeSetup(t *testing.T) {
	setupHome(t)
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")
	fakeGPTKWineOnPath(t)
	runtimePackage := filepath.Join(t.TempDir(), "Game_Porting_Toolkit_3.0.dmg")
	if err := os.WriteFile(runtimePackage, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := porthome.ConfigureGPTKRuntimePackage(porthome.GPTKPackage{File: runtimePackage}); err != nil {
		t.Fatal(err)
	}

	report, err := Run()
	if err != nil {
		t.Fatal(err)
	}
	if report.Config.Status != "ok" {
		t.Fatalf("config status = %s", report.Config.Status)
	}
	if report.GPTKRunner.Status != "discovered" {
		t.Fatalf("runner status = %s", report.GPTKRunner.Status)
	}
	summary := Summarize(report)
	if findNeed(summary, "gptk_runner") == nil {
		t.Fatalf("expected gptk need: %#v", summary.Needs)
	}
}

func findNeed(summary Summary, id string) *Need {
	for i := range summary.Needs {
		if summary.Needs[i].ID == id {
			return &summary.Needs[i]
		}
	}
	return nil
}

func setupHome(t *testing.T) {
	t.Helper()

	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	if _, err := porthome.Init(); err != nil {
		t.Fatal(err)
	}
}

func fakeGPTKWineOnPath(t *testing.T) {
	t.Helper()

	appBin := filepath.Join(t.TempDir(), "Game Porting Toolkit.app", "Contents", "Resources", "wine", "bin")
	if err := os.MkdirAll(appBin, 0o755); err != nil {
		t.Fatal(err)
	}
	wine64 := filepath.Join(appBin, "wine64")
	wineserver := filepath.Join(appBin, "wineserver")
	writeExecutable(t, wine64)
	writeExecutable(t, wineserver)

	binDir := t.TempDir()
	if err := os.Symlink(wine64, filepath.Join(binDir, "wine64")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wineserver, filepath.Join(binDir, "wineserver")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

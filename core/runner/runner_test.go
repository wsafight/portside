package runner

import (
	"os"
	"path/filepath"
	"testing"

	"portside/core/porthome"
)

func TestUseStoresExplicitRunner(t *testing.T) {
	setupHome(t)
	command := filepath.Join(t.TempDir(), "wine64")
	server := filepath.Join(t.TempDir(), "wineserver")
	writeExecutable(t, command)
	writeExecutable(t, server)

	configured, err := Use(UseOptions{
		Name:          "gptk",
		Command:       command,
		ServerCommand: server,
		Kind:          "gptk-wine",
		Source:        "manual",
		Version:       "3.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if configured.Command != command {
		t.Fatalf("command = %s, want %s", configured.Command, command)
	}
	if configured.ServerCommand != server {
		t.Fatalf("server command = %s, want %s", configured.ServerCommand, server)
	}
	if configured.Kind != "gptk-wine" {
		t.Fatalf("kind = %s", configured.Kind)
	}

	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Status != "default" {
		t.Fatalf("runner list = %#v", list)
	}
}

func TestUseFindsGamePortingToolkitInPath(t *testing.T) {
	setupHome(t)
	binDir := t.TempDir()
	command := filepath.Join(binDir, "gameportingtoolkit")
	noHUD := filepath.Join(binDir, "gameportingtoolkit-no-hud")
	writeExecutable(t, command)
	writeExecutable(t, noHUD)
	t.Setenv("PATH", binDir)

	configured, err := Use(UseOptions{Name: "gptk"})
	if err != nil {
		t.Fatal(err)
	}
	if configured.Command != command {
		t.Fatalf("command = %s, want %s", configured.Command, command)
	}
	if configured.NoHUDCommand != noHUD {
		t.Fatalf("no-hud command = %s, want %s", configured.NoHUDCommand, noHUD)
	}
	if configured.Kind != "gameportingtoolkit" {
		t.Fatalf("kind = %s", configured.Kind)
	}
}

func TestDiscoverGPTKWineFromPath(t *testing.T) {
	setupHome(t)
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")
	wine64, wineserver := fakeGPTKWineOnPath(t)

	discovered, ok := DiscoverGPTK()
	if !ok {
		t.Fatal("expected GPTK wine discovery")
	}
	if discovered.Command != wine64 {
		t.Fatalf("command = %s, want %s", discovered.Command, wine64)
	}
	if discovered.ServerCommand != wineserver {
		t.Fatalf("server command = %s, want %s", discovered.ServerCommand, wineserver)
	}
	if discovered.Kind != "gptk-wine" {
		t.Fatalf("kind = %s", discovered.Kind)
	}
}

func TestSetupGPTKRegistersDiscoveredWine(t *testing.T) {
	setupHome(t)
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")
	wine64, _ := fakeGPTKWineOnPath(t)

	plan, err := SetupGPTK()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "configured" {
		t.Fatalf("status = %s", plan.Status)
	}
	if plan.Configured == nil {
		t.Fatal("expected configured runner")
	}
	if plan.Configured.Command != wine64 {
		t.Fatalf("configured command = %s, want %s", plan.Configured.Command, wine64)
	}
}

func TestSetupGPTKWithoutRunnerReturnsGuidance(t *testing.T) {
	setupHome(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")

	plan, err := SetupGPTK()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "needs_runner" {
		t.Fatalf("status = %s", plan.Status)
	}
	if plan.Provider != "homebrew-cask" {
		t.Fatalf("provider = %s", plan.Provider)
	}
	if !contains(plan.NextCommands, "brew install --cask gcenx/wine/game-porting-toolkit") {
		t.Fatalf("next commands = %#v", plan.NextCommands)
	}
}

func TestImportGPTKPackagePlanRequiresFileOrRegistersPackage(t *testing.T) {
	setupHome(t)

	empty, err := ImportGPTKPackagePlan("")
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != "needs_file" {
		t.Fatalf("empty status = %s", empty.Status)
	}

	file := filepath.Join(t.TempDir(), "Game_Porting_Toolkit_3.0.dmg")
	if err := os.WriteFile(file, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := ImportGPTKPackagePlan(file)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "registered" {
		t.Fatalf("file status = %s", plan.Status)
	}
	home, err := porthome.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	cachedFile := filepath.Join(porthome.InstallersDir(home.Path), filepath.Base(file))
	if plan.File != cachedFile {
		t.Fatalf("file = %s, want %s", plan.File, cachedFile)
	}
	cachedData, err := os.ReadFile(cachedFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(cachedData) != "fake" {
		t.Fatalf("cached file data = %q", string(cachedData))
	}
	config, err := porthome.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.GPTK.RuntimePackage.File != cachedFile {
		t.Fatalf("configured runtime package = %s, want %s", config.GPTK.RuntimePackage.File, cachedFile)
	}

	unsupported := filepath.Join(t.TempDir(), "gptk.txt")
	if err := os.WriteFile(unsupported, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = ImportGPTKPackagePlan(unsupported)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "unsupported_file_type" {
		t.Fatalf("unsupported status = %s", plan.Status)
	}
}

func TestImportGPTKPackagePlanRejectsDirectory(t *testing.T) {
	setupHome(t)

	if _, err := ImportGPTKPackagePlan(t.TempDir()); err == nil {
		t.Fatal("expected directory error")
	}
}

func setupHome(t *testing.T) {
	t.Helper()

	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	if _, err := porthome.Init(); err != nil {
		t.Fatal(err)
	}
}

func fakeGPTKWineOnPath(t *testing.T) (string, string) {
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
	wine64Link := filepath.Join(binDir, "wine64")
	wineserverLink := filepath.Join(binDir, "wineserver")
	if err := os.Symlink(wine64, wine64Link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wineserver, wineserverLink); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	return wine64Link, wineserverLink
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

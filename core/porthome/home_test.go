package porthome

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesHomeConfigAndStateDirs(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	result, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	if result.Home != home {
		t.Fatalf("home = %s, want %s", result.Home, home)
	}
	if result.ConfigPath != filepath.Join(home, ConfigFileName) {
		t.Fatalf("config path = %s", result.ConfigPath)
	}
	for _, dir := range StateDirs {
		path := filepath.Join(home, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("state dir %s missing: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("state path %s is not a directory", path)
		}
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Schema != "portside.config/v1" {
		t.Fatalf("schema = %s", config.Schema)
	}
	if config.Runner.Default != "gptk" {
		t.Fatalf("default runner = %s", config.Runner.Default)
	}
	if got := config.Runner.Runners["gptk"].Source; got != "unconfigured" {
		t.Fatalf("default gptk source = %s", got)
	}
	if got := config.GPTK.RuntimePackage.Provider; got != "official-file" {
		t.Fatalf("default gptk runtime package provider = %s", got)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	first, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Created) == 0 {
		t.Fatal("first init should create paths")
	}
	if len(second.Created) != 0 {
		t.Fatalf("second init created paths: %#v", second.Created)
	}
	if len(second.Existing) == 0 {
		t.Fatal("second init should report existing paths")
	}
}

func TestConfigureRunnerInitializesMissingHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	command := filepath.Join(t.TempDir(), "wine64")
	writeExecutable(t, command)
	config, err := ConfigureRunner("gptk", RunnerEntry{
		Command:       command,
		ServerCommand: filepath.Join(t.TempDir(), "wineserver"),
		Kind:          "gptk-wine",
		Source:        "manual",
		Version:       "3.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := config.Runner.Runners["gptk"]
	if entry.Command != command {
		t.Fatalf("command = %s, want %s", entry.Command, command)
	}
	if entry.PrefixMode != "env" {
		t.Fatalf("prefix mode = %s", entry.PrefixMode)
	}
	if entry.EXEPathStyle != "windows" {
		t.Fatalf("exe path style = %s", entry.EXEPathStyle)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Runner.Runners["gptk"].Source != "manual" {
		t.Fatalf("loaded source = %s", loaded.Runner.Runners["gptk"].Source)
	}
}

func TestConfigureGPTKRuntimePackageInitializesMissingHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	file := filepath.Join(t.TempDir(), "Game_Porting_Toolkit_3.0.dmg")
	if err := os.WriteFile(file, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := ConfigureGPTKRuntimePackage(GPTKPackage{File: file})
	if err != nil {
		t.Fatal(err)
	}
	if config.GPTK.RuntimePackage.File != file {
		t.Fatalf("runtime package file = %s, want %s", config.GPTK.RuntimePackage.File, file)
	}
	if config.GPTK.RuntimePackage.Provider != "official-file" {
		t.Fatalf("runtime package provider = %s", config.GPTK.RuntimePackage.Provider)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GPTK.RuntimePackage.File != file {
		t.Fatalf("loaded runtime package file = %s, want %s", loaded.GPTK.RuntimePackage.File, file)
	}
}

func TestWriteJSONAtomicCreatesParentAndValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "value.json")
	value := map[string]any{"name": "portside", "ok": true}

	if err := WriteJSONAtomic(path, value); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("written file is not valid json: %v\n%s", err, string(data))
	}
	if decoded["name"] != "portside" {
		t.Fatalf("decoded name = %v", decoded["name"])
	}
}

func TestExpandPathExpandsHome(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ExpandPath("~/Games/portside")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(userHome, "Games", "portside")
	if got != want {
		t.Fatalf("expanded path = %s, want %s", got, want)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInitPrefixGameFlow(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	runOK(t, "init", "--json")
	prefixList := runOK(t, "--json", "prefix", "list")
	assertJSONDataArrayLen(t, prefixList, 0)

	runOK(t, "prefix", "create", "steam-main", "--json")
	gameAdd := runOK(t, "--json", "game", "add", "elden-ring", "--appid", "1245620", "--prefix", "steam-main", "--name", "Elden Ring")
	assertJSONField(t, gameAdd, "id", "elden-ring")

	gameShow := runOK(t, "game", "show", "elden-ring", "--json")
	assertJSONField(t, gameShow, "prefix", "steam-main")

	dryRun := runOK(t, "--json", "run", "elden-ring", "--dry-run")
	assertJSONField(t, dryRun, "status", "dry_run")

	gameInstall := runOK(t, "--json", "game", "install", "elden-ring")
	assertJSONField(t, gameInstall, "status", "manual_steam_required")
}

func TestCLIRunnerUseFlow(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	runOK(t, "init")
	command := filepath.Join(t.TempDir(), "gameportingtoolkit")
	writeExecutable(t, command)

	used := runOK(t, "--json", "runner", "use", "gptk", "--command", command, "--version", "test")
	assertJSONField(t, used, "status", "default")

	list := runOK(t, "--json", "runner", "list")
	assertJSONDataArrayLen(t, list, 1)
}

func TestCLIRunnerUseFindsGPTKInPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	binDir := t.TempDir()
	command := filepath.Join(binDir, "gameportingtoolkit")
	noHUD := filepath.Join(binDir, "gameportingtoolkit-no-hud")
	writeExecutable(t, command)
	writeExecutable(t, noHUD)
	t.Setenv("PATH", binDir)

	runOK(t, "init")
	used := runOK(t, "--json", "runner", "use", "gptk")
	assertJSONField(t, used, "command", command)
	assertJSONField(t, used, "no_hud_command", noHUD)
}

func TestCLIRunnerSetupUsesGPTKFromPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	binDir := t.TempDir()
	command := filepath.Join(binDir, "gameportingtoolkit")
	writeExecutable(t, command)
	t.Setenv("PATH", binDir)

	runOK(t, "init")
	setup := runOK(t, "--json", "runner", "setup", "gptk")
	assertJSONField(t, setup, "status", "configured")

	list := runOK(t, "--json", "runner", "list")
	assertJSONDataArrayLen(t, list, 1)
}

func TestCLIRunnerSetupUsesGPTKWineFromPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")

	wine64, wineserver := fakeGPTKWineOnPath(t)

	runOK(t, "init")
	setup := runOK(t, "--json", "runner", "setup", "gptk")
	assertJSONField(t, setup, "status", "configured")
	assertJSONNestedField(t, setup, "configured", "command", wine64)
	assertJSONNestedField(t, setup, "configured", "server_command", wineserver)
	assertJSONNestedField(t, setup, "configured", "kind", "gptk-wine")

	list := runOK(t, "--json", "runner", "list")
	assertJSONDataArrayLen(t, list, 1)
}

func TestCLIRunnerSetupWithoutRunnerReturnsGuidance(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")

	runOK(t, "init")
	setup := runOK(t, "--json", "runner", "setup", "gptk")
	assertJSONField(t, setup, "status", "needs_runner")
	assertJSONField(t, setup, "provider", "homebrew-cask")
	assertJSONSliceContains(t, setup, "next_commands", "brew install --cask gcenx/wine/game-porting-toolkit")
	assertJSONSliceContains(t, setup, "next_commands", "portside runner setup gptk")
}

func TestCLIRunnerUseStoresServerCommand(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	runOK(t, "init")
	command := filepath.Join(t.TempDir(), "wine64")
	server := filepath.Join(t.TempDir(), "wineserver")
	writeExecutable(t, command)
	writeExecutable(t, server)

	used := runOK(t, "--json", "runner", "use", "gptk", "--command", command, "--server-command", server, "--source", "manual", "--version", "3.0")
	assertJSONField(t, used, "command", command)
	assertJSONField(t, used, "server_command", server)
	assertJSONField(t, used, "source", "manual")
	assertJSONField(t, used, "version", "3.0")
}

func TestCLIRunnerImportWithoutFileReturnsGuidance(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")

	runOK(t, "init")
	plan := runOK(t, "--json", "runner", "import", "gptk")
	assertJSONField(t, plan, "status", "needs_file")
}

func TestCLIRunnerImportWithFileRegistersPackage(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	runOK(t, "init")
	file := filepath.Join(t.TempDir(), "Game_Porting_Toolkit_3.0.dmg")
	if err := os.WriteFile(file, []byte("fake dmg"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := runOK(t, "--json", "runner", "import", "gptk", "--file", file)
	assertJSONField(t, plan, "status", "registered")
	assertJSONField(t, plan, "file", filepath.Join(home, "installers", filepath.Base(file)))
	assertJSONSliceContains(t, plan, "next_commands", "portside runner setup gptk")
}

func TestCLIRunnerImportRejectsDirectory(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	runOK(t, "init")
	err := runErr(t, "--json", "runner", "import", "gptk", "--file", t.TempDir())
	if !strings.Contains(err.Error(), "must not be a directory") {
		t.Fatalf("expected directory error, got %v", err)
	}
}

func TestCLIRunnerInstallIsNotACommand(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)

	runOK(t, "init")
	err := runErr(t, "--json", "runner", "install", "gptk")
	if !strings.Contains(err.Error(), "unknown runner command: install") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestCLIDoctorDefaultsToSummary(t *testing.T) {
	t.Setenv("PORTSIDE_HOME", filepath.Join(t.TempDir(), "home"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")

	output := runOK(t, "--json", "doctor")
	assertJSONField(t, output, "status", "needs_action")

	var envelope struct {
		OK   bool                   `json:"ok"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, string(output))
	}
	if _, ok := envelope.Data["directories"]; ok {
		t.Fatalf("default doctor should not include verbose directories: %s", string(output))
	}

	runOK(t, "init")
	output = runOK(t, "--json", "doctor")
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("unmarshal initialized output: %v\n%s", err, string(output))
	}
	assertNeedActions(t, envelope.Data, "gptk_runner", []string{
		"添加 Homebrew tap：brew tap gcenx/wine",
		"安装预构建 runner：brew install --cask gcenx/wine/game-porting-toolkit",
		"安装后运行：portside runner setup gptk",
	})
	assertNeedActions(t, envelope.Data, "gptk_runtime", []string{
		"从 Apple Developer 下载 GPTK dmg：https://developer.apple.com/games/game-porting-toolkit/",
		"portside runner import gptk --file ~/Downloads/Game_Porting_Toolkit_3.0.dmg",
		"portside doctor",
	})

	verbose := runOK(t, "--json", "doctor", "--verbose")
	if err := json.Unmarshal(verbose, &envelope); err != nil {
		t.Fatalf("unmarshal verbose output: %v\n%s", err, string(verbose))
	}
	if _, ok := envelope.Data["directories"]; !ok {
		t.Fatalf("verbose doctor should include directories: %s", string(verbose))
	}
}

func TestCLIDoctorPromptsSetupWhenRunnerIsDiscovered(t *testing.T) {
	t.Setenv("PORTSIDE_HOME", filepath.Join(t.TempDir(), "home"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PORTSIDE_DISABLE_GPTK_APP_DISCOVERY", "1")
	fakeGPTKWineOnPath(t)

	runOK(t, "init")
	output := runOK(t, "--json", "doctor")
	assertNeedActions(t, jsonDataMap(t, output), "gptk_runner", []string{
		"portside runner setup gptk",
		"portside doctor",
	})

	runOK(t, "--json", "runner", "setup", "gptk")
	output = runOK(t, "--json", "doctor")
	assertNoNeed(t, jsonDataMap(t, output), "gptk_runner")
}

func assertNeedActions(t *testing.T, data map[string]interface{}, id string, want []string) {
	t.Helper()

	needs, ok := data["needs"].([]interface{})
	if !ok {
		t.Fatalf("needs is missing or not an array: %#v", data["needs"])
	}
	for _, rawNeed := range needs {
		need, ok := rawNeed.(map[string]interface{})
		if !ok {
			continue
		}
		if need["id"] != id {
			continue
		}
		rawActions, ok := need["actions"].([]interface{})
		if !ok {
			t.Fatalf("actions missing for need %s: %#v", id, need)
		}
		if len(rawActions) != len(want) {
			t.Fatalf("actions length = %d, want %d: %#v", len(rawActions), len(want), rawActions)
		}
		for i, action := range rawActions {
			if action != want[i] {
				t.Fatalf("action[%d] = %v, want %s", i, action, want[i])
			}
		}
		return
	}
	t.Fatalf("need %s not found in %#v", id, needs)
}

func assertNoNeed(t *testing.T, data map[string]interface{}, id string) {
	t.Helper()

	needs, ok := data["needs"].([]interface{})
	if !ok {
		t.Fatalf("needs is missing or not an array: %#v", data["needs"])
	}
	for _, rawNeed := range needs {
		need, ok := rawNeed.(map[string]interface{})
		if ok && need["id"] == id {
			t.Fatalf("need %s should not be present: %#v", id, needs)
		}
	}
}

func TestCLIJSONErrorEnvelope(t *testing.T) {
	t.Setenv("PORTSIDE_HOME", filepath.Join(t.TempDir(), "home"))

	var stdout, stderr bytes.Buffer
	err := mainErr([]string{"--json", "game", "show", "missing-game"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected an error")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var envelope struct {
		OK    bool            `json:"ok"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if envelope.OK {
		t.Fatalf("expected ok=false, got output %s", stdout.String())
	}
	if len(envelope.Error) == 0 {
		t.Fatalf("expected error object, got output %s", stdout.String())
	}
}

func runOK(t *testing.T, args ...string) []byte {
	t.Helper()

	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("run %v: %v\nstdout:\n%s\nstderr:\n%s", args, err, stdout.String(), stderr.String())
	}
	return stdout.Bytes()
}

func runErr(t *testing.T, args ...string) error {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := run(args, &stdout, &stderr)
	if err == nil {
		t.Fatalf("run %v: expected error\nstdout:\n%s\nstderr:\n%s", args, stdout.String(), stderr.String())
	}
	return err
}

func jsonDataMap(t *testing.T, output []byte) map[string]interface{} {
	t.Helper()

	var envelope struct {
		OK   bool                   `json:"ok"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, string(output))
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true, got %s", string(output))
	}
	return envelope.Data
}

func assertJSONDataArrayLen(t *testing.T, output []byte, want int) {
	t.Helper()

	var envelope struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, string(output))
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true, got %s", string(output))
	}
	var data []any
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("unmarshal data array: %v\n%s", err, string(output))
	}
	if len(data) != want {
		t.Fatalf("array length = %d, want %d\n%s", len(data), want, string(output))
	}
}

func assertJSONField(t *testing.T, output []byte, field string, want any) {
	t.Helper()

	var envelope struct {
		OK   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, string(output))
	}
	if !envelope.OK {
		t.Fatalf("expected ok=true, got %s", string(output))
	}
	if got := envelope.Data[field]; got != want {
		t.Fatalf("data[%s] = %v, want %v\n%s", field, got, want, string(output))
	}
}

func assertJSONNestedField(t *testing.T, output []byte, objectField, field string, want any) {
	t.Helper()

	data := jsonDataMap(t, output)
	rawObject, ok := data[objectField].(map[string]interface{})
	if !ok {
		t.Fatalf("data[%s] is missing or not an object: %#v\n%s", objectField, data[objectField], string(output))
	}
	if got := rawObject[field]; got != want {
		t.Fatalf("data[%s][%s] = %v, want %v\n%s", objectField, field, got, want, string(output))
	}
}

func assertJSONSliceContains(t *testing.T, output []byte, field, want string) {
	t.Helper()

	data := jsonDataMap(t, output)
	values, ok := data[field].([]interface{})
	if !ok {
		t.Fatalf("data[%s] is missing or not an array: %#v\n%s", field, data[field], string(output))
	}
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("data[%s] does not contain %q: %#v\n%s", field, want, values, string(output))
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

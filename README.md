# Portside

Portside is a GPTK environment manager for Apple Silicon Macs. It aims to make Windows Steam game environments repeatable, observable, and recoverable instead of leaving users with scattered Wine prefixes, shell scripts, and logs.

The current focus is the CLI and the agent skill. The TUI is the next interaction layer for the CLI and is entered with `portside tui`. The SwiftUI GUI remains a separate macOS app project and will call the same CLI protocol through a bundled `portside` helper.

## Current Capabilities

- Initialize `PORTSIDE_HOME`.
- Check the local machine and Portside state.
- Configure a local GPTK-compatible runner.
- Create and list prefixes.
- Add, list, and inspect game profiles.
- Emit a stable JSON envelope for automation.
- Generate `run --dry-run` launch specifications.
- Return guided Windows Steam and Steam game install steps.
- Reserve real runner adapter, logs, snapshots, GUI, and TUI implementation work.
- Provide `skills/portside-cli` for agents that need to call Portside safely.

## Project Layout

```text
portside/
  cmd/portside/          # Go CLI entry point
  core/                  # Shared Go core
  internal/tui/          # Internal implementation for `portside tui`
  apps/gui/              # SwiftUI macOS app scaffold
  skills/portside-cli/   # Portside CLI agent skill
```

## Build

```bash
go test ./...
go build -o bin/portside ./cmd/portside
```

SwiftUI GUI scaffold:

```bash
swift build --package-path apps/gui
```

If `go run` or `go build` cannot write to the default Go cache in a restricted sandbox, use a temporary cache directory:

```bash
GOCACHE=/private/tmp/portside-go-cache go run ./cmd/portside help
```

## CLI Usage

The default state directory is:

```text
~/Games/portside
```

For development and tests, use a temporary home to avoid touching real state:

```bash
export PORTSIDE_HOME=/private/tmp/portside-demo
```

Initialize Portside:

```bash
./bin/portside init
./bin/portside doctor
```

`doctor` shows only the currently required actions by default. Use verbose mode for the full check report:

```bash
./bin/portside doctor --verbose
```

If no GPTK-compatible runner is configured, `doctor` points to a complete prebuilt runner. Apple's official GPTK 3 dmg contains the `Evaluation environment for Windows games 3.0` runtime components; it is not a complete runner by itself.

```text
Add the Homebrew tap: brew tap gcenx/wine
Install the prebuilt runner: brew install --cask gcenx/wine/game-porting-toolkit
After installation, run: portside runner setup gptk
Download and register the official Apple GPTK 3 dmg for later D3DMetal/evaluation environment runtime updates.
```

If Homebrew has already installed the runner, `doctor` should only ask for:

```bash
./bin/portside runner setup gptk
```

That command does not reinstall GPTK. It discovers the local `wine64` and `wineserver` binaries and writes them to Portside config.

### Set Up GPTK

Portside does not bundle or redistribute GPTK. The recommended flow is to install a complete runner first, then let Portside register it. The Apple GPTK `Evaluation environment` dmg is a redist/runtime component and cannot be used alone as the runner.

Recommended setup:

```bash
brew tap gcenx/wine
brew install --cask gcenx/wine/game-porting-toolkit
./bin/portside runner setup gptk
```

`runner setup gptk` searches for an installed GPTK-compatible runner. The Homebrew cask commonly exposes `wine64` and `wineserver`, for example `/opt/homebrew/bin/wine64` pointing into `/Applications/Game Porting Toolkit.app/.../wine64`. Once found, Portside stores the path in config so GUI, scripts, and non-interactive environments do not depend on the current shell `PATH`.

You can also register a known runner manually:

```bash
./bin/portside runner use gptk
./bin/portside runner use gptk --command /path/to/wine64 --server-command /path/to/wineserver
./bin/portside runner list
./bin/portside doctor
```

If `gameportingtoolkit` or the GPTK cask `wine64` is already in `PATH`, this also works:

```bash
./bin/portside runner use gptk
```

`runner setup gptk` is still the preferred discovery command. Use `runner use` when you already know the exact command path.

The official GPTK runtime package flow registers the local Apple dmg/pkg/zip now and leaves extraction/import as a future step:

```bash
./bin/portside runner import gptk --file ~/Downloads/Game_Porting_Toolkit_3.0.dmg
```

`runner import gptk` does not install the Homebrew cask and does not download Apple files. Without `--file`, it asks for a local official file. With `--file`, it verifies the local dmg/pkg/zip, copies it into Portside's `installers/` directory, writes that path to Portside config, and returns the future import plan.

## JSON Output

Both global and command-local JSON flags are supported:

```bash
./bin/portside --json doctor
./bin/portside doctor --json
```

Default JSON doctor output is a summary:

```json
{
  "ok": true,
  "data": {
    "status": "needs_action",
    "needs": []
  }
}
```

Use verbose JSON for the full check report:

```bash
./bin/portside --json doctor --verbose
```

## Prefixes And Games

Create a prefix:

```bash
./bin/portside prefix create steam-main
./bin/portside prefix list
```

Windows-only games should not be managed through Mac Steam. Portside's intended flow is to install Windows Steam inside a prefix, then let that Windows Steam install and run the game.

```bash
./bin/portside steam install --prefix steam-main
./bin/portside steam open --prefix steam-main
```

These commands currently return structured plans. Real execution will be added with the runner adapter. The target flow is:

```text
1. Download SteamSetup.exe from Steam.
2. Run SteamSetup.exe inside the steam-main prefix through the GPTK runner.
3. Sign in to Windows Steam.
4. Install future games through that Windows Steam instance.
```

Add a game profile:

```bash
./bin/portside game add elden-ring --appid 1245620 --prefix steam-main --name "Elden Ring"
./bin/portside game list
./bin/portside game show elden-ring
```

Install a Steam game:

```bash
./bin/portside game install elden-ring
```

This command does not silently download the game. It guides you back into the Windows Steam instance in the same prefix:

```text
1. portside steam open --prefix steam-main
2. Install AppID 1245620 inside Windows Steam.
3. After installation, run portside run elden-ring.
```

Steam login, library selection, EULA prompts, and first-run dependency installers may still require user interaction inside Windows Steam.

Inspect the launch spec:

```bash
./bin/portside --json run elden-ring --dry-run
```

Open the TUI placeholder:

```bash
./bin/portside tui
```

Check update metadata:

```bash
./bin/portside update check
./bin/portside --json update check
```

## Current Limits

- `steam install/open` currently returns structured placeholder results.
- `run` does not launch processes yet; use `--dry-run` for the current behavior.
- `runner setup gptk` can register an installed GPTK-compatible runner.
- `runner import gptk --file` validates, copies, and records the local official runtime package, then returns an import plan; it does not mount or extract the package yet.
- The GPTK runner adapter, real Windows Steam installation, run records, and snapshot/restore are not implemented yet.
- Snapshots should use Go's standard library to produce `.tar.gz`; no extra `zstd` installation is required.
- Config and profiles currently use JSON. Future versions may migrate formats, but external tools should use the CLI instead of editing state files directly.

## Agent Skill

The Portside CLI skill lives at:

```text
skills/portside-cli/
```

It describes how code agents should use `portside` safely:

- Prefer `portside --json ...`.
- Do not edit files inside `PORTSIDE_HOME` directly.
- Do not hand-build GPTK/Wine launch commands.
- Diagnose through `doctor`, `prefix list`, `game list`, and `run --dry-run`.

The skill can later be installed into a local Codex skills directory.

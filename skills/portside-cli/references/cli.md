# Portside CLI Reference

## Command Shape

Global JSON mode:

```text
portside --json doctor
```

Command-local JSON mode:

```text
portside doctor --json
```

Both forms should be accepted by clients.

Default `doctor` JSON is a summary:

```json
{
  "ok": true,
  "data": {
    "home": "~/Games/portside",
    "status": "needs_action",
    "needs": [
      {
        "id": "gptk_runner",
        "title": "配置 GPTK runner",
        "actions": [
          "添加 Homebrew tap：brew tap gcenx/wine",
          "安装预构建 runner：brew install --cask gcenx/wine/game-porting-toolkit",
          "安装后运行：portside runner setup gptk",
          "Apple 官方 GPTK 3 dmg 可先保留，后续用于更新 runner 的 D3DMetal/evaluation environment 运行库：https://developer.apple.com/games/game-porting-toolkit/"
        ]
      }
    ]
  }
}
```

Use `portside --json doctor --verbose` for the full check report.

## P0 Commands

```text
portside init [--json]
portside doctor [--json] [--verbose]
portside runner list [--json]
portside runner doctor [--json]
portside runner setup gptk [--json]
portside runner use <name> [--command <path>] [--no-hud-command <path>] [--server-command <path>] [--version <version>] [--json]
portside runner import gptk [--file <dmg-or-pkg>] [--json]
portside prefix create <id> [--json]
portside prefix list [--json]
portside game add <id> --appid <appid> --prefix <prefix> [--name <name>] [--json]
portside game install <id> [--json]
portside game list [--json]
portside game show <id> [--json]
portside run <game> [--dry-run] [--json]
portside logs [game] [--json]
portside steam install --prefix <prefix> [--json]
portside steam open --prefix <prefix> [--json]
portside update check [--json]
portside tui
```

## JSON Envelope

Successful query:

```json
{
  "ok": true,
  "data": {}
}
```

Failed query:

```json
{
  "ok": false,
  "error": {
    "code": "prefix_not_found",
    "message": "prefix steam-main does not exist",
    "hint": "Run: portside prefix create steam-main"
  }
}
```

## State Boundary

Default home:

```text
~/Games/portside
```

Override:

```text
PORTSIDE_HOME=/path/to/home portside doctor
```

Important directories:

```text
config.json
prefixes/
profiles/
logs/
snapshots/
state/
cache/
```

The current scaffold stores config and profiles as JSON. Future versions may migrate to YAML, but agents should use CLI commands rather than depending on the file format.

## Steam Game Install Flow

1. Configure GPTK with `portside runner setup gptk`. It auto-registers an installed `gameportingtoolkit` or GPTK cask `wine64` when available. Use `portside runner use gptk --command <path>` for a fixed path.
2. Create a prefix with `portside prefix create steam-main`.
3. Install Windows Steam in that prefix with `portside steam install --prefix steam-main`.
4. Open Windows Steam with `portside steam open --prefix steam-main`.
5. Add the profile with `portside game add <id> --appid <appid> --prefix steam-main`.
6. Use `portside game install <id>` to retrieve the guided install steps.
7. Install the game inside Windows Steam.
8. Launch with `portside run <id>`.

Do not use Mac Steam as the Windows game install manager.

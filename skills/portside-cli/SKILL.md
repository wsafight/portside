---
name: portside-cli
description: Use when operating or troubleshooting the Portside CLI for GPTK-backed Windows Steam game environments on Apple Silicon Mac. Covers safe CLI workflows, JSON/NDJSON protocol usage, PORTSIDE_HOME boundaries, profile/prefix management, logs, runner diagnostics, and when not to edit state directly.
metadata:
  short-description: Operate Portside CLI safely
---

# Portside CLI

Use this skill when a task involves the local `portside` command, Portside profiles, prefixes, logs, snapshots, runner checks, or automation through JSON/NDJSON.

## Core Rules

- Treat `portside` as the authority for state changes.
- Do not directly edit `PORTSIDE_HOME` files unless the user explicitly asks for low-level repair and the CLI cannot do it.
- Use `portside --json <command>` or `<command> --json` for query automation.
- Long-running task commands should be consumed as NDJSON once implemented.
- Never hand-build GPTK/Wine launch commands when `portside run <game>` can express the operation.
- GUI, TUI, helper, scripts, and agents must use the same `PORTSIDE_HOME`.

## First Checks

Run these before making assumptions about a user's Portside installation:

```text
portside --json doctor
portside --json prefix list
portside --json game list
```

`doctor` returns the current required actions by default. Use `portside --json doctor --verbose` only when you need every low-level check.

If `doctor` reports a missing config or state directories, use:

```text
portside init
```

## Common Workflows

Initialize:

```text
portside init
portside doctor
```

Configure an existing GPTK runner:

```text
portside runner setup gptk
portside runner use gptk
portside runner use gptk --command /path/to/wine64 --server-command /path/to/wineserver
```

Use `runner setup gptk` first. It discovers an installed GPTK-compatible runner from `PATH`, `/Applications/Game Porting Toolkit.app`, or Homebrew Caskroom and writes it to config. Use `runner use gptk` when you explicitly want to persist a known command path. Use `runner import gptk --file <path>` only to copy and register a local Apple official runtime package for later dmg/pkg import work.

Create a prefix:

```text
portside prefix create steam-main
```

Install/open Windows Steam in the prefix:

```text
portside steam install --prefix steam-main
portside steam open --prefix steam-main
```

Add a Steam game profile:

```text
portside game add elden-ring --appid 1245620 --prefix steam-main
portside game install elden-ring
```

Inspect a launch plan without starting a process:

```text
portside --json run elden-ring --dry-run
```

Open the terminal UI:

```text
portside tui
```

## References

- For command shapes and JSON contracts, read `references/cli.md`.
- For diagnostics and recovery order, read `references/troubleshooting.md`.

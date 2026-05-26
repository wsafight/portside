# Portside Troubleshooting

## Diagnostic Order

1. Run `portside --json doctor` and follow the returned `needs[].actions`.
2. Use `portside --json doctor --verbose` only if the summary is not enough.
3. List prefixes with `portside --json prefix list`.
4. List profiles with `portside --json game list`.
5. Inspect a profile with `portside --json game show <id>`.
6. Use `portside --json run <id> --dry-run` to inspect the command spec.
7. Read logs through `portside logs <id>` or the returned log path.

## Common Findings

Missing config:

```text
portside init
```

Missing prefix:

```text
portside prefix create steam-main
```

Profile references the wrong prefix:

- Prefer creating a corrected profile through CLI once edit commands exist.
- For early scaffolds, ask before manually repairing profile files.

GPTK runner missing:

- Do not install or download GPTK automatically.
- Report that Portside only detects and calls the user's runner.
- For normal users, point them to a complete prebuilt runner first: `brew tap gcenx/wine`, then `brew install --cask gcenx/wine/game-porting-toolkit`.
- Explain that Apple's GPTK 3 `Evaluation environment for Windows games` is a redist/runtime component, not a complete runner by itself.
- After the prebuilt runner is installed, use `portside runner setup gptk`.
- Use `portside runner import gptk --file <path>` to copy and register the local Apple runtime package for later import work.
- Rerun `portside doctor`.

Steam game not installed:

- Confirm the profile prefix with `portside --json game show <id>`.
- Open Windows Steam in that prefix: `portside steam open --prefix <prefix>`.
- Use `portside game install <id>` for guided steps.
- The user may need to sign in, choose a library, accept EULA prompts, and let Steam install dependencies.

GUI/CLI mismatch:

- Check both use the same `PORTSIDE_HOME`.
- GUI should call its bundled helper.
- CLI shim should resolve to the current app helper when installed by GUI.

## Things To Avoid

- Do not parse human CLI output in automation when JSON is available.
- Do not bypass `portside run` by constructing Wine/GPTK shell commands.
- Do not mutate prefix internals during a running game.
- Do not delete prefixes or snapshots without explicit user intent.

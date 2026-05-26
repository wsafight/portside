package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"portside/core/doctor"
	"portside/core/porthome"
	"portside/core/profiles"
	"portside/core/response"
	"portside/core/runner"
	"portside/core/update"
	"portside/internal/tui"
)

const version = "0.0.0-dev"

type globalOptions struct {
	JSON bool
}

func main() {
	if err := mainErr(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

func mainErr(args []string, stdout, stderr io.Writer) error {
	if err := run(args, stdout, stderr); err != nil {
		global, _ := parseGlobalOptions(args)
		if global.JSON {
			_ = response.WriteError(stdout, "command_failed", err.Error(), "")
		} else {
			fmt.Fprintln(stderr, err)
		}
		return err
	}
	return nil
}

func run(args []string, stdout, stderr io.Writer) error {
	global, args := parseGlobalOptions(args)
	if len(args) == 0 {
		printUsage(stderr)
		return fmt.Errorf("no command provided")
	}

	switch args[0] {
	case "help", "--help", "-h":
		printUsage(stdout)
		return nil
	case "version":
		fmt.Fprintln(stdout, version)
		return nil
	case "init":
		return runInit(args[1:], stdout, global)
	case "doctor":
		return runDoctor(args[1:], stdout, global)
	case "prefix":
		return runPrefix(args[1:], stdout, global)
	case "runner":
		return runRunner(args[1:], stdout, global)
	case "game":
		return runGame(args[1:], stdout, global)
	case "steam":
		return runSteam(args[1:], stdout, global)
	case "run":
		return runGameLaunch(args[1:], stdout, global)
	case "logs":
		return runLogs(args[1:], stdout, global)
	case "update":
		return runUpdate(args[1:], stdout, global)
	case "tui":
		return tui.Run(stdout)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runInit(args []string, stdout io.Writer, global globalOptions) error {
	flags := newFlagSet("init")
	jsonOut := flags.Bool("json", false, "write JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("init does not accept positional arguments")
	}

	result, err := porthome.Init()
	if err != nil {
		return err
	}
	if global.JSON || *jsonOut {
		return response.WriteJSON(stdout, result)
	}

	fmt.Fprintf(stdout, "Portside home: %s\n", result.Home)
	for _, path := range result.Created {
		fmt.Fprintf(stdout, "created %s\n", path)
	}
	for _, path := range result.Existing {
		fmt.Fprintf(stdout, "exists %s\n", path)
	}
	return nil
}

func runDoctor(args []string, stdout io.Writer, global globalOptions) error {
	flags := newFlagSet("doctor")
	jsonOut := flags.Bool("json", false, "write JSON output")
	verbose := flags.Bool("verbose", false, "show every check")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("doctor does not accept positional arguments")
	}

	report, err := doctor.Run()
	if err != nil {
		return err
	}
	summary := doctor.Summarize(report)
	if global.JSON || *jsonOut {
		if *verbose {
			return response.WriteJSON(stdout, report)
		}
		return response.WriteJSON(stdout, summary)
	}

	if !*verbose {
		printDoctorSummary(stdout, summary)
		return nil
	}

	fmt.Fprintf(stdout, "home: %s\n", report.Home)
	printCheck(stdout, report.Config)
	printCheck(stdout, report.MacOS)
	printCheck(stdout, report.Arch)
	printCheck(stdout, report.Rosetta)
	printCheck(stdout, report.GPTKRunner)
	printCheck(stdout, report.GPTKNoHUDRunner)
	printCheck(stdout, report.GPTKRuntime)
	for _, check := range report.Directories {
		printCheck(stdout, check)
	}
	return nil
}

func printDoctorSummary(out io.Writer, summary doctor.Summary) {
	fmt.Fprintf(out, "home: %s\n", summary.Home)
	if summary.Status == "ok" {
		fmt.Fprintln(out, "status: ok")
		fmt.Fprintln(out, "当前没有需要处理的事项。")
		fmt.Fprintln(out, "查看完整检查: portside doctor --verbose")
		return
	}

	fmt.Fprintln(out, "status: needs_action")
	for i, need := range summary.Needs {
		fmt.Fprintf(out, "\n%d. %s\n", i+1, need.Title)
		if need.Message != "" {
			fmt.Fprintf(out, "   %s\n", need.Message)
		}
		for _, action := range need.Actions {
			fmt.Fprintf(out, "   - %s\n", action)
		}
	}
	fmt.Fprintln(out, "\n查看完整检查: portside doctor --verbose")
}

func runPrefix(args []string, stdout io.Writer, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("prefix command required: create, list")
	}
	switch args[0] {
	case "create":
		flags := newFlagSet("prefix create")
		jsonOut := flags.Bool("json", false, "write JSON output")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside prefix create <id>")
		}
		prefix, err := profiles.CreatePrefix(positional[0])
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, prefix)
		}
		fmt.Fprintf(stdout, "created prefix %s at %s\n", prefix.ID, prefix.Path)
		return nil
	case "list":
		flags := newFlagSet("prefix list")
		jsonOut := flags.Bool("json", false, "write JSON output")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("prefix list does not accept positional arguments")
		}
		prefixes, err := profiles.ListPrefixes()
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, prefixes)
		}
		if len(prefixes) == 0 {
			fmt.Fprintln(stdout, "no prefixes")
			return nil
		}
		for _, prefix := range prefixes {
			fmt.Fprintf(stdout, "%s\t%s\n", prefix.ID, prefix.Path)
		}
		return nil
	default:
		return fmt.Errorf("unknown prefix command: %s", args[0])
	}
}

func runGame(args []string, stdout io.Writer, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("game command required: add, list, show, install")
	}
	switch args[0] {
	case "add":
		flags := newFlagSet("game add")
		jsonOut := flags.Bool("json", false, "write JSON output")
		appID := flags.Int("appid", 0, "Steam AppID")
		prefix := flags.String("prefix", "", "prefix id")
		name := flags.String("name", "", "display name")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside game add <id> --appid <appid> --prefix <prefix>")
		}
		profile, err := profiles.AddProfile(profiles.AddOptions{
			ID:     positional[0],
			Name:   *name,
			AppID:  *appID,
			Prefix: *prefix,
		})
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, profile)
		}
		fmt.Fprintf(stdout, "added game %s (%d) using prefix %s\n", profile.ID, profile.AppID, profile.Prefix)
		return nil
	case "list":
		flags := newFlagSet("game list")
		jsonOut := flags.Bool("json", false, "write JSON output")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("game list does not accept positional arguments")
		}
		list, err := profiles.ListProfiles()
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, list)
		}
		if len(list) == 0 {
			fmt.Fprintln(stdout, "no games")
			return nil
		}
		for _, profile := range list {
			fmt.Fprintf(stdout, "%s\t%d\t%s\t%s\n", profile.ID, profile.AppID, profile.Prefix, profile.Name)
		}
		return nil
	case "show":
		flags := newFlagSet("game show")
		jsonOut := flags.Bool("json", false, "write JSON output")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside game show <id>")
		}
		profile, err := profiles.ReadProfile(positional[0])
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, profile)
		}
		fmt.Fprintf(stdout, "id: %s\nname: %s\nappid: %d\nprefix: %s\nlauncher: %s\n", profile.ID, profile.Name, profile.AppID, profile.Prefix, profile.Launcher)
		fmt.Fprintf(stdout, "run: %s %s\n", profile.Run.EXE, strings.Join(profile.Run.Args, " "))
		return nil
	case "install":
		flags := newFlagSet("game install")
		jsonOut := flags.Bool("json", false, "write JSON output")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside game install <id>")
		}
		profile, err := profiles.ReadProfile(positional[0])
		if err != nil {
			return err
		}
		data := map[string]any{
			"profile": profile.ID,
			"appid":   profile.AppID,
			"prefix":  profile.Prefix,
			"status":  "manual_steam_required",
			"message": "Open Windows Steam in the same prefix, sign in, install the game there, then use portside run.",
			"commands": []string{
				"portside steam open --prefix " + profile.Prefix,
				fmt.Sprintf("steam://install/%d", profile.AppID),
				"portside run " + profile.ID,
			},
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, data)
		}
		fmt.Fprintf(stdout, "Install %s inside Windows Steam for prefix %s.\n", profile.Name, profile.Prefix)
		fmt.Fprintf(stdout, "1. portside steam open --prefix %s\n", profile.Prefix)
		fmt.Fprintf(stdout, "2. In Windows Steam, install AppID %d.\n", profile.AppID)
		fmt.Fprintf(stdout, "3. portside run %s\n", profile.ID)
		return nil
	default:
		return fmt.Errorf("unknown game command: %s", args[0])
	}
}

func runRunner(args []string, stdout io.Writer, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("runner command required: list, doctor, setup, use, import")
	}
	switch args[0] {
	case "list", "doctor":
		flags := newFlagSet("runner " + args[0])
		jsonOut := flags.Bool("json", false, "write JSON output")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("runner %s does not accept positional arguments", args[0])
		}
		list, err := runner.List()
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, list)
		}
		if len(list) == 0 {
			fmt.Fprintln(stdout, "no runners configured")
			return nil
		}
		for _, item := range list {
			if item.Command == "" {
				fmt.Fprintf(stdout, "%s\t%s\t%s\n", item.Name, item.Status, item.Message)
				continue
			}
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", item.Name, item.Status, item.Command)
		}
		return nil
	case "setup":
		flags := newFlagSet("runner setup")
		jsonOut := flags.Bool("json", false, "write JSON output")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside runner setup gptk")
		}
		if positional[0] != "gptk" {
			return fmt.Errorf("unsupported runner: %s", positional[0])
		}
		plan, err := runner.SetupGPTK()
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, plan)
		}
		fmt.Fprintf(stdout, "GPTK setup status: %s\n%s\n", plan.Status, plan.Message)
		if plan.Configured != nil {
			fmt.Fprintf(stdout, "configured runner %s: %s\n", plan.Configured.Name, plan.Configured.Command)
			if plan.Configured.ServerCommand != "" {
				fmt.Fprintf(stdout, "configured wineserver: %s\n", plan.Configured.ServerCommand)
			}
		}
		for _, step := range plan.PlannedSteps {
			fmt.Fprintf(stdout, "- %s\n", step)
		}
		for _, command := range plan.NextCommands {
			fmt.Fprintf(stdout, "next: %s\n", command)
		}
		return nil
	case "use":
		flags := newFlagSet("runner use")
		jsonOut := flags.Bool("json", false, "write JSON output")
		command := flags.String("command", "", "runner command path")
		noHUDCommand := flags.String("no-hud-command", "", "no-HUD runner command path")
		serverCommand := flags.String("server-command", "", "runner server command path")
		version := flags.String("version", "", "runner version")
		source := flags.String("source", "", "runner source")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside runner use <name> [--command <path>]")
		}
		item, err := runner.Use(runner.UseOptions{
			Name:          positional[0],
			Command:       *command,
			NoHUDCommand:  *noHUDCommand,
			ServerCommand: *serverCommand,
			Version:       *version,
			Source:        *source,
		})
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, item)
		}
		fmt.Fprintf(stdout, "configured runner %s: %s\n", item.Name, item.Command)
		return nil
	case "import":
		flags := newFlagSet("runner import")
		jsonOut := flags.Bool("json", false, "write JSON output")
		file := flags.String("file", "", "official local GPTK dmg/pkg/zip")
		provider := flags.String("provider", "official-file", "package provider")
		positional, err := parseMixedFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if len(positional) != 1 {
			return fmt.Errorf("usage: portside runner import gptk [--file <dmg-or-pkg>]")
		}
		if positional[0] != "gptk" {
			return fmt.Errorf("unsupported runner: %s", positional[0])
		}
		if *provider != "official-file" {
			return fmt.Errorf("unsupported GPTK package provider: %s", *provider)
		}
		plan, err := runner.ImportGPTKPackagePlan(*file)
		if err != nil {
			return err
		}
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, plan)
		}
		fmt.Fprintf(stdout, "GPTK package import status: %s\n%s\n", plan.Status, plan.Message)
		if plan.Configured != nil {
			fmt.Fprintf(stdout, "configured runner %s: %s\n", plan.Configured.Name, plan.Configured.Command)
		}
		for _, step := range plan.PlannedSteps {
			fmt.Fprintf(stdout, "- %s\n", step)
		}
		for _, command := range plan.NextCommands {
			fmt.Fprintf(stdout, "next: %s\n", command)
		}
		return nil
	default:
		return fmt.Errorf("unknown runner command: %s", args[0])
	}
}

func runSteam(args []string, stdout io.Writer, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("steam command required: install, open")
	}
	if args[0] != "install" && args[0] != "open" {
		return fmt.Errorf("unknown steam command: %s", args[0])
	}
	flags := newFlagSet("steam " + args[0])
	jsonOut := flags.Bool("json", false, "write JSON output")
	prefix := flags.String("prefix", "", "prefix id")
	positional, err := parseMixedFlags(flags, args[1:])
	if err != nil {
		return err
	}
	if len(positional) != 0 {
		return fmt.Errorf("steam %s does not accept positional arguments", args[0])
	}
	if *prefix == "" {
		return fmt.Errorf("--prefix is required")
	}

	data := map[string]any{
		"command": args[0],
		"prefix":  *prefix,
		"status":  "runner_required",
		"message": "Windows Steam must be installed and opened inside this prefix through the configured runner.",
	}
	if args[0] == "install" {
		data["steps"] = []string{
			"Download SteamSetup.exe from the official Steam website.",
			"Run SteamSetup.exe inside the selected prefix with the configured runner.",
			"Sign in to Windows Steam and keep this prefix as the game's Steam environment.",
		}
		data["next_commands"] = []string{
			"portside runner doctor",
			"portside steam open --prefix " + *prefix,
			"portside game install <game>",
		}
	} else {
		data["next_commands"] = []string{
			"portside game install <game>",
			"portside run <game>",
		}
	}
	if global.JSON || *jsonOut {
		return response.WriteJSON(stdout, data)
	}
	fmt.Fprintf(stdout, "steam %s for prefix %s is not implemented yet\n", args[0], *prefix)
	if args[0] == "install" {
		fmt.Fprintln(stdout, "Download SteamSetup.exe from Steam, then run it inside this prefix once the runner adapter is implemented.")
	} else {
		fmt.Fprintln(stdout, "Open Windows Steam in this prefix, install the game, then run the profile.")
	}
	return nil
}

func runGameLaunch(args []string, stdout io.Writer, global globalOptions) error {
	flags := newFlagSet("run")
	jsonOut := flags.Bool("json", false, "write JSON output")
	dryRun := flags.Bool("dry-run", false, "print command spec without launching")
	positional, err := parseMixedFlags(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("usage: portside run <game> [--dry-run]")
	}
	profile, err := profiles.ReadProfile(positional[0])
	if err != nil {
		return err
	}

	spec := map[string]any{
		"profile": profile.ID,
		"prefix":  profile.Prefix,
		"runner":  "gptk",
		"exe":     profile.Run.EXE,
		"args":    profile.Run.Args,
		"cwd":     profile.Run.CWD,
		"env":     profile.Env,
		"status":  "dry_run",
	}
	if !*dryRun {
		spec["status"] = "not_implemented"
		spec["message"] = "process launch is reserved for the runner adapter"
	}
	if global.JSON || *jsonOut {
		return response.WriteJSON(stdout, spec)
	}
	fmt.Fprintf(stdout, "profile: %s\nprefix: %s\nrunner: gptk\nexe: %s\nargs: %s\nstatus: %s\n", profile.ID, profile.Prefix, profile.Run.EXE, strings.Join(profile.Run.Args, " "), spec["status"])
	return nil
}

func runLogs(args []string, stdout io.Writer, global globalOptions) error {
	flags := newFlagSet("logs")
	jsonOut := flags.Bool("json", false, "write JSON output")
	positional, err := parseMixedFlags(flags, args)
	if err != nil {
		return err
	}
	if len(positional) > 1 {
		return fmt.Errorf("usage: portside logs [game]")
	}
	home, err := porthome.Resolve()
	if err != nil {
		return err
	}
	target := porthome.LogsDir(home.Path)
	if len(positional) == 1 {
		target += string(os.PathSeparator) + positional[0]
	}
	data := map[string]string{"path": target}
	if global.JSON || *jsonOut {
		return response.WriteJSON(stdout, data)
	}
	fmt.Fprintln(stdout, target)
	return nil
}

func runUpdate(args []string, stdout io.Writer, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("update command required: check")
	}
	switch args[0] {
	case "check":
		flags := newFlagSet("update check")
		jsonOut := flags.Bool("json", false, "write JSON output")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("update check does not accept positional arguments")
		}
		result := update.Check(version)
		if global.JSON || *jsonOut {
			return response.WriteJSON(stdout, result)
		}
		fmt.Fprintf(stdout, "current version: %s\nchannel: %s\nsource: %s\nstatus: %s\n%s\n", result.CurrentVersion, result.Channel, result.Source, result.Status, result.Message)
		return nil
	default:
		return fmt.Errorf("unknown update command: %s", args[0])
	}
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  portside <command> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Common commands:")
	fmt.Fprintln(out, "  portside doctor")
	fmt.Fprintln(out, "  portside init")
	fmt.Fprintln(out, "  portside runner doctor")
	fmt.Fprintln(out, "  portside runner setup gptk")
	fmt.Fprintln(out, "  portside runner import gptk --file ~/Downloads/Game_Porting_Toolkit_3.0.dmg")
	fmt.Fprintln(out, "  portside prefix create steam-main")
	fmt.Fprintln(out, "  portside steam install --prefix steam-main")
	fmt.Fprintln(out, "  portside game list")
	fmt.Fprintln(out, "  portside run <game>")
	fmt.Fprintln(out, "  portside tui")
}

func parseGlobalOptions(args []string) (globalOptions, []string) {
	var global globalOptions
	var rest []string
	for _, arg := range args {
		switch arg {
		case "--json":
			global.JSON = true
		default:
			rest = append(rest, arg)
		}
	}
	return global, rest
}

func printCheck(out io.Writer, check doctor.Check) {
	if check.Path != "" && check.Message != "" {
		fmt.Fprintf(out, "%s: %s (%s) %s\n", check.Name, check.Status, check.Path, check.Message)
		return
	}
	if check.Path != "" {
		fmt.Fprintf(out, "%s: %s (%s)\n", check.Name, check.Status, check.Path)
		return
	}
	if check.Message != "" {
		fmt.Fprintf(out, "%s: %s (%s)\n", check.Name, check.Status, check.Message)
		return
	}
	fmt.Fprintf(out, "%s: %s\n", check.Name, check.Status)
}

func newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func parseMixedFlags(flags *flag.FlagSet, args []string) ([]string, error) {
	var flagArgs []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") {
				name := strings.TrimLeft(arg, "-")
				if f := flags.Lookup(name); f != nil && !isBoolFlag(f) {
					if i+1 >= len(args) {
						return nil, fmt.Errorf("flag needs an argument: %s", arg)
					}
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
			continue
		}
		positional = append(positional, arg)
	}
	if err := flags.Parse(flagArgs); err != nil {
		return nil, err
	}
	return positional, nil
}

func isBoolFlag(f *flag.Flag) bool {
	getter, ok := f.Value.(flag.Getter)
	if !ok {
		return false
	}
	_, ok = getter.Get().(bool)
	return ok
}

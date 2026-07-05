package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "herdr-launcher: a herdr plugin; run its actions through herdr")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "open":
		launchOpen()
	case "picker":
		runPicker()
	default:
		fmt.Fprintf(os.Stderr, "herdr-launcher: unknown command %q\n", os.Args[1])
		os.Exit(1)
	}
}

// launchOpen is the action entry point. herdr runs it server-side from a
// keybinding. It asks herdr to open the picker as an overlay plugin pane.
func launchOpen() {
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}

	cmd := exec.Command(herdr, "plugin", "pane", "open",
		"--plugin", "nicolegros.herdr-launcher",
		"--entrypoint", "picker",
		"--placement", "overlay",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: could not open picker: %v\n", err)
		os.Exit(1)
	}
}

// runPicker renders the fuzzy picker overlay. It runs inside the pane herdr
// opens for the 'picker' entrypoint (which has a real terminal).
func runPicker() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}

	entries, warnings := scanDirectories(cfg)

	if len(entries) == 0 && len(warnings) == 0 {
		cp, _ := configPath()
		fmt.Fprintf(os.Stderr, "herdr-launcher: no directories found in configured paths\nConfig: %s\n", cp)
		os.Exit(1)
	}

	p := tea.NewProgram(newPickerModel(entries, warnings), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}

	m, ok := result.(pickerModel)
	if !ok || m.chosen == nil {
		// Cancelled — herdr closes the pane when we exit.
		return
	}

	if err := openOrFocusWorkspace(m.chosen.Path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}
}

// openOrFocusWorkspace either focuses an existing workspace whose label matches
// the directory name, or creates a new workspace at the given path. If a new
// workspace is created and cfg.OnCreate is set, the command is fired async with
// context env vars.
func openOrFocusWorkspace(dir string, cfg Config) error {
	client, err := newHerdrClient()
	if err != nil {
		return err
	}

	dirName := filepath.Base(dir)

	// Check for existing workspace with matching label
	workspaces, err := client.workspaceList()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	for _, ws := range workspaces {
		if strings.EqualFold(ws.Label, dirName) {
			return client.workspaceFocus(ws.WorkspaceID)
		}
	}

	// No match — create a new workspace
	wsID, err := client.workspaceCreate(dir, true)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	// Fire on_create command async if configured
	if strings.TrimSpace(cfg.OnCreate) != "" {
		runOnCreate(cfg.OnCreate, dir, dirName, wsID)
	}

	return nil
}

// runOnCreate fires the on_create command asynchronously via sh -c, with
// context passed as environment variables.
func runOnCreate(command, dir, name, workspaceID string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(os.Environ(),
		"HERDR_LAUNCHER_DIR="+dir,
		"HERDR_LAUNCHER_NAME="+name,
		"HERDR_LAUNCHER_WORKSPACE_ID="+workspaceID,
	)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr // captured by herdr plugin log
	cmd.Stderr = os.Stderr
	_ = cmd.Start() // fire and forget
}

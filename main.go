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
	case "switch":
		launchSwitch()
	case "switcher-ui":
		runSwitcherUI()
	case "switch-next":
		switchNext()
	case "switch-previous":
		switchPrevious()
	case "handle-event":
		handleEvent()
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

// launchSwitch is the action entry point for the switcher overlay.
func launchSwitch() {
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}

	cmd := exec.Command(herdr, "plugin", "pane", "open",
		"--plugin", "nicolegros.herdr-launcher",
		"--entrypoint", "switcher",
		"--placement", "overlay",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: could not open switcher: %v\n", err)
		os.Exit(1)
	}
}

// runSwitcherUI renders the workspace history switcher overlay. It reads the
// history state and workspace list, then shows a navigable list.
func runSwitcherUI() {
	state, err := readState()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}

	if len(state.Stack) == 0 {
		// Nothing to show
		return
	}

	// Get workspace labels for display
	client, err := newHerdrClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}

	workspaces, err := client.workspaceList()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}

	// Build a map of workspace ID to label
	labelMap := make(map[string]string, len(workspaces))
	for _, ws := range workspaces {
		labelMap[ws.WorkspaceID] = ws.Label
	}

	// Determine which workspace is currently focused
	var focusedID string
	for _, ws := range workspaces {
		if ws.Focused {
			focusedID = ws.WorkspaceID
			break
		}
	}

	// Build entries from the stack, skipping workspaces that no longer exist
	var entries []switcherEntry
	for _, wsID := range state.Stack {
		label, ok := labelMap[wsID]
		if !ok {
			continue // workspace was closed
		}
		entries = append(entries, switcherEntry{
			WorkspaceID: wsID,
			Label:       label,
			Current:     wsID == focusedID,
		})
	}

	if len(entries) == 0 {
		return
	}

	// Pre-position cursor at index 1 (previous workspace)
	startCursor := 1
	if len(entries) <= 1 {
		startCursor = 0
	}

	p := tea.NewProgram(newSwitcherModel(entries, startCursor), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
		os.Exit(1)
	}

	m, ok := result.(switcherModel)
	if !ok || m.chosen == nil {
		// Cancelled
		return
	}

	// Focus the chosen workspace
	if err := client.workspaceFocus(m.chosen.WorkspaceID); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-launcher:", err)
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

	// Mark entries that have an open workspace
	if client, err := newHerdrClient(); err == nil {
		if workspaces, err := client.workspaceList(); err == nil {
			openLabels := make(map[string]bool, len(workspaces))
			for _, ws := range workspaces {
				openLabels[strings.ToLower(ws.Label)] = true
			}
			for i := range entries {
				if openLabels[strings.ToLower(entries[i].Name)] {
					entries[i].Active = true
				}
			}
		}
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

// runOnCreate fires the on_create command via sh -c, with context passed as
// environment variables. Runs synchronously since the picker pane is torn down
// when this process exits — a detached child would be killed.
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
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: on_create failed: %v\n", err)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// handleEvent is the subcommand dispatched by herdr on workspace.focused and
// workspace.closed events. It reads HERDR_PLUGIN_EVENT and HERDR_PLUGIN_EVENT_JSON
// to update the history state file.
func handleEvent() {
	event := os.Getenv("HERDR_PLUGIN_EVENT")
	payload := os.Getenv("HERDR_PLUGIN_EVENT_JSON")

	switch event {
	case "workspace.focused":
		handleWorkspaceFocused(payload)
	case "workspace.closed":
		handleWorkspaceClosed(payload)
	default:
		fmt.Fprintf(os.Stderr, "herdr-launcher: unknown event %q\n", event)
		os.Exit(1)
	}
}

// focusedEvent is the relevant subset of the workspace.focused event payload.
type focusedEvent struct {
	WorkspaceID string `json:"workspace_id"`
}

func handleWorkspaceFocused(payload string) {
	var ev focusedEvent
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: parse focused event: %v\n", err)
		os.Exit(1)
	}
	if ev.WorkspaceID == "" {
		fmt.Fprintln(os.Stderr, "herdr-launcher: focused event missing workspace_id")
		os.Exit(1)
	}

	if err := withState(func(s *historyState) {
		// Smart cursor detection: if the focused workspace matches what the
		// cursor currently points to, this is a switch-initiated focus - leave
		// the cursor alone.
		if s.Cursor >= 0 && s.Cursor < len(s.Stack) && s.Stack[s.Cursor] == ev.WorkspaceID {
			return
		}
		// Normal navigation: push to front, reset cursor to 0.
		s.Stack = stackPushFront(s.Stack, ev.WorkspaceID)
		s.Cursor = 0
	}); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: update state: %v\n", err)
		os.Exit(1)
	}
}

// closedEvent is the relevant subset of the workspace.closed event payload.
type closedEvent struct {
	WorkspaceID string `json:"workspace_id"`
	Workspace   *struct {
		WorkspaceID string `json:"workspace_id"`
	} `json:"workspace"`
}

func handleWorkspaceClosed(payload string) {
	var ev closedEvent
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: parse closed event: %v\n", err)
		os.Exit(1)
	}

	// workspace.closed may have workspace_id at top level or nested in workspace
	wsID := ev.WorkspaceID
	if wsID == "" && ev.Workspace != nil {
		wsID = ev.Workspace.WorkspaceID
	}
	if wsID == "" {
		// Nothing to remove
		return
	}

	if err := withState(func(s *historyState) {
		idx := stackIndex(s.Stack, wsID)
		if idx < 0 {
			return
		}
		s.Stack = stackRemove(s.Stack, idx)
		// Adjust cursor if it was at or past the removed index
		if s.Cursor >= idx && s.Cursor > 0 {
			s.Cursor--
		}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: update state: %v\n", err)
		os.Exit(1)
	}
}

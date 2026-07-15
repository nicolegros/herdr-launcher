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

// eventEnvelope wraps the event data herdr passes in HERDR_PLUGIN_EVENT_JSON.
type eventEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// focusedEventData is the workspace_focused event data payload.
type focusedEventData struct {
	WorkspaceID string `json:"workspace_id"`
}

func handleWorkspaceFocused(payload string) {
	wsID := extractWorkspaceID(payload)
	if wsID == "" {
		fmt.Fprintln(os.Stderr, "herdr-launcher: focused event missing workspace_id")
		os.Exit(1)
	}

	if err := withState(func(s *historyState) {
		// Smart cursor detection: if the focused workspace matches what the
		// cursor currently points to, this is a switch-initiated focus - leave
		// the cursor alone.
		if s.Cursor >= 0 && s.Cursor < len(s.Stack) && s.Stack[s.Cursor] == wsID {
			return
		}
		// Normal navigation: push to front, reset cursor to 0.
		s.Stack = stackPushFront(s.Stack, wsID)
		s.Cursor = 0
	}); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: update state: %v\n", err)
		os.Exit(1)
	}
}

// closedEventData is the workspace_closed event data payload.
type closedEventData struct {
	WorkspaceID string `json:"workspace_id"`
	Workspace   *struct {
		WorkspaceID string `json:"workspace_id"`
	} `json:"workspace"`
}

func handleWorkspaceClosed(payload string) {
	wsID := extractClosedWorkspaceID(payload)
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

// extractWorkspaceID parses workspace_id from the event payload, handling both
// the envelope format {"event":"...","data":{...}} and flat format {"workspace_id":"..."}.
func extractWorkspaceID(payload string) string {
	// Try envelope format first
	var env eventEnvelope
	if err := json.Unmarshal([]byte(payload), &env); err == nil && len(env.Data) > 0 {
		var data focusedEventData
		if json.Unmarshal(env.Data, &data) == nil && data.WorkspaceID != "" {
			return data.WorkspaceID
		}
	}
	// Fall back to flat format
	var flat focusedEventData
	if json.Unmarshal([]byte(payload), &flat) == nil {
		return flat.WorkspaceID
	}
	return ""
}

// extractClosedWorkspaceID parses workspace_id from a workspace.closed event,
// handling envelope and flat formats, and nested workspace object.
func extractClosedWorkspaceID(payload string) string {
	// Try envelope format
	var env eventEnvelope
	if err := json.Unmarshal([]byte(payload), &env); err == nil && len(env.Data) > 0 {
		var data closedEventData
		if json.Unmarshal(env.Data, &data) == nil {
			if data.WorkspaceID != "" {
				return data.WorkspaceID
			}
			if data.Workspace != nil {
				return data.Workspace.WorkspaceID
			}
		}
	}
	// Fall back to flat format
	var flat closedEventData
	if json.Unmarshal([]byte(payload), &flat) == nil {
		if flat.WorkspaceID != "" {
			return flat.WorkspaceID
		}
		if flat.Workspace != nil {
			return flat.Workspace.WorkspaceID
		}
	}
	return ""
}

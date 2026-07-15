package main

import (
	"fmt"
	"os"
)

// switchNext moves the cursor toward more recent workspaces (cursor - 1) with
// wrap-around, and focuses the target workspace.
func switchNext() {
	if err := doSwitch(-1); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: %v\n", err)
		os.Exit(1)
	}
}

// switchPrevious moves the cursor toward older workspaces (cursor + 1) with
// wrap-around, and focuses the target workspace.
func switchPrevious() {
	if err := doSwitch(1); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-launcher: %v\n", err)
		os.Exit(1)
	}
}

// doSwitch moves the cursor by delta (with wrap), updates the state file, and
// focuses the workspace at the new cursor position.
func doSwitch(delta int) error {
	client, err := newHerdrClient()
	if err != nil {
		return err
	}

	var targetID string

	if err := withState(func(s *historyState) {
		if len(s.Stack) <= 1 {
			// Nothing to switch to
			return
		}
		newCursor := ((s.Cursor + delta) % len(s.Stack) + len(s.Stack)) % len(s.Stack)
		s.Cursor = newCursor
		targetID = s.Stack[newCursor]
	}); err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	if targetID == "" {
		// Silent no-op: stack empty or single entry
		return nil
	}

	return client.workspaceFocus(targetID)
}

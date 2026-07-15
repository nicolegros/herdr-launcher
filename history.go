package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const maxStackSize = 50

// historyState is the persisted workspace navigation history.
type historyState struct {
	Stack  []string `json:"stack"`  // workspace IDs, most recent first
	Cursor int      `json:"cursor"` // current position in stack
}

// stateDir returns the plugin state directory from HERDR_PLUGIN_STATE_DIR.
func stateDir() (string, error) {
	dir := os.Getenv("HERDR_PLUGIN_STATE_DIR")
	if dir == "" {
		return "", fmt.Errorf("HERDR_PLUGIN_STATE_DIR is not set")
	}
	return dir, nil
}

// statePath returns the full path to the history state file.
func statePath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// withState acquires an exclusive lock on the state file, reads the current
// state, calls fn to modify it, and writes the result atomically. If the state
// file does not exist, fn receives a zero-value historyState.
func withState(fn func(*historyState)) error {
	path, err := statePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	// Open or create the lock file (same as state file)
	lockPath := path + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer lockFile.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	// Read current state
	state := historyState{}
	data, err := os.ReadFile(path)
	if err == nil {
		// Ignore decode errors - treat as empty state
		json.Unmarshal(data, &state)
	}

	// Apply modification
	fn(&state)

	// Enforce stack cap
	if len(state.Stack) > maxStackSize {
		state.Stack = state.Stack[:maxStackSize]
	}

	// Clamp cursor
	if len(state.Stack) == 0 {
		state.Cursor = 0
	} else {
		state.Cursor = ((state.Cursor % len(state.Stack)) + len(state.Stack)) % len(state.Stack)
	}

	// Write atomically: temp file + rename
	out, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename state: %w", err)
	}

	return nil
}

// readState reads the current state without modifying it. Still acquires a
// shared lock for consistency.
func readState() (historyState, error) {
	path, err := statePath()
	if err != nil {
		return historyState{}, err
	}

	lockPath := path + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return historyState{}, fmt.Errorf("open lock file: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_SH); err != nil {
		return historyState{}, fmt.Errorf("acquire lock: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	state := historyState{}
	data, err := os.ReadFile(path)
	if err != nil {
		return state, nil // no state file yet
	}
	json.Unmarshal(data, &state)
	return state, nil
}

// stackIndex returns the index of workspaceID in the stack, or -1 if not found.
func stackIndex(stack []string, workspaceID string) int {
	for i, id := range stack {
		if id == workspaceID {
			return i
		}
	}
	return -1
}

// stackRemove removes the element at index i from the stack.
func stackRemove(stack []string, i int) []string {
	return append(stack[:i], stack[i+1:]...)
}

// stackPushFront moves workspaceID to position 0, removing any existing occurrence.
func stackPushFront(stack []string, workspaceID string) []string {
	if idx := stackIndex(stack, workspaceID); idx >= 0 {
		stack = stackRemove(stack, idx)
	}
	return append([]string{workspaceID}, stack...)
}

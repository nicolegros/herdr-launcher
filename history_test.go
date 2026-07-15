package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStackPushFront(t *testing.T) {
	tests := []struct {
		name   string
		stack  []string
		pushID string
		want   []string
	}{
		{"empty stack", nil, "w1", []string{"w1"}},
		{"new entry", []string{"w1", "w2"}, "w3", []string{"w3", "w1", "w2"}},
		{"existing entry moves to front", []string{"w1", "w2", "w3"}, "w2", []string{"w2", "w1", "w3"}},
		{"already at front", []string{"w1", "w2"}, "w1", []string{"w1", "w2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stackPushFront(tt.stack, tt.pushID)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStackRemove(t *testing.T) {
	stack := []string{"w1", "w2", "w3"}

	got := stackRemove(stack, 1)
	want := []string{"w1", "w3"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStackIndex(t *testing.T) {
	stack := []string{"w1", "w2", "w3"}

	if got := stackIndex(stack, "w2"); got != 1 {
		t.Errorf("stackIndex(w2) = %d, want 1", got)
	}
	if got := stackIndex(stack, "w99"); got != -1 {
		t.Errorf("stackIndex(w99) = %d, want -1", got)
	}
}

func TestWithState_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	err := withState(func(s *historyState) {
		s.Stack = []string{"w1"}
		s.Cursor = 0
	})
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "history.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var state historyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Stack) != 1 || state.Stack[0] != "w1" {
		t.Errorf("stack = %v, want [w1]", state.Stack)
	}
}

func TestWithState_EnforcesMaxStackSize(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	err := withState(func(s *historyState) {
		for i := 0; i < 60; i++ {
			s.Stack = append(s.Stack, fmt.Sprintf("w%d", i))
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	state, err := readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Stack) != maxStackSize {
		t.Errorf("stack len = %d, want %d", len(state.Stack), maxStackSize)
	}
}

func TestWithState_ClampsCursor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	err := withState(func(s *historyState) {
		s.Stack = []string{"w1", "w2", "w3"}
		s.Cursor = 10 // out of bounds
	})
	if err != nil {
		t.Fatal(err)
	}

	state, err := readState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Cursor < 0 || state.Cursor >= len(state.Stack) {
		t.Errorf("cursor = %d, should be in [0, %d)", state.Cursor, len(state.Stack))
	}
}

func TestHandleWorkspaceFocused_NormalNavigation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	// Seed state: stack [w1, w2, w3], cursor 0
	seedState(t, dir, historyState{Stack: []string{"w1", "w2", "w3"}, Cursor: 0})

	// Focus a new workspace (normal navigation)
	payload := `{"workspace_id": "w4"}`
	handleWorkspaceFocused(payload)

	state, _ := readState()
	if state.Stack[0] != "w4" {
		t.Errorf("stack[0] = %q, want w4", state.Stack[0])
	}
	if state.Cursor != 0 {
		t.Errorf("cursor = %d, want 0", state.Cursor)
	}
}

func TestHandleWorkspaceFocused_SwitchInitiated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	// Seed state: stack [w1, w2, w3], cursor 1 (pointing to w2)
	seedState(t, dir, historyState{Stack: []string{"w1", "w2", "w3"}, Cursor: 1})

	// Focus w2 (matches cursor position - switch-initiated)
	payload := `{"workspace_id": "w2"}`
	handleWorkspaceFocused(payload)

	state, _ := readState()
	// Stack should be unchanged
	if state.Stack[0] != "w1" || state.Stack[1] != "w2" || state.Stack[2] != "w3" {
		t.Errorf("stack was modified: %v", state.Stack)
	}
	if state.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", state.Cursor)
	}
}

func TestHandleWorkspaceFocused_ExistingMovesToFront(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	// Seed: [w1, w2, w3], cursor 0
	seedState(t, dir, historyState{Stack: []string{"w1", "w2", "w3"}, Cursor: 0})

	// Focus w3 (normal nav - not at cursor position)
	payload := `{"workspace_id": "w3"}`
	handleWorkspaceFocused(payload)

	state, _ := readState()
	if state.Stack[0] != "w3" {
		t.Errorf("stack[0] = %q, want w3", state.Stack[0])
	}
	if len(state.Stack) != 3 {
		t.Errorf("stack len = %d, want 3 (deduplicated)", len(state.Stack))
	}
	if state.Cursor != 0 {
		t.Errorf("cursor = %d, want 0", state.Cursor)
	}
}

func TestHandleWorkspaceClosed_RemovesAndAdjustsCursor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	// Seed: [w1, w2, w3], cursor 2
	seedState(t, dir, historyState{Stack: []string{"w1", "w2", "w3"}, Cursor: 2})

	// Close w2 (index 1, cursor was at 2)
	payload := `{"workspace_id": "w2"}`
	handleWorkspaceClosed(payload)

	state, _ := readState()
	if len(state.Stack) != 2 {
		t.Fatalf("stack len = %d, want 2", len(state.Stack))
	}
	if stackIndex(state.Stack, "w2") != -1 {
		t.Error("w2 should have been removed")
	}
	if state.Cursor != 1 {
		t.Errorf("cursor = %d, want 1 (adjusted)", state.Cursor)
	}
}

func TestHandleWorkspaceClosed_CursorBeforeRemoved(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	// Seed: [w1, w2, w3], cursor 0
	seedState(t, dir, historyState{Stack: []string{"w1", "w2", "w3"}, Cursor: 0})

	// Close w3 (index 2, cursor at 0 - before removed)
	payload := `{"workspace_id": "w3"}`
	handleWorkspaceClosed(payload)

	state, _ := readState()
	if state.Cursor != 0 {
		t.Errorf("cursor = %d, want 0 (unchanged)", state.Cursor)
	}
}

func TestHandleWorkspaceClosed_NotInStack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_STATE_DIR", dir)

	seedState(t, dir, historyState{Stack: []string{"w1", "w2"}, Cursor: 0})

	payload := `{"workspace_id": "w99"}`
	handleWorkspaceClosed(payload)

	state, _ := readState()
	if len(state.Stack) != 2 {
		t.Errorf("stack len = %d, want 2", len(state.Stack))
	}
}

func seedState(t *testing.T, dir string, state historyState) {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "history.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

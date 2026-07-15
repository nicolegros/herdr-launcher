package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
)

// herdrClient talks to the running herdr instance over its unix domain socket.
// Protocol: newline-delimited JSON, one request per line, one response per line.
type herdrClient struct {
	socketPath string
}

// newHerdrClient builds a client from the HERDR_SOCKET_PATH environment variable.
func newHerdrClient() (*herdrClient, error) {
	path := os.Getenv("HERDR_SOCKET_PATH")
	if path == "" {
		return nil, errors.New("HERDR_SOCKET_PATH is not set; are you running inside herdr?")
	}
	return &herdrClient{socketPath: path}, nil
}

type request struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

type herdrError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *herdrError     `json:"error"`
}

// call sends a single request and decodes the result into out.
func (c *herdrClient) call(method string, params map[string]any, out any) error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("connect herdr socket: %w", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(request{ID: "herdr-launcher", Method: method, Params: params}); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	var resp response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("herdr error %s: %s", resp.Error.Code, resp.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

// workspaceInfo is the subset of workspace metadata we use.
type workspaceInfo struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	Focused     bool   `json:"focused"`
}

// workspaceList returns all open workspaces.
func (c *herdrClient) workspaceList() ([]workspaceInfo, error) {
	var out struct {
		Workspaces []workspaceInfo `json:"workspaces"`
	}
	if err := c.call("workspace.list", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out.Workspaces, nil
}

// workspaceCreate makes a new workspace rooted at cwd and returns its ID.
// Label is omitted so herdr auto-derives it from the directory.
func (c *herdrClient) workspaceCreate(cwd string, focus bool) (string, error) {
	var out struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"workspace"`
	}
	err := c.call("workspace.create", map[string]any{
		"cwd":   cwd,
		"focus": focus,
	}, &out)
	if err != nil {
		return "", err
	}
	return out.Workspace.WorkspaceID, nil
}

// workspaceFocus switches to an existing workspace.
func (c *herdrClient) workspaceFocus(workspaceID string) error {
	return c.call("workspace.focus", map[string]any{
		"workspace_id": workspaceID,
	}, nil)
}

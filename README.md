# herdr-launcher

A [herdr](https://herdr.dev) plugin that provides a fuzzy directory picker for quickly creating or switching to workspaces.

Press a keybinding, fuzzy-find a project directory, hit enter — you're in a focused workspace.

## Install

```sh
herdr plugin install nicolegros/herdr-launcher
```

## Configuration

Create a config file at `~/.config/herdr/plugins/config/nicolegros.herdr-launcher/config.toml`:

```toml
# Directories whose immediate children are listed as projects
paths = ["~/Developer", "~/Projects"]

# Individual directories listed directly
projects = ["~/.config", "~/.dotfiles"]

# Shell command to run after creating a new workspace (optional)
# Receives env vars: HERDR_LAUNCHER_DIR, HERDR_LAUNCHER_NAME, HERDR_LAUNCHER_WORKSPACE_ID
on_create = "~/.config/herdr/scripts/setup-workspace.sh"
```

## Keybinding

Add to your `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+f"
type = "plugin_action"
command = "nicolegros.herdr-launcher.open"
description = "project launcher"
```

## Behavior

- **Fuzzy search** across all project names and parent paths
- **Duplicate detection** — if a workspace with a matching label already exists, it focuses that workspace instead of creating a new one
- **on_create hook** — runs a shell command after creating a new workspace, useful for setting up custom layouts (splits, tabs, startup commands)
- **Warnings** — missing configured paths show a warning in the picker but don't block the rest

## Example: on_create layout script

```sh
#!/bin/sh
# ~/.config/herdr/scripts/setup-workspace.sh
# nvim top 80%, shell bottom 20%

ROOT_PANE=$(herdr pane list --workspace "$HERDR_LAUNCHER_WORKSPACE_ID" | jq -r '.result.panes[0].pane_id')
herdr pane split "$ROOT_PANE" --direction down --ratio 0.8 --no-focus
herdr pane run "$ROOT_PANE" nvim
```

## Development

```sh
git clone https://github.com/nicolegros/herdr-launcher
cd herdr-launcher
make build
herdr plugin link .
```

## License

MIT

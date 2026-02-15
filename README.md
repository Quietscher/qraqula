# qraqula

A terminal-based GraphQL client. Query, explore schemas, and manage results — all from your terminal.

Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Query editor with syntax highlighting, auto-formatting, and vim keybindings
- Schema introspection browser with keyboard navigation
- Result viewer with JSON folding, pretty-printing, and virtual scrolling for large responses
- Tab system — each tab is a separate workspace with its own endpoint
- Variables panel with JSON validation against the schema
- Environment support (dev/staging/prod) with per-environment endpoints, headers, and variables
- Query history (25 entries, organized in folders)
- Headers configurable per tab, per folder, or globally
- Query abort — cancel running queries instantly
- cURL export
- Status bar with response metadata (status code, time, size)

## Install

```sh
go install github.com/qraqula/qla@latest
```

## Usage

Launch qraqula:

```sh
qla
```

Set your GraphQL endpoint URL in the top bar, then start writing queries in the editor.

Press `Ctrl+Enter` to execute a query.

## Keybindings

| Key | Action |
|---|---|
| `Ctrl+Enter` | Execute query |
| `Ctrl+h/j/k/l` | Navigate between panels |
| `Ctrl+c` | Abort running query |
| Vim keys | Editor navigation and editing |

## Configuration

Configuration and data are stored at `~/.config/qraqula/` following the XDG spec.

All data is stored in human-readable JSON: queries, history, fragments, environments, and settings.

## License

MIT

## Disclaimer

"GraphQL" is a registered trademark of The GraphQL Foundation. This project is not affiliated with, endorsed by, or associated with The GraphQL Foundation.

qraqula is an independent open source project that implements a client for the [GraphQL specification](https://spec.graphql.org).

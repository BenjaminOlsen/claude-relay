# Claude Relay

Talk to Claude Code from your phone. Voice or text, from anywhere.

Claude Relay is a thin server that runs on your dev machine and exposes Claude Code over a WebSocket. Connect from any browser — say something into your phone's mic, and Claude Code edits your files, runs commands, and talks back.

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated (`claude` in your PATH)
- Chrome or Edge recommended (best Web Speech API support for voice)

## Quick start

```bash
# Build
make build
# or: go build -o claude-relay .

# Run
./claude-relay --token YOUR_SECRET --dir /path/to/your/project
```

Open `http://localhost:8080` on your phone or browser. Enter the token and start talking.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--token` | — | Auth token (required, or set `CLAUDE_RELAY_TOKEN` env var) |
| `--dir` | `.` | Project directory Claude Code works in |
| `--addr` | `:8080` | Listen address |

## Remote access

For access outside your local network:

| Method | Setup |
|--------|-------|
| **Tailscale** | Install on dev machine + phone. Access at `http://<machine>:8080`. No config needed. |
| **ngrok** | `ngrok http 8080`. Use the generated URL. |
| **Cloudflare Tunnel** | `cloudflared tunnel --url http://localhost:8080` |

## How it works

Each message you send spawns a `claude -p --output-format stream-json` process. Session continuity is maintained via Claude Code's `--resume` flag — your full conversation history carries across messages.

The web UI uses the browser's Web Speech API for voice input (STT) and output (TTS). No server-side audio processing, no API keys needed for voice.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design.

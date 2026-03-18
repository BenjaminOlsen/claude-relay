# Claude Relay — Architecture

## What This Is

A thin server that gives you remote voice/text access to Claude Code. Run it on your dev machine, connect from your phone or any browser, and continue working with Claude Code conversationally — hands-free, from anywhere.

This is **not** a new agent framework. It's a relay. Claude Code does all the thinking, tool use, and file editing. This project just bridges the gap between "sitting at a terminal" and "walking around with a phone."

## Core Flow

```
Phone/Browser (anywhere)
  |
  | WebSocket (authenticated)
  |
Claude Relay Server (runs on your dev machine)
  |
  |--- STT: voice audio → text
  |--- TTS: text → voice audio
  |
  | stdin/stdout (stream-json)
  |
Claude Code process
  |
  |--- reads/writes your files
  |--- runs shell commands
  |--- full Claude Code capabilities
  |--- uses your existing OAuth session (Max/Pro plan)
```

## Design Principles

1. **Claude Code is the agent.** We don't reimplement any LLM logic, tool handling, or conversation management. Claude Code already does all of that well.

2. **The relay is stateless.** All conversation state lives in Claude Code's session system. The relay just pipes bytes.

3. **Voice is first-class.** The primary use case is hands-free interaction. Text is supported but voice is the reason this exists.

4. **Runs on your machine.** The server runs where your code lives. It has access to your filesystem because Claude Code needs it. Remote clients connect over the network.

## Components

### 1. Relay Server (Go)

Single binary. Starts an HTTP/WebSocket server and manages Claude Code processes.

```
claude-relay
  ├── main.go              # entry point, config, signal handling
  ├── server.go            # HTTP server, static files, WebSocket upgrade
  ├── session.go           # Claude Code process lifecycle
  ├── relay.go             # WebSocket ↔ Claude Code stdin/stdout bridge
  ├── voice.go             # STT/TTS coordination
  └── auth.go              # connection authentication
```

**Responsibilities:**
- Serve the web UI (single HTML file)
- Authenticate WebSocket connections (token-based)
- Spawn or resume Claude Code processes
- Relay messages between WebSocket and Claude Code's stream-json interface
- Coordinate STT/TTS for voice sessions

### 2. Claude Code Interface

Claude Code supports structured streaming I/O:

```bash
claude -p \
  --input-format stream-json \
  --output-format stream-json \
  --resume <session-id> \
  --add-dir /path/to/project
```

**Input (stdin):** JSON messages, one per line
```json
{"type": "user_message", "content": "refactor the auth module to use JWT"}
```

**Output (stdout):** JSON events, streamed
```json
{"type": "text_delta", "content": "I'll refactor..."}
{"type": "tool_use", "tool": "Edit", "file": "auth.go", ...}
{"type": "result", "content": "Done. I've updated..."}
```

The relay reads these JSON events and forwards them over the WebSocket. No parsing or interpretation needed — just pipe them through.

**Session continuity:** Claude Code's `--resume` and `--continue` flags let you pick up existing sessions. The relay tracks which session ID maps to which project directory.

### 3. Voice Layer

**Speech-to-Text (STT):**
- **Option A: Browser-side** — Web Speech API (free, Chrome/Edge, no server cost). Audio never leaves the client. The browser sends transcribed text over the WebSocket.
- **Option B: Server-side** — Whisper API or local Whisper model. Browser sends raw audio chunks, server transcribes. More accurate, works in all browsers, but costs money or CPU.

Start with Option A (browser-side). Fall back to Option B if accuracy is a problem.

**Text-to-Speech (TTS):**
- **Option A: Browser-side** — Web Speech API `speechSynthesis`. Free, instant, works offline. Quality is acceptable.
- **Option B: Server-side** — OpenAI TTS API or ElevenLabs. Much higher quality voices. Browser receives audio chunks to play.

Start with Option A. Upgrade to Option B later if desired.

**Voice Protocol:**
```
Browser                          Server
  |                                |
  |  [mic audio] → Web Speech API |
  |  transcribed text ───────────→ |
  |                                | → Claude Code stdin
  |                                | ← Claude Code stdout
  |  ←─────────── assistant text   |
  |  Web Speech API → [speaker]    |
  |                                |
```

For server-side STT/TTS, audio binary frames go over the same WebSocket alongside JSON text frames.

### 4. Web UI

Single HTML file. Minimal. Three things:

1. **Mic button** — hold to talk (push-to-talk) or toggle (hands-free mode)
2. **Text input** — fallback for typing
3. **Transcript** — scrolling conversation log showing both sides

No framework. Vanilla HTML/JS/CSS. The UI should work on a phone screen in portrait mode.

Optional later: session picker (list existing Claude Code sessions to resume).

### 5. Authentication

The relay exposes Claude Code to the network. Auth is mandatory.

**Approach:** Pre-shared token. Set `CLAUDE_RELAY_TOKEN` env var. Clients pass it as a query param or header on the WebSocket handshake. Simple, sufficient for a single-user tool.

```
wss://your-machine:8443/ws?token=<your-token>
```

**TLS:** The server should support HTTPS/WSS. For local network use, a self-signed cert works. For remote access over the internet, use a tunnel (Tailscale, Cloudflare Tunnel, ngrok) or a real cert.

## Session Management

```
┌─────────────────────────────────────┐
│ Sessions                            │
│                                     │
│  session-abc  ← /home/ben/myproject │
│    └── claude process (PID 1234)    │
│                                     │
│  session-def  ← /home/ben/other     │
│    └── claude process (PID 5678)    │
│                                     │
│  session-ghi  ← (no process)       │
│    └── resumable, last used 2h ago  │
└─────────────────────────────────────┘
```

- **Create:** Client sends `{"action": "new", "dir": "/path/to/project"}`. Server spawns `claude -p --output-format stream-json --input-format stream-json` in that directory.
- **Resume:** Client sends `{"action": "resume", "session_id": "abc"}`. Server spawns claude with `--resume abc`.
- **List:** Client sends `{"action": "list"}`. Server returns known sessions (could also shell out to `claude --resume` to list Claude Code's own sessions).
- **Detach:** Client disconnects. Claude Code process stays alive (or is killed after a timeout). Client can reconnect and resume later.

## Network Access

For accessing from outside your local network:

| Method | Complexity | Security |
|--------|-----------|----------|
| **Tailscale** | Low | High — WireGuard VPN, no exposed ports |
| **Cloudflare Tunnel** | Low | High — no exposed ports, Cloudflare auth |
| **ngrok** | Lowest | Medium — public URL with token auth |
| **Port forward + TLS** | Medium | Medium — exposed port, need real cert |

Recommendation: **Tailscale**. Install on your dev machine and phone. Access the relay at `http://your-machine:8080` over the Tailscale network. No certs, no port forwarding, encrypted by default.

## What's NOT in Scope

- **Container isolation** — Claude Code runs natively on the host. If you want sandboxing, run the whole relay inside a container with your project mounted.
- **Multi-user** — this is a single-user tool. One person, one machine.
- **Custom agent logic** — Claude Code is the agent. We don't intercept or modify its behavior.
- **Persistent server** — this is a dev tool, not a production service. Start it when you need it.

## Tech Stack

- **Language:** Go (single binary, easy cross-compile, good WebSocket support)
- **Dependencies:** minimal — `gorilla/websocket` or `nhooyr.io/websocket`, `os/exec` for process management
- **Claude Code:** must be installed on the host machine and logged in (`claude` command available in PATH)
- **Browser:** Chrome or Edge recommended (best Web Speech API support)

## File Layout

```
claude-relay/
├── main.go                 # CLI flags, config, startup
├── server.go               # HTTP server, routes, static file serving
├── session.go              # Claude Code process spawn/resume/kill
├── relay.go                # WebSocket ↔ process stdio bridge
├── voice.go                # STT/TTS coordination (if server-side)
├── auth.go                 # Token validation middleware
├── static/
│   └── index.html          # Single-page voice/text UI
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## MVP Scope

Get this working end-to-end:

1. `claude-relay --dir /path/to/project --token mysecret`
2. Open `http://localhost:8080` in Chrome on your phone
3. Hold mic button, say "what files are in this project?"
4. Hear Claude Code's response read back to you
5. See the transcript on screen

That's it. Everything else is polish.

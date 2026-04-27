# crush-a2a

A2A v1.0 protocol frontend that bridges A2A clients to a Crush ACP backend.

## Architecture

```
A2A v1.0 Client  --(JSON-RPC 2.0 over HTTP)--> crush-a2a --(ACP HTTP REST)--> crush serve
```

crush-a2a exposes the [A2A v1.0 protocol](https://google.github.io/A2A/) and translates requests to the Crush ACP backend.

## Supported A2A Methods

| Method | Description |
|--------|-------------|
| `GET /.well-known/agent-card.json` | Agent discovery |
| `SendMessage` | Synchronous message exchange |
| `SendStreamingMessage` | SSE streaming response |
| `GetTask` | Retrieve task by ID |
| `CancelTask` | Cancel a running task |
| `ListTasks` | List all tasks |

## Usage

```bash
# Build
make build

# Run (defaults: port 8200, ACP at localhost:8199)
./crush-a2a --port 8200 --acp-url http://localhost:8199 --agent-name crush
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8200` | HTTP listen port |
| `--acp-url` | `http://localhost:8199` | Crush ACP backend URL |
| `--agent-name` | `crush` | ACP agent name to proxy |

## Docker

```bash
docker build -t crush-a2a .
docker run -p 8200:8200 crush-a2a --acp-url http://host.docker.internal:8199
```

## Quick Test

```bash
# Agent card discovery
curl http://localhost:8200/.well-known/agent-card.json

# Send a message
curl -X POST http://localhost:8200/ \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"kind":"message","messageId":"m1","role":"user","parts":[{"kind":"text","text":"Hello"}]}}}'
```

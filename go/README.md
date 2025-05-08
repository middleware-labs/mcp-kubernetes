# MCP Kubernetes Go Module

This directory contains the Go implementation for the [Azure MCP Kubernetes](https://github.com/Azure/mcp-kubernetes) project.

## Project Structure

- `cmd/`: Application entry points
- `internal/`: Private libraries specific to this project

## Getting Started

```bash
# Build the project
go build -o mcp-kubernetes ./cmd/mcp-kubernetes

# Run the project
./mcp-kubernetes

# Test with an example client
cd examples
go run test-client.go
```

## Debugging

If you're seeing parse errors, ensure your client is sending valid JSON-RPC 2.0 requests. The server expects messages in this format:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "callTool",
  "params": {
    "name": "Run-kubectl-command",
    "arguments": {
      "command": "version"
    }
  }
}
```

Each request should be on a single line, terminated with a newline character.

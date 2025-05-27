package tools

import (
	"context"
	"fmt"

	"github.com/Azure/mcp-kubernetes/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandlerFunc is a function that processes tool requests
// Deprecated: Use CommandExecutorFunc instead
type ToolHandlerFunc func(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error)

// CreateToolHandler creates an adapter that converts CommandExecutor to the format expected by MCP server
func CreateToolHandler(executor CommandExecutor, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map[string]interface{}, got " + fmt.Sprintf("%T", req.Params.Arguments)), nil
		}
		result, err := executor.Execute(args, cfg)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

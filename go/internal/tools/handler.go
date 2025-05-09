package tools

import (
	"context"

	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandlerFunc is a function that processes tool requests
// Deprecated: Use CommandExecutorFunc instead
type ToolHandlerFunc func(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error)

// CreateToolHandler creates an adapter that converts CommandExecutor to the format expected by MCP server
func CreateToolHandler(executor CommandExecutor, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := executor.Execute(req.Params.Arguments, cfg)
		if err != nil {
			return nil, err
		}

		// Convert result to mcp.CallToolResult
		if resultMap, ok := result.(map[string]interface{}); ok {
			if errMsg, ok := resultMap["error"]; ok && errMsg != nil {
				return mcp.NewToolResultError(errMsg.(string)), nil
			}
			if text, ok := resultMap["text"]; ok && text != nil {
				return mcp.NewToolResultText(text.(string)), nil
			}
		}
		return mcp.NewToolResultText(""), nil
	}
}

package helm

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterHelm registers the helm tool
func RegisterHelm() mcp.Tool {
	return mcp.NewTool("Run-helm-command",
		mcp.WithDescription("Run helm command and get result, The command should start with helm"),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("The helm command to execute"),
		),
	)
}
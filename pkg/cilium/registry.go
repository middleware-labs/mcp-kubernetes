package cilium

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterCilium registers the cilium tool
func RegisterCilium() mcp.Tool {
	return mcp.NewTool("Run-cilium-command",
		mcp.WithDescription("Run cilium command and get result, The command should start with cilium"),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("The cilium command to execute"),
		),
	)
}

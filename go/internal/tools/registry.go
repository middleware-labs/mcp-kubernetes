package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterKubectl registers the kubectl tool
func RegisterKubectl() mcp.Tool {
	return mcp.NewTool("Run-kubectl-command",
		mcp.WithDescription("Run kubectl command and get result"),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("The kubectl command to execute"),
		),
	)
}

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



package server

import (
	"fmt"
	"log"

	"github.com/Azure/mcp-kubernetes/go/internal/cilium"
	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/Azure/mcp-kubernetes/go/internal/helm"
	"github.com/Azure/mcp-kubernetes/go/internal/kubectl"
	"github.com/Azure/mcp-kubernetes/go/internal/tools"
	"github.com/Azure/mcp-kubernetes/go/internal/version"
	"github.com/mark3labs/mcp-go/server"
)

// Service represents the MCP Kubernetes service
type Service struct {
	cfg      *config.ConfigData
	mcpServer *server.MCPServer
}

// NewService creates a new MCP Kubernetes service
func NewService(cfg *config.ConfigData) *Service {
	return &Service{
		cfg: cfg,
	}
}

// Initialize initializes the service
func (s *Service) Initialize() error {
	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"MCP Kubernetes",
		version.GetVersion(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Register kubectl tool
	kubectlTool := tools.RegisterKubectl()
	s.mcpServer.AddTool(kubectlTool, tools.CreateToolHandler(kubectl.NewExecutor(), s.cfg))

	// Register additional tools
	if s.cfg.AdditionalTools["helm"] {
		helmTool := tools.RegisterHelm()
		s.mcpServer.AddTool(helmTool, tools.CreateToolHandler(helm.NewExecutor(), s.cfg))
	}

	if s.cfg.AdditionalTools["cilium"] {
		ciliumTool := tools.RegisterCilium()
		s.mcpServer.AddTool(ciliumTool, tools.CreateToolHandler(cilium.NewExecutor(), s.cfg))
	}

	return nil
}

// Run starts the service with the specified transport
func (s *Service) Run() error {
	// Start the server
	if s.cfg.Transport == "stdio" {
		log.Println("MCP Kubernetes version:", version.GetVersion())
		log.Println("Listening for requests on STDIO...")
		return server.ServeStdio(s.mcpServer)
	} else if s.cfg.Transport == "sse" {
		// Create SSE server using the MCP server
		sse := server.NewSSEServer(s.mcpServer)
		
		log.Println("MCP Kubernetes version:", version.GetVersion())
		log.Printf("SSE server listening on port: %d", s.cfg.Port)
		return sse.Start(fmt.Sprintf(":%d", s.cfg.Port))
	} else {
		return fmt.Errorf("invalid transport type: %s (must be 'stdio' or 'sse')", s.cfg.Transport)
	}
}
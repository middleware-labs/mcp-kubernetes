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

	// Register individual kubectl commands based on permission level
	s.registerKubectlCommands()

	// Legacy registration for kubectl tool, if needed
	// // Register generic kubectl tool for backward compatibility
	// kubectlTool := tools.RegisterKubectl()
	// s.mcpServer.AddTool(kubectlTool, tools.CreateToolHandler(kubectl.NewExecutor(), s.cfg))

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

// registerKubectlCommands registers individual kubectl commands as separate tools
func (s *Service) registerKubectlCommands() {
	// Register read-only kubectl commands
	for _, cmd := range tools.GetReadOnlyKubectlCommands() {
		kubectlTool := tools.RegisterKubectlCommand(cmd)
		commandExecutor := kubectl.CreateCommandExecutor(cmd.Name)
		s.mcpServer.AddTool(kubectlTool, tools.CreateToolHandler(tools.CommandExecutorFunc(commandExecutor), s.cfg))
	}

	// Only register read-write and admin commands if not in read-only mode
	if !s.cfg.ReadOnly {
		// Register read-write kubectl commands
		for _, cmd := range tools.GetReadWriteKubectlCommands() {
			kubectlTool := tools.RegisterKubectlCommand(cmd)
			commandExecutor := kubectl.CreateCommandExecutor(cmd.Name)
			s.mcpServer.AddTool(kubectlTool, tools.CreateToolHandler(tools.CommandExecutorFunc(commandExecutor), s.cfg))
		}

		// Register admin kubectl commands
		for _, cmd := range tools.GetAdminKubectlCommands() {
			kubectlTool := tools.RegisterKubectlCommand(cmd)
			commandExecutor := kubectl.CreateCommandExecutor(cmd.Name)
			s.mcpServer.AddTool(kubectlTool, tools.CreateToolHandler(tools.CommandExecutorFunc(commandExecutor), s.cfg))
		}
	}
}
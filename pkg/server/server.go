package server

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/mcp-kubernetes/pkg/cilium"
	"github.com/Azure/mcp-kubernetes/pkg/config"
	"github.com/Azure/mcp-kubernetes/pkg/helm"
	"github.com/Azure/mcp-kubernetes/pkg/hubble"
	"github.com/Azure/mcp-kubernetes/pkg/kubectl"
	"github.com/Azure/mcp-kubernetes/pkg/security"
	"github.com/Azure/mcp-kubernetes/pkg/tools"
	"github.com/Azure/mcp-kubernetes/pkg/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// PermissionMetadata stores information about the current permission state
type PermissionMetadata struct {
	CurrentAccessLevel   string   `json:"current_access_level"`
	RequestedAccessLevel string   `json:"requested_access_level"`
	WasDowngraded        bool     `json:"was_downgraded"`
	ClusterRoleFound     bool     `json:"cluster_role_found"`
	ValidationEnabled    bool     `json:"validation_enabled"`
	ValidationError      string   `json:"validation_error,omitempty"`
	AvailableTools       []string `json:"available_tools"`
	Timestamp            string   `json:"timestamp"`
}

// Service represents the MCP Kubernetes service
type Service struct {
	cfg                *config.ConfigData
	mcpServer          *server.MCPServer
	pulsarWorker       *kubectl.Worker
	Hostname           string // Hostname of the user
	permissionMetadata *PermissionMetadata
}

// NewService creates a new MCP Kubernetes service
func NewService(cfg *config.ConfigData) *Service {
	return &Service{
		cfg: cfg,
	}
}

// Initialize initializes the service
func (s *Service) Initialize() error {
	// Initialize configuration

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"MCP Kubernetes",
		version.GetVersion(),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	timeout := 60
	var err error
	if os.Getenv("TIMEOUT") != "" {
		timeout, err = strconv.Atoi(os.Getenv("TIMEOUT"))
		if err != nil {
			timeout = 60
		}
	}

	fingerprint := strconv.FormatInt(time.Now().UnixMilli(), 10)
	if os.Getenv("FINGERPRINT") != "" {
		fingerprint = os.Getenv("FINGERPRINT")
	}

	// Register individual kubectl commands based on permission level
	pulsar, _ := kubectl.New(&kubectl.Config{
		Mode:                1,
		Location:            os.Getenv("HOSTNAME"),
		AccountUID:          os.Getenv("ACCOUNT_UID"),
		Hostname:            os.Getenv("HOSTNAME"),
		PulsarHost:          os.Getenv("PULSAR_HOST"),
		Timeout:             timeout,
		NCAPassword:         os.Getenv("NCA_PASSWORD"),
		UnsubscribeEndpoint: os.Getenv("UNSUBSCRIBE_ENDPOINT"),
		Token:               os.Getenv("TOKEN"),
		Fingerprint:         fingerprint,
	})
	s.pulsarWorker = pulsar

	topic := fmt.Sprintf("mcp-%s-%x", strings.ToLower(os.Getenv("TOKEN")), sha1.Sum([]byte(strings.ToLower(os.Getenv("HOSTNAME")))))
	if err := s.pulsarWorker.StartSubscriber(topic + "-unsubscribe"); err != nil {
		log.Fatalf("failed to start subscriber: %v", err)
	}

	// Initialize permission metadata
	requestedAccessLevel := s.cfg.AccessLevel
	s.permissionMetadata = &PermissionMetadata{
		CurrentAccessLevel:   s.cfg.AccessLevel,
		RequestedAccessLevel: requestedAccessLevel,
		WasDowngraded:        false,
		ClusterRoleFound:     false,
		ValidationEnabled:    s.cfg.ValidateClusterRole,
		ValidationError:      "",
		AvailableTools:       []string{},
		Timestamp:            time.Now().Format(time.RFC3339),
	}

	if s.cfg.ValidateClusterRole && (s.cfg.AccessLevel == "admin" || s.cfg.AccessLevel == "readwrite") {
		log.Printf("Validating permissions by checking for mw-opsai-cluster-role...")

		time.Sleep(2 * time.Second)

		result := s.pulsarWorker.CheckClusterRolePermission(timeout)

		s.permissionMetadata.ClusterRoleFound = result.ClusterRoleFound

		if !result.Success {
			switch result.ErrorType {
			case "connection":
				log.Printf("Connection issue during cluster role validation: %s", result.ErrorMessage)
				s.permissionMetadata.ValidationError = fmt.Sprintf("connection_issue: %s", result.ErrorMessage)
				s.permissionMetadata.CurrentAccessLevel = "readonly"
				s.permissionMetadata.WasDowngraded = true

			case "timeout":
				log.Printf("Timeout during cluster role validation: %s", result.ErrorMessage)
				s.permissionMetadata.ValidationError = fmt.Sprintf("timeout_issue: %s", result.ErrorMessage)
				s.permissionMetadata.CurrentAccessLevel = "readonly"
				s.permissionMetadata.WasDowngraded = true

			default:
				log.Printf("Warning: Failed to validate cluster role: %s", result.ErrorMessage)
				log.Printf("Downgrading from '%s' to 'readonly' for safety", s.cfg.AccessLevel)
				s.cfg.AccessLevel = "readonly"
				s.cfg.SecurityConfig.AccessLevel = security.AccessLevelReadOnly
				s.permissionMetadata.CurrentAccessLevel = "readonly"
				s.permissionMetadata.WasDowngraded = true
				s.permissionMetadata.ValidationError = result.ErrorMessage
			}
		} else if !result.HasAdminRole {
			log.Printf("mw-opsai-cluster-role not found, downgrading from '%s' to 'readonly'", s.cfg.AccessLevel)
			s.cfg.AccessLevel = "readonly"
			s.cfg.SecurityConfig.AccessLevel = security.AccessLevelReadOnly
			s.permissionMetadata.CurrentAccessLevel = "readonly"
			s.permissionMetadata.WasDowngraded = true
		} else {
			log.Printf("mw-opsai-cluster-role found, using requested access level: %s", s.cfg.AccessLevel)
			s.permissionMetadata.CurrentAccessLevel = s.cfg.AccessLevel
		}
	}

	s.registerKubectlCommands()

	if s.cfg.AdditionalTools["helm"] {
		helmTool := helm.RegisterHelm()
		s.mcpServer.AddTool(helmTool, tools.CreateToolHandler(helm.NewExecutor(), s.cfg))
	}

	if s.cfg.AdditionalTools["cilium"] {
		ciliumTool := cilium.RegisterCilium()
		s.mcpServer.AddTool(ciliumTool, tools.CreateToolHandler(cilium.NewExecutor(), s.cfg))
	}

	if s.cfg.AdditionalTools["hubble"] {
		hubbleTool := hubble.RegisterHubble()
		s.mcpServer.AddTool(hubbleTool, tools.CreateToolHandler(hubble.NewExecutor(), s.cfg))
	}

	return nil
}

// Run starts the service with the specified transport
func (s *Service) Run() error {
	log.Println("MCP Kubernetes version:", version.GetVersion())

	// Start the server
	switch s.cfg.Transport {
	case "stdio":
		log.Println("Listening for requests on STDIO...")
		return server.ServeStdio(s.mcpServer)
	case "sse":
		sse := server.NewSSEServer(s.mcpServer)
		addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
		log.Printf("SSE server listening on %s", addr)
		return sse.Start(addr)
	case "streamable-http":
		streamableServer := server.NewStreamableHTTPServer(s.mcpServer)
		addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
		log.Printf("Streamable HTTP server listening on %s", addr)
		return streamableServer.Start(addr)
	default:
		return fmt.Errorf("invalid transport type: %s (must be 'stdio', 'sse' or 'streamable-http')", s.cfg.Transport)
	}
}

// registerKubectlCommands registers kubectl tools based on access level
func (s *Service) registerKubectlCommands() {
	// Get kubectl tools filtered by access level
	kubectlTools := kubectl.RegisterKubectlTools(s.cfg.AccessLevel)

	// Create a kubectl executor
	kubectlExecutor := kubectl.NewKubectlToolExecutor(s.pulsarWorker)

	// Register each kubectl tool
	for _, tool := range kubectlTools {
		// Collect tool names for metadata
		s.permissionMetadata.AvailableTools = append(s.permissionMetadata.AvailableTools, tool.Name)

		// Special handler for check_permissions tool
		if tool.Name == "kubectl_check_permissions" {
			handler := s.createCheckPermissionsHandler()
			s.mcpServer.AddTool(tool, handler)
		} else {
			// Create a handler that injects the tool name into params
			handler := tools.CreateToolHandlerWithName(kubectlExecutor, s.cfg, tool.Name)
			s.mcpServer.AddTool(tool, handler)
		}
	}
}

// createCheckPermissionsHandler creates a custom handler for the check_permissions tool
func (s *Service) createCheckPermissionsHandler() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Return the current permission metadata as JSON
		jsonData, err := json.MarshalIndent(s.permissionMetadata, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve permission metadata: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

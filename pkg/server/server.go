package server

import (
	"crypto/sha1"
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
	"github.com/mark3labs/mcp-go/server"
)

// Service represents the MCP Kubernetes service
type Service struct {
	cfg          *config.ConfigData
	mcpServer    *server.MCPServer
	pulsarWorker *kubectl.Worker
	Hostname     string // Hostname of the user
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

	// print the pulsar all configs
	fmt.Printf("Pulsar Configs: Mode=%d, Location=%s, AccountUID=%s, Hostname=%s, PulsarHost=%s, Timeout=%d, NCAPassword=%s, UnsubscribeEndpoint=%s, Token=%s, Fingerprint=%s\n",
		1,
		os.Getenv("HOSTNAME"),
		os.Getenv("ACCOUNT_UID"),
		os.Getenv("HOSTNAME"),
		os.Getenv("PULSAR_HOST"),
		timeout,
		os.Getenv("NCA_PASSWORD"),
		os.Getenv("UNSUBSCRIBE_ENDPOINT"),
	)

	topic := fmt.Sprintf("mcp-%s-%x", strings.ToLower(os.Getenv("TOKEN")), sha1.Sum([]byte(strings.ToLower(os.Getenv("HOSTNAME")))))
	if err := s.pulsarWorker.StartSubscriber(topic + "-unsubscribe"); err != nil {
		log.Fatalf("failed to start subscriber: %v", err)
	}

	// Validate permissions via cluster role check if enabled
	if s.cfg.ValidateClusterRole && (s.cfg.AccessLevel == "admin" || s.cfg.AccessLevel == "readwrite") {
		log.Printf("Validating permissions by checking for mw-opsai-cluster-role...")

		// Wait a bit for subscriber to be ready
		time.Sleep(2 * time.Second)

		hasAdminRole, err := s.pulsarWorker.CheckClusterRolePermission(timeout)
		if err != nil {
			// If validation fails (timeout, error, etc.), downgrade to readonly for safety
			log.Printf("Warning: Failed to validate cluster role: %v", err)
			log.Printf("Downgrading from '%s' to 'readonly' for safety", s.cfg.AccessLevel)
			s.cfg.AccessLevel = "readonly"
			s.cfg.SecurityConfig.AccessLevel = security.AccessLevelReadOnly
		} else if !hasAdminRole {
			// Cluster role not found, downgrade to readonly
			log.Printf("mw-opsai-cluster-role not found, downgrading from '%s' to 'readonly'", s.cfg.AccessLevel)
			s.cfg.AccessLevel = "readonly"
			s.cfg.SecurityConfig.AccessLevel = security.AccessLevelReadOnly
		} else {
			// Cluster role found, use requested access level
			log.Printf("mw-opsai-cluster-role found, using requested access level: %s", s.cfg.AccessLevel)
		}
	}

	// Register kubectl commands AFTER validation (so tools are filtered correctly)
	s.registerKubectlCommands()

	// Register additional tools
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
		// Create a handler that injects the tool name into params
		handler := tools.CreateToolHandlerWithName(kubectlExecutor, s.cfg, tool.Name)
		s.mcpServer.AddTool(tool, handler)
	}
}

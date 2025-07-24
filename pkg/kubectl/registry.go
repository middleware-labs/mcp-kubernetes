package kubectl

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// KubectlCommand defines a specific kubectl command to be registered as a tool
type KubectlCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments, such as "pods" or "-f deployment.yaml"
}

// Example shows how to use a consolidated tool
type Example struct {
	Description string
	Operation   string
	Resource    string
	Args        string
}

// Access level constants
const (
	AccessLevelReadOnly  = "readonly"
	AccessLevelReadWrite = "readwrite"
	AccessLevelAdmin     = "admin"
)

// toolCreator is a function that creates a tool, possibly with read-only restrictions
type toolCreator func(readOnly bool) mcp.Tool

// toolCreatorSimple is a function that creates a tool without read-only parameter
type toolCreatorSimple func() mcp.Tool

// toolRegistration defines when a tool should be registered
type toolRegistration struct {
	creator      interface{} // either toolCreator or toolCreatorSimple
	minAccess    string      // minimum access level required: "readonly", "readwrite", or "admin"
	readOnlyMode bool        // whether to pass true to creator when in readonly mode
}

// RegisterKubectlTools returns kubectl tools filtered by access level
func RegisterKubectlTools(accessLevel string) []mcp.Tool {
	// Define tool registry with access requirements
	toolRegistry := []toolRegistration{
		{creator: toolCreator(createResourcesTool), minAccess: AccessLevelReadOnly, readOnlyMode: true},
		{creator: toolCreatorSimple(createDiagnosticsTool), minAccess: AccessLevelReadOnly},
		{creator: toolCreatorSimple(createClusterTool), minAccess: AccessLevelReadOnly},
		{creator: toolCreator(createConfigTool), minAccess: AccessLevelReadOnly, readOnlyMode: true},
		{creator: toolCreatorSimple(createWorkloadsTool), minAccess: AccessLevelReadWrite},
		{creator: toolCreatorSimple(createMetadataTool), minAccess: AccessLevelReadWrite},
		{creator: toolCreatorSimple(createNodesTool), minAccess: AccessLevelAdmin},
	}

	// Normalize access level
	if !isValidAccessLevel(accessLevel) {
		accessLevel = AccessLevelReadOnly // Default to readonly for safety
	}

	var tools []mcp.Tool
	for _, reg := range toolRegistry {
		if shouldRegisterTool(reg.minAccess, accessLevel) {
			tool := createToolFromRegistration(reg, accessLevel)
			tools = append(tools, tool)
		}
	}

	return tools
}

// isValidAccessLevel checks if the given access level is valid
func isValidAccessLevel(accessLevel string) bool {
	return accessLevel == AccessLevelReadOnly ||
		accessLevel == AccessLevelReadWrite ||
		accessLevel == AccessLevelAdmin
}

// shouldRegisterTool determines if a tool should be registered based on access levels
func shouldRegisterTool(minAccess, currentAccess string) bool {
	accessLevels := map[string]int{
		AccessLevelReadOnly:  1,
		AccessLevelReadWrite: 2,
		AccessLevelAdmin:     3,
	}

	minLevel := accessLevels[minAccess]
	currentLevel := accessLevels[currentAccess]

	return currentLevel >= minLevel
}

// createToolFromRegistration creates a tool from its registration definition
func createToolFromRegistration(reg toolRegistration, accessLevel string) mcp.Tool {
	switch creator := reg.creator.(type) {
	case toolCreator:
		// Tools that support read-only mode
		readOnly := accessLevel == AccessLevelReadOnly && reg.readOnlyMode
		return creator(readOnly)
	case toolCreatorSimple:
		// Tools that don't have read-only variants
		return creator()
	default:
		panic("invalid tool creator type")
	}
}

// createResourcesTool creates the main resource management tool
func createResourcesTool(readOnly bool) mcp.Tool {
	var description string
	var operationDesc string

	if readOnly {
		description = `View Kubernetes resources with read-only operations.

Available operations:
- get: Display one or many resources
- describe: Show detailed information about resources

Common resources: pods, deployments, services, configmaps, secrets, namespaces, etc.

Examples:
- Get pods: operation='get', resource='pods', args='-n default'
- Describe deployment: operation='describe', resource='deployment', args='myapp -n production'`
		operationDesc = "The operation to perform: get, describe"
	} else {
		description = `Manage Kubernetes resources with standard CRUD operations.

Available operations:
- get: Display one or many resources
- describe: Show detailed information about resources
- create: Create a resource from a file or stdin
- delete: Delete resources
- apply: Apply a configuration to a resource
- patch: Update fields of a resource
- replace: Replace a resource

Common resources: pods, deployments, services, configmaps, secrets, namespaces, etc.

Examples:
- Get pods: operation='get', resource='pods', args='-n default'
- Describe deployment: operation='describe', resource='deployment', args='myapp -n production'
- Apply config: operation='apply', resource='', args='-f deployment.yaml'
- Delete service: operation='delete', resource='service', args='myservice -n default'`
		operationDesc = "The operation to perform: get, describe, create, delete, apply, patch, replace"
	}

	return mcp.NewTool("kubectl_resources",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description(operationDesc),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type (e.g., pods, deployments, services) or empty for file-based operations"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Additional arguments like resource names, namespaces, and flags"),
		),
	)
}

// createWorkloadsTool creates the workload management tool
func createWorkloadsTool() mcp.Tool {
	description := `Manage Kubernetes workloads and their lifecycle.

Available operations:
- run: Run a particular image on the cluster
- expose: Expose a resource as a new Kubernetes service
- scale: Set a new size for a deployment, replica set, or replication controller
- autoscale: Auto-scale a deployment, replica set, stateful set, or replication controller
- rollout: Manage the rollout of resources (status, history, undo, restart, pause, resume)

Examples:
- Run nginx: operation='run', resource='deployment', args='nginx --image=nginx:latest'
- Scale deployment: operation='scale', resource='deployment', args='myapp --replicas=3'
- Rollout status: operation='rollout', resource='status', args='deployment/myapp'
- Autoscale: operation='autoscale', resource='deployment', args='myapp --min=2 --max=10'`

	return mcp.NewTool("kubectl_workloads",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: run, expose, scale, autoscale, rollout"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type or rollout subcommand"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Additional arguments specific to the operation"),
		),
	)
}

// createMetadataTool creates the metadata management tool
func createMetadataTool() mcp.Tool {
	description := `Manage metadata for Kubernetes resources.

Available operations:
- label: Update labels on a resource
- annotate: Update annotations on a resource
- set: Set specific features on objects (e.g., image, resources, selector)

Examples:
- Add label: operation='label', resource='pods', args='mypod env=production'
- Remove label: operation='label', resource='pods', args='mypod env-'
- Add annotation: operation='annotate', resource='deployment', args='myapp description="My application"'
- Set image: operation='set', resource='image', args='deployment/myapp nginx=nginx:1.19'`

	return mcp.NewTool("kubectl_metadata",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: label, annotate, set"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type to modify"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Resource names and metadata changes"),
		),
	)
}

// createDiagnosticsTool creates the diagnostics and debugging tool
func createDiagnosticsTool() mcp.Tool {
	description := `Diagnose and debug Kubernetes resources.

Available operations:
- logs: Print logs for a container in a pod
- events: Display events
- top: Display resource usage (CPU/Memory)
- exec: Execute a command in a container
- cp: Copy files to/from containers

Examples:
- View logs: operation='logs', resource='pod', args='mypod -n default'
- Follow logs: operation='logs', resource='pod', args='mypod -f --tail=100'
- Get events: operation='events', resource='', args='--all-namespaces'
- Exec shell: operation='exec', resource='pod', args='mypod -it -- /bin/bash'
- Copy file: operation='cp', resource='', args='mypod:/path/to/file ./local/file'`

	return mcp.NewTool("kubectl_diagnostics",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: logs, events, top, exec, cp"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type (usually 'pod' or empty for some operations)"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Resource names and operation-specific flags"),
		),
	)
}

// createClusterTool creates the cluster information tool
func createClusterTool() mcp.Tool {
	description := `Get information about the Kubernetes cluster and API.

Available operations:
- cluster-info: Display cluster information
- api-resources: Print supported API resources
- api-versions: Print supported API versions
- explain: Get documentation for a resource

Examples:
- Cluster info: operation='cluster-info', resource='', args=''
- List resources: operation='api-resources', resource='', args='--namespaced=true'
- API versions: operation='api-versions', resource='', args=''
- Explain pod: operation='explain', resource='pod', args='--recursive'`

	return mcp.NewTool("kubectl_cluster",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: cluster-info, api-resources, api-versions, explain"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type for explain operation, or empty for others"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Additional flags and options"),
		),
	)
}

// createNodesTool creates the node management tool
func createNodesTool() mcp.Tool {
	description := `Manage Kubernetes nodes.

Available operations:
- cordon: Mark node as unschedulable
- uncordon: Mark node as schedulable
- drain: Drain node in preparation for maintenance
- taint: Update taints on nodes

Examples:
- Cordon node: operation='cordon', resource='node', args='worker-1'
- Drain node: operation='drain', resource='node', args='worker-1 --ignore-daemonsets'
- Add taint: operation='taint', resource='nodes', args='worker-1 key=value:NoSchedule'
- Remove taint: operation='taint', resource='nodes', args='worker-1 key:NoSchedule-'`

	return mcp.NewTool("kubectl_nodes",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: cordon, uncordon, drain, taint"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("Usually 'node' or 'nodes'"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Node names and operation-specific flags"),
		),
	)
}

// createConfigTool creates the configuration tool
func createConfigTool(readOnly bool) mcp.Tool {
	var description string
	var operationDesc string

	if readOnly {
		description = `Work with Kubernetes configurations (read-only).

Available operations:
- diff: Diff the live version against what would be applied
- auth: Inspect authorization (can-i)

Examples:
- Diff config: operation='diff', resource='', args='-f deployment.yaml'
- Check auth: operation='auth', resource='can-i', args='create pods --namespace=default'`
		operationDesc = "The operation to perform: diff, auth"
	} else {
		description = `Work with Kubernetes configurations.

Available operations:
- diff: Diff the live version against what would be applied
- auth: Inspect authorization (can-i)
- certificate: Manage certificate resources (approve, deny)

Examples:
- Diff config: operation='diff', resource='', args='-f deployment.yaml'
- Check auth: operation='auth', resource='can-i', args='create pods --namespace=default'
- Approve cert: operation='certificate', resource='approve', args='csr-name'`
		operationDesc = "The operation to perform: diff, auth, certificate"
	}

	return mcp.NewTool("kubectl_config",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description(operationDesc),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("Subcommand for auth/certificate operations, or empty for diff"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Operation-specific arguments"),
		),
	)
}

// GetKubectlToolNames returns the names of all kubectl tools
func GetKubectlToolNames() []string {
	return []string{
		"kubectl_resources",
		"kubectl_workloads",
		"kubectl_metadata",
		"kubectl_diagnostics",
		"kubectl_cluster",
		"kubectl_nodes",
		"kubectl_config",
	}
}

// MapOperationToCommand maps consolidated operations to kubectl commands
func MapOperationToCommand(toolName, operation, resource string) (string, error) {
	// This function will be used by the executor to map operations to actual kubectl commands
	// For now, return a basic mapping
	switch toolName {
	case "kubectl_resources":
		return operation, nil
	case "kubectl_workloads":
		if operation == "rollout" {
			return "rollout " + resource, nil
		}
		return operation, nil
	case "kubectl_metadata":
		return operation, nil
	case "kubectl_diagnostics":
		return operation, nil
	case "kubectl_cluster":
		return operation, nil
	case "kubectl_nodes":
		return operation, nil
	case "kubectl_config":
		if operation == "auth" {
			return "auth " + resource, nil
		}
		if operation == "certificate" {
			return "certificate " + resource, nil
		}
		return operation, nil
	default:
		return "", nil
	}
}

// GetReadOnlyKubectlCommands returns all read-only kubectl commands
func GetReadOnlyKubectlCommands() []KubectlCommand {
	return []KubectlCommand{
		{Name: "get", Description: "Display one or many resources", ArgsExample: "pods"},
		{Name: "describe", Description: "Show details of a specific resource or group of resources", ArgsExample: "pod nginx-pod"},
		{Name: "explain", Description: "Documentation of resources", ArgsExample: "pods.spec.containers"},
		{Name: "logs", Description: "Print the logs for a container in a pod", ArgsExample: "nginx-pod"},
		{Name: "api-resources", Description: "Print the supported API resources on the server", ArgsExample: "--namespaced=true"},
		{Name: "api-versions", Description: "Print the supported API versions on the server", ArgsExample: ""},
		{Name: "diff", Description: "Diff live configuration against a would-be applied file", ArgsExample: "-f deployment.yaml"},
		{Name: "cluster-info", Description: "Display cluster info", ArgsExample: ""},
		{Name: "top", Description: "Display resource usage", ArgsExample: "pods"},
		{Name: "events", Description: "List events in the cluster", ArgsExample: "--all-namespaces"},
		{Name: "auth", Description: "Inspect authorization", ArgsExample: "can-i create pods"},
	}
}

// GetReadWriteKubectlCommands returns all read-write kubectl commands
func GetReadWriteKubectlCommands() []KubectlCommand {
	return []KubectlCommand{
		{Name: "create", Description: "Create a resource from a file or from stdin", ArgsExample: "deployment nginx --image=nginx"},
		{Name: "delete", Description: "Delete resources by file names, stdin, resources and names, or by resources and label selector", ArgsExample: "pod nginx-pod"},
		{Name: "apply", Description: "Apply a configuration to a resource by file name or stdin", ArgsExample: "-f deployment.yaml"},
		{Name: "expose", Description: "Take a replication controller, service, deployment or pod and expose it as a new Kubernetes Service", ArgsExample: "deployment nginx --port=80 --type=ClusterIP"},
		{Name: "run", Description: "Run a particular image on the cluster", ArgsExample: "nginx --image=nginx"},
		{Name: "set", Description: "Set specific features on objects", ArgsExample: "image deployment/nginx nginx=nginx:latest"},
		{Name: "rollout", Description: "Manage the rollout of a resource", ArgsExample: "status deployment/nginx"},
		{Name: "scale", Description: "Set a new size for a Deployment, ReplicaSet, Replication Controller, or StatefulSet", ArgsExample: "deployment/nginx --replicas=3"},
		{Name: "autoscale", Description: "Auto-scale a Deployment, ReplicaSet, or StatefulSet", ArgsExample: "deployment/nginx --min=2 --max=5 --cpu-percent=80"},
		{Name: "label", Description: "Update the labels on a resource", ArgsExample: "pod nginx-pod app=web"},
		{Name: "annotate", Description: "Update the annotations on a resource", ArgsExample: "pod nginx-pod description='Web server'"},
		{Name: "patch", Description: "Update field(s) of a resource", ArgsExample: "pod nginx-pod -p '{\"spec\":{\"containers\":[{\"name\":\"nginx\",\"image\":\"nginx:latest\"}]}}'"},
		{Name: "replace", Description: "Replace a resource by file name or stdin", ArgsExample: "-f updated-deployment.yaml"},
		{Name: "cp", Description: "Copy files and directories to and from containers", ArgsExample: "nginx-pod:/var/log/nginx/access.log ./access.log"},
		{Name: "exec", Description: "Execute a command in a container", ArgsExample: "nginx-pod -- ls /usr/share/nginx/html"},
	}
}

// GetAdminKubectlCommands returns all admin kubectl commands
func GetAdminKubectlCommands() []KubectlCommand {
	return []KubectlCommand{
		{Name: "cordon", Description: "Mark node as unschedulable", ArgsExample: "worker-node-1"},
		{Name: "uncordon", Description: "Mark node as schedulable", ArgsExample: "worker-node-1"},
		{Name: "drain", Description: "Drain node in preparation for maintenance", ArgsExample: "worker-node-1 --ignore-daemonsets"},
		{Name: "taint", Description: "Update the taints on one or more nodes", ArgsExample: "worker-node-1 key=value:NoSchedule"},
		{Name: "certificate", Description: "Modify certificate resources", ArgsExample: "approve my-cert-csr"},
	}
}

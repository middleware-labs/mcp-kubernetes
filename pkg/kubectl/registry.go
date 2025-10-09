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
		{creator: toolCreatorSimple(createCheckPermissionsTool), minAccess: AccessLevelReadOnly},
		{creator: toolCreatorSimple(createWorkloadsTool), minAccess: AccessLevelReadWrite},
		{creator: toolCreatorSimple(createMetadataTool), minAccess: AccessLevelReadWrite},
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
- Get specific pod: operation='get', resource='pods', args='nginx-pod -n default'
- Get with selector: operation='get', resource='pods', args='-l app=nginx'
- Get all namespaces: operation='get', resource='pods', args='--all-namespaces'
- Describe deployment: operation='describe', resource='deployment', args='myapp -n production'
- Describe all pods: operation='describe', resource='pods', args=''
- Describe with selector: operation='describe', resource='pods', args='-l name=myLabel'`
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
- cordon: Mark node as unschedulable (admin only)
- uncordon: Mark node as schedulable (admin only)
- drain: Drain node in preparation for maintenance (admin only)
- taint: Update taints on nodes (admin only)

Common resources: pods, deployments, services, configmaps, secrets, namespaces, nodes, etc.

Examples:
- Get pods: operation='get', resource='pods', args='-n default'
- Get specific pod: operation='get', resource='pods', args='nginx-pod -n default'
- Get with selector: operation='get', resource='pods', args='-l app=nginx'
- Get all namespaces: operation='get', resource='pods', args='--all-namespaces'
- Describe deployment: operation='describe', resource='deployment', args='myapp -n production'
- Describe all pods: operation='describe', resource='pods', args=''
- Describe with selector: operation='describe', resource='pods', args='-l name=myLabel'
- Create from file: operation='create', resource='', args='-f deployment.yaml'
- Create deployment: operation='create', resource='deployment', args='nginx --image=nginx'
- Create configmap: operation='create', resource='configmap', args='my-config --from-literal=key1=value1'
- Apply config: operation='apply', resource='', args='-f deployment.yaml'
- Apply kustomize: operation='apply', resource='', args='-k ./manifests/'
- Patch node: operation='patch', resource='node', args='k8s-node-1 -p \'{"spec":{"unschedulable":true}}\''
- Patch from file: operation='patch', resource='', args='-f node.json -p \'{"spec":{"unschedulable":true}}\''
- Patch pod image: operation='patch', resource='pod', args='valid-pod -p \'{"spec":{"containers":[{"name":"app","image":"nginx:1.20"}]}}\''
- Patch with JSON type: operation='patch', resource='pod', args='valid-pod --type=json -p \'[{"op":"replace","path":"/spec/containers/0/image","value":"nginx:1.20"}]\''
- Replace from file: operation='replace', resource='', args='-f ./updated-pod.json'
- Force replace: operation='replace', resource='', args='--force -f ./pod.json'
- Delete service: operation='delete', resource='service', args='myservice -n default'
- Delete from file: operation='delete', resource='', args='-f pod.yaml'
- Delete with selector: operation='delete', resource='pods', args='-l name=myLabel'
- Cordon node: operation='cordon', resource='node', args='worker-1'
- Uncordon node: operation='uncordon', resource='node', args='worker-1'
- Cordon with selector: operation='cordon', resource='node', args='-l node-type=worker'
- Drain node: operation='drain', resource='node', args='worker-1 --ignore-daemonsets'
- Drain with force: operation='drain', resource='node', args='worker-1 --force --ignore-daemonsets'
- Drain with grace period: operation='drain', resource='node', args='worker-1 --grace-period=900'
- Add taint: operation='taint', resource='nodes', args='worker-1 dedicated=special-user:NoSchedule'
- Remove taint: operation='taint', resource='nodes', args='worker-1 dedicated:NoSchedule-'
- Taint with selector: operation='taint', resource='node', args='-l myLabel=X dedicated=foo:PreferNoSchedule'`
		operationDesc = "The operation to perform: get, describe, create, delete, apply, patch, replace, cordon, uncordon, drain, taint"
	}

	return mcp.NewTool("kubectl_resources",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description(operationDesc),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type (e.g., pods, deployments, services) or empty string '' for file-based operations (create -f, apply -f, patch -f, replace -f, delete -f)"),
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
- run: Run a Pod with particular image on the cluster
- expose: Expose a resource as a new Kubernetes service
- scale: Set a new size for a deployment, replica set, or replication controller
- autoscale: Auto-scale a deployment, replica set, stateful set, or replication controller
- rollout: Manage the rollout of resources (status, history, undo, restart, pause, resume)

Examples:
- Run nginx pod: operation='run', resource='', args='nginx --image=nginx'
- Run with port: operation='run', resource='', args='hazelcast --image=hazelcast/hazelcast --port=5701'
- Run with env vars: operation='run', resource='', args='hazelcast --image=hazelcast/hazelcast --env="DNS_DOMAIN=cluster"'
- Run with labels: operation='run', resource='', args='nginx --image=nginx --labels="app=web,env=prod"'
- Expose deployment: operation='expose', resource='deployment', args='nginx --port=80 --target-port=8000'
- Expose pod: operation='expose', resource='pod', args='valid-pod --port=444 --name=frontend'
- Scale deployment: operation='scale', resource='deployment', args='myapp --replicas=3'
- Autoscale deployment: operation='autoscale', resource='deployment', args='foo --min=2 --max=10'
- Autoscale with CPU: operation='autoscale', resource='rc', args='foo --max=5 --cpu-percent=80'
- Rollout status: operation='rollout', resource='status', args='deployment/myapp'
- Rollout history: operation='rollout', resource='history', args='deployment/abc'
- Rollout undo: operation='rollout', resource='undo', args='deployment/abc'
- Rollout restart: operation='rollout', resource='restart', args='deployment/abc'`

	return mcp.NewTool("kubectl_workloads",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: run, expose, scale, autoscale, rollout"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type for expose/scale/autoscale, subcommand for rollout, or empty string '' for run operation"),
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
- Add label: operation='label', resource='pods', args='foo unhealthy=true'
- Overwrite label: operation='label', resource='pods', args='--overwrite foo status=unhealthy'
- Remove label: operation='label', resource='pods', args='foo bar-'
- Add annotation: operation='annotate', resource='pods', args='foo description="my frontend"'
- Overwrite annotation: operation='annotate', resource='pods', args='--overwrite foo description="my frontend running nginx"'
- Remove annotation: operation='annotate', resource='pods', args='foo description-'
- Set image: operation='set', resource='image', args='deployment/nginx busybox=busybox nginx=nginx:1.9.1'`

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
- Logs for default container: operation='logs', resource='', args='nginx'
- Logs for specific container: operation='logs', resource='', args='nginx -c ruby-container'
- Logs with selector: operation='logs', resource='', args='-l app=nginx --all-containers=true'
- Get events: operation='events', resource='', args='--all-namespaces'
- Get events namespace: operation='events', resource='', args='-n default'
- Top pods: operation='top', resource='pod', args=''
- Top nodes: operation='top', resource='node', args=''
- Top with containers: operation='top', resource='pod', args='POD_NAME --containers'
- Exec command: operation='exec', resource='', args='mypod -n NAMESPACE -- date'
- Copy to pod: operation='cp', resource='', args='/tmp/foo_dir some-pod:/tmp/bar_dir'
- Copy from pod: operation='cp', resource='', args='some-namespace/some-pod:/tmp/foo /tmp/bar'
- Copy with container: operation='cp', resource='', args='/tmp/foo some-pod:/tmp/bar -c specific-container'`

	return mcp.NewTool("kubectl_diagnostics",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: logs, events, top, exec, cp"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type: 'node'/'pod' for top, empty string '' for logs/events/exec/cp"),
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
- Cluster info dump: operation='cluster-info', resource='dump', args=''
- List resources: operation='api-resources', resource='', args=''
- Namespaced resources: operation='api-resources', resource='', args='--namespaced=true'
- Non-namespaced resources: operation='api-resources', resource='', args='--namespaced=false'
- Resources by group: operation='api-resources', resource='', args='--api-group=rbac.authorization.k8s.io'
- API versions: operation='api-versions', resource='', args=''
- Explain pod: operation='explain', resource='pods', args=''
- Explain field: operation='explain', resource='pods.spec.containers', args=''
- Explain with version: operation='explain', resource='deployments', args='--api-version=apps/v1'`

	return mcp.NewTool("kubectl_cluster",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform: cluster-info, api-resources, api-versions, explain"),
		),
		mcp.WithString("resource",
			mcp.Required(),
			mcp.Description("The resource type for explain operation, or empty string '' for cluster-info/api-resources/api-versions"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Additional flags and options"),
		),
	)
}

// createCheckPermissionsTool creates the permission checking tool
func createCheckPermissionsTool() mcp.Tool {
	description := `Check the current permission level and validation status of the MCP Kubernetes server.

This tool returns metadata about:
- Current access level (readonly/readwrite/admin)
- Requested access level at startup
- Whether access level was downgraded for safety
- Whether mw-opsai-cluster-role was found
- Validation status and any errors
- Available kubectl tools for current access level

Use this tool to verify what operations are available before attempting them.

Examples:
- Check current permissions: No parameters required, just call the tool

Returns JSON with:
{
  "current_access_level": "readonly|readwrite|admin",
  "requested_access_level": "readonly|readwrite|admin",
  "was_downgraded": true|false,
  "cluster_role_found": true|false,
  "validation_enabled": true|false,
  "validation_error": "error message if any",
  "available_tools": ["kubectl_resources", "kubectl_diagnostics", ...],
  "timestamp": "2025-10-03T10:59:48Z"
}`

	return mcp.NewTool("kubectl_check_permissions",
		mcp.WithDescription(description),
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
- Diff config: operation='diff', resource='', args='-f pod.json'
- Diff from stdin: operation='diff', resource='', args='-f -'
- Diff with selector: operation='diff', resource='', args='-f manifest.yaml -l app=nginx'
- Check auth: operation='auth', resource='can-i', args='create pods --all-namespaces'
- Check auth resource: operation='auth', resource='can-i', args='list deployments.apps'
- Check auth as user: operation='auth', resource='can-i', args='list pods --as=system:serviceaccount:dev:foo -n prod'
- List permissions: operation='auth', resource='can-i', args='--list --namespace=foo'`
		operationDesc = "The operation to perform: diff, auth"
	} else {
		description = `Work with Kubernetes configurations.

Available operations:
- diff: Diff the live version against what would be applied
- auth: Inspect authorization (can-i)
- certificate: Manage certificate resources (approve, deny)

Examples:
- Diff config: operation='diff', resource='', args='-f pod.json'
- Diff with selector: operation='diff', resource='', args='-f manifest.yaml -l app=nginx'
- Check auth: operation='auth', resource='can-i', args='create pods --all-namespaces'
- Check auth resource: operation='auth', resource='can-i', args='list deployments.apps'
- Check auth as user: operation='auth', resource='can-i', args='list pods --as=system:serviceaccount:dev:foo -n prod'
- List permissions: operation='auth', resource='can-i', args='--list --namespace=foo'
- Approve cert: operation='certificate', resource='approve', args='csr-name'
- Deny cert: operation='certificate', resource='deny', args='csr-name'`
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
			mcp.Description("Subcommand for auth/certificate operations, or empty string '' for diff operation"),
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
		"kubectl_config",
		"kubectl_check_permissions",
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

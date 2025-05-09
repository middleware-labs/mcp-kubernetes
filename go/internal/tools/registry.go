package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// KubectlCommand defines a specific kubectl command to be registered as a tool
type KubectlCommand struct {
	Name        string
	Description string
	ArgsExample string // Example of command arguments, such as "pods" or "-f deployment.yaml"
}

// RegisterKubectl registers the generic kubectl tool (legacy)
func RegisterKubectl() mcp.Tool {
	return mcp.NewTool("Run-kubectl-command",
		mcp.WithDescription("Run kubectl command and get result"),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("The kubectl command to execute"),
		),
	)
}

// RegisterKubectlCommand registers a specific kubectl command as an MCP tool
func RegisterKubectlCommand(cmd KubectlCommand) mcp.Tool {
	description := "Run kubectl " + cmd.Name + " command: " + cmd.Description + "."
	
	// Add example if available, with proper punctuation
	if cmd.ArgsExample != "" {
		description += "\n\nExample: `" + cmd.ArgsExample + "`"
	}
	
	return mcp.NewTool("kubectl_"+cmd.Name,
		mcp.WithDescription(description),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Arguments for the `kubectl "+cmd.Name+"` command"),
		),
	)
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



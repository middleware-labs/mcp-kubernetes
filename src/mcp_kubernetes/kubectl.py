from mcp_kubernetes.command import ShellProcess
from mcp_kubernetes.security_validator import (
    KUBECTL_READ_OPERATIONS,
    validate_namespace_scope,
)
from mcp_kubernetes.tool_registry import (
    kubectl_func_register,
    KUBECTL_READONLY,
    KUBECTL_RW,
    KUBECTL_ADMIN,
)


def _kubectl(command_prefix: str, args: str) -> str:
    """
    Run a generic kubectl command and return the output.

    Args:
        command_prefix (str): The complete kubectl command prefix, e.g., 'kubectl get'.
        args (str): Arguments to pass to the command.

    Returns:
        str: The output of the kubectl command or an error message.
    """
    error = validate_namespace_scope(args)
    if error:
        return error

    process = ShellProcess(command=command_prefix)
    output = process.run(args)
    return output


# ----- Basic Commands (Beginner) -----


@kubectl_func_register(KUBECTL_RW)
def kubectl_create(args: str) -> str:
    """
    Run a `kubectl create` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl create`.
                       For example:
                       - "deployment nginx --image=nginx" to create a deployment.
                       - "namespace test-ns" to create a namespace.

    Returns:
        str: The output of the `kubectl create` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl create` prefix; it is added automatically.
    """
    return _kubectl("kubectl create", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_expose(args: str) -> str:
    """
    Run a `kubectl expose` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl expose`.
                       For example:
                       - "deployment nginx --port=80 --target-port=8000" to expose a deployment.

    Returns:
        str: The output of the `kubectl expose` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl expose` prefix; it is added automatically.
    """
    return _kubectl("kubectl expose", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_run(args: str) -> str:
    """
    Run a `kubectl run` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl run`.
                       For example:
                       - "nginx --image=nginx" to run a specific image on the cluster.

    Returns:
        str: The output of the `kubectl run` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl run` prefix; it is added automatically.
    """
    return _kubectl("kubectl run", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_set(args: str) -> str:
    """
    Run a `kubectl set` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl set`.
                       For example:
                       - "image deployment/nginx nginx=nginx:1.16.1" to update an image.
                       - "resources deployment/nginx -c=nginx --limits=cpu=200m,memory=512Mi" to set resource limits.

    Returns:
        str: The output of the `kubectl set` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl set` prefix; it is added automatically.
    """
    return _kubectl("kubectl set", args)


# ----- Basic Commands (Intermediate) -----


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_explain(args: str) -> str:
    """
    Run a `kubectl explain` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl explain`.
                       For example:
                       - "pods" to get documentation about the pods resource.
                       - "pods.spec.containers" to get documentation about container specifications.

    Returns:
        str: The output of the `kubectl explain` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl explain` prefix; it is added automatically.
    """
    return _kubectl("kubectl explain", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_get(args: str) -> str:
    """
    Run a `kubectl get` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl get`.
                       For example:
                       - "pods" to list all pods in the current namespace.
                       - "pods -n kube-system" to list all pods in the "kube-system" namespace.
                       - "deployments --all-namespaces" to list all deployments across namespaces.

    Returns:
        str: The output of the `kubectl get` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl get` prefix; it is added automatically.
    """
    return _kubectl("kubectl get", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_delete(args: str) -> str:
    """
    Run a `kubectl delete` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl delete`.
                       For example:
                       - "pod nginx" to delete a specific pod.
                       - "deployment nginx" to delete a deployment.

    Returns:
        str: The output of the `kubectl delete` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl delete` prefix; it is added automatically.
    """
    return _kubectl("kubectl delete", args)


# ----- Deploy Commands -----


@kubectl_func_register(KUBECTL_RW)
def kubectl_rollout(args: str) -> str:
    """
    Run a `kubectl rollout` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl rollout`.
                       For example:
                       - "status deployment/nginx" to check rollout status.
                       - "history deployment/nginx" to view rollout history.
                       - "undo deployment/nginx" to undo a rollout.

    Returns:
        str: The output of the `kubectl rollout` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl rollout` prefix; it is added automatically.
    """
    return _kubectl("kubectl rollout", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_scale(args: str) -> str:
    """
    Run a `kubectl scale` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl scale`.
                       For example:
                       - "deployment/nginx --replicas=3" to scale a deployment to 3 replicas.

    Returns:
        str: The output of the `kubectl scale` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl scale` prefix; it is added automatically.
    """
    return _kubectl("kubectl scale", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_autoscale(args: str) -> str:
    """
    Run a `kubectl autoscale` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl autoscale`.
                       For example:
                       - "deployment/nginx --min=2 --max=10 --cpu-percent=80" to autoscale a deployment.

    Returns:
        str: The output of the `kubectl autoscale` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl autoscale` prefix; it is added automatically.
    """
    return _kubectl("kubectl autoscale", args)


# ----- Cluster Management Commands -----


@kubectl_func_register(KUBECTL_ADMIN)
def kubectl_certificate(args: str) -> str:
    """
    Run a `kubectl certificate` command and return the output.

    Args:
        args (str): The specific options to pass to `kubectl certificate`.
                       For example:
                       - "approve csr-xxxxx" to approve a certificate signing request.

    Returns:
        str: The output of the `kubectl certificate` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl certificate` prefix; it is added automatically.
    """
    return _kubectl("kubectl certificate", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_cluster_info(args: str) -> str:
    """
    Run a `kubectl cluster-info` command and return the output.

    Args:
        args (str): The specific options to pass to `kubectl cluster-info`.
                       For example:
                       - "" (empty string) to get basic cluster info.
                       - "dump" to get more detailed information.

    Returns:
        str: The output of the `kubectl cluster-info` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl cluster-info` prefix; it is added automatically.
    """
    return _kubectl("kubectl cluster-info", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_top(args: str) -> str:
    """
    Run a `kubectl top` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl top`.
                       For example:
                       - "pods" to show resource usage of pods.
                       - "nodes" to show resource usage of nodes.

    Returns:
        str: The output of the `kubectl top` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl top` prefix; it is added automatically.
    """
    return _kubectl("kubectl top", args)


@kubectl_func_register(KUBECTL_ADMIN)
def kubectl_cordon(args: str) -> str:
    """
    Run a `kubectl cordon` command and return the output.

    Args:
        args (str): The name of the node to mark as unschedulable.
                       For example:
                       - "node-1" to mark node-1 as unschedulable.

    Returns:
        str: The output of the `kubectl cordon` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl cordon` prefix; it is added automatically.
    """
    return _kubectl("kubectl cordon", args)


@kubectl_func_register(KUBECTL_ADMIN)
def kubectl_uncordon(args: str) -> str:
    """
    Run a `kubectl uncordon` command and return the output.

    Args:
        args (str): The name of the node to mark as schedulable.
                       For example:
                       - "node-1" to mark node-1 as schedulable.

    Returns:
        str: The output of the `kubectl uncordon` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl uncordon` prefix; it is added automatically.
    """
    return _kubectl("kubectl uncordon", args)


@kubectl_func_register(KUBECTL_ADMIN)
def kubectl_drain(args: str) -> str:
    """
    Run a `kubectl drain` command and return the output.

    Args:
        args (str): The specific node and options to pass to `kubectl drain`.
                       For example:
                       - "node-1 --ignore-daemonsets" to drain node-1.

    Returns:
        str: The output of the `kubectl drain` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl drain` prefix; it is added automatically.
    """
    return _kubectl("kubectl drain", args)


@kubectl_func_register(KUBECTL_ADMIN)
def kubectl_taint(args: str) -> str:
    """
    Run a `kubectl taint` command and return the output.

    Args:
        args (str): The specific node and options to pass to `kubectl taint`.
                       For example:
                       - "node-1 key=value:NoSchedule" to add a taint.
                       - "node-1 key:NoSchedule-" to remove a taint.

    Returns:
        str: The output of the `kubectl taint` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl taint` prefix; it is added automatically.
    """
    return _kubectl("kubectl taint", args)


# ----- Troubleshooting and Debugging Commands -----


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_describe(args: str) -> str:
    """
    Run a `kubectl describe` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl describe`.
                       For example:
                       - "pod nginx" to describe the nginx pod.
                       - "deployment app -n production" to describe the app deployment in the production namespace.

    Returns:
        str: The output of the `kubectl describe` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl describe` prefix; it is added automatically.
    """
    return _kubectl("kubectl describe", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_logs(args: str) -> str:
    """
    Run a `kubectl logs` command and return the output.

    Args:
        args (str): The specific pod and options to pass to `kubectl logs`.
                       For example:
                       - "nginx" to get logs from the nginx pod.
                       - "nginx -c container-name" to get logs from a specific container.
                       - "nginx --tail=20" to get the last 20 lines of logs.

    Returns:
        str: The output of the `kubectl logs` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl logs` prefix; it is added automatically.
    """
    return _kubectl("kubectl logs", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_attach(args: str) -> str:
    """
    Run a `kubectl attach` command and return the output.

    Args:
        args (str): The specific pod and options to pass to `kubectl attach`.
                       For example:
                       - "nginx" to attach to the nginx pod.
                       - "nginx -c container-name" to attach to a specific container.

    Returns:
        str: The output of the `kubectl attach` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl attach` prefix; it is added automatically.
        - This command may not work properly in all environments.
    """
    return _kubectl("kubectl attach", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_exec(args: str) -> str:
    """
    Run a `kubectl exec` command and return the output.

    Args:
        args (str): The specific pod and command to pass to `kubectl exec`.
                       For example:
                       - "nginx -- ls" to list files in the nginx pod.
                       - "nginx -c container-name -- bash" to run bash in a specific container.

    Returns:
        str: The output of the `kubectl exec` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl exec` prefix; it is added automatically.
        - This command may modify container state depending on what is executed.
    """
    return _kubectl("kubectl exec", args)


# @kubectl_func_register(KUBECTL_READONLY)
# def kubectl_port_forward(args: str) -> str:
#     """
#     Run a `kubectl port-forward` command and return the output.

#     Args:
#         args (str): The specific resource and port options to pass to `kubectl port-forward`.
#                        For example:
#                        - "pod/nginx 8080:80" to forward local port 8080 to pod port 80.
#                        - "service/web 8080" to forward local port 8080 to service port.

#     Returns:
#         str: The output of the `kubectl port-forward` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl port-forward` prefix; it is added automatically.
#         - This is a long-running command that may block execution.
#     """
#     return _kubectl("kubectl port-forward", args)


# @kubectl_func_register(KUBECTL_READONLY)
# def kubectl_proxy(args: str) -> str:
#     """
#     Run a `kubectl proxy` command and return the output.

#     Args:
#         args (str): The specific options to pass to `kubectl proxy`.
#                        For example:
#                        - "--port=8080" to run the proxy on port 8080.
#                        - "--address=0.0.0.0 --port=8080" to bind to all interfaces.

#     Returns:
#         str: The output of the `kubectl proxy` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl proxy` prefix; it is added automatically.
#         - This is a long-running command that may block execution.
#     """
#     return _kubectl("kubectl proxy", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_cp(args: str) -> str:
    """
    Run a `kubectl cp` command and return the output.

    Args:
        args (str): The source and destination paths for the copy operation.
                       For example:
                       - "/path/to/local/file nginx:/path/in/container" to copy a file to a pod.
                       - "nginx:/path/in/container /path/to/local/file" to copy a file from a pod.

    Returns:
        str: The output of the `kubectl cp` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl cp` prefix; it is added automatically.
    """
    return _kubectl("kubectl cp", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_auth(args: str) -> str:
    """
    Run a `kubectl auth` command and return the output.

    Args:
        args (str): The specific options to pass to `kubectl auth`.
                       For example:
                       - "can-i create deployments" to check if the user can create deployments.
                       - "can-i list pods --namespace production" to check permissions in a specific namespace.

    Returns:
        str: The output of the `kubectl auth` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl auth` prefix; it is added automatically.
    """
    return _kubectl("kubectl auth", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_debug(args: str) -> str:
    """
    Run a `kubectl debug` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl debug`.
                       For example:
                       - "pod/nginx -it --image=busybox" to debug a pod with busybox.
                       - "node/node-1 -it --image=ubuntu" to debug a node.

    Returns:
        str: The output of the `kubectl debug` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl debug` prefix; it is added automatically.
        - This command may block and require interactive input.
    """
    return _kubectl("kubectl debug", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_events(args: str) -> str:
    """
    Run a `kubectl events` command and return the output.

    Args:
        args (str): The specific options to pass to `kubectl events`.
                       For example:
                       - "" (empty string) to list all events.
                       - "--field-selector type=Warning" to show only warning events.

    Returns:
        str: The output of the `kubectl events` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl events` prefix; it is added automatically.
    """
    return _kubectl("kubectl events", args)


# ----- Advanced Commands -----


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_diff(args: str) -> str:
    """
    Run a `kubectl diff` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl diff`.
                       For example:
                       - "-f deployment.yaml" to show differences between the current state and the file.

    Returns:
        str: The output of the `kubectl diff` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl diff` prefix; it is added automatically.
    """
    return _kubectl("kubectl diff", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_apply(args: str) -> str:
    """
    Run a `kubectl apply` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl apply`.
                       For example:
                       - "-f deployment.yaml" to apply a deployment from a file.
                       - "-f config/ --recursive" to apply all resources in the config directory.

    Returns:
        str: The output of the `kubectl apply` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl apply` prefix; it is added automatically.
    """
    return _kubectl("kubectl apply", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_patch(args: str) -> str:
    """
    Run a `kubectl patch` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl patch`.
                       For example:
                       - 'deployment/nginx -p \'{"spec":{"replicas":3}}\'' to patch a deployment.

    Returns:
        str: The output of the `kubectl patch` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl patch` prefix; it is added automatically.
    """
    return _kubectl("kubectl patch", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_replace(args: str) -> str:
    """
    Run a `kubectl replace` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl replace`.
                       For example:
                       - "-f deployment.yaml" to replace a resource using a file.

    Returns:
        str: The output of the `kubectl replace` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl replace` prefix; it is added automatically.
    """
    return _kubectl("kubectl replace", args)


# @kubectl_func_register(KUBECTL_READONLY)
# def kubectl_wait(args: str) -> str:
#     """
#     Run a `kubectl wait` command and return the output.

#     Args:
#         args (str): The specific resource and options to pass to `kubectl wait`.
#                        For example:
#                        - "--for=condition=Ready pod/nginx" to wait until a pod is ready.
#                        - "--for=delete pod/nginx --timeout=60s" to wait until a pod is deleted.

#     Returns:
#         str: The output of the `kubectl wait` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl wait` prefix; it is added automatically.
#         - This command may block execution until the condition is met or timeout occurs.
#     """
#     return _kubectl("kubectl wait", args)


# @kubectl_func_register(KUBECTL_READONLY)
# def kubectl_kustomize(args: str) -> str:
#     """
#     Run a `kubectl kustomize` command and return the output.

#     Args:
#         args (str): The specific directory or options to pass to `kubectl kustomize`.
#                        For example:
#                        - "./kustomize" to build kustomization in the ./kustomize directory.

#     Returns:
#         str: The output of the `kubectl kustomize` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl kustomize` prefix; it is added automatically.
#     """
#     return _kubectl("kubectl kustomize", args)


# # ----- Settings Commands -----


@kubectl_func_register(KUBECTL_RW)
def kubectl_label(args: str) -> str:
    """
    Run a `kubectl label` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl label`.
                       For example:
                       - "pod nginx env=prod" to label a pod.
                       - "pods --all env=prod" to label all pods.

    Returns:
        str: The output of the `kubectl label` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl label` prefix; it is added automatically.
        - Function name is kubectl_label to avoid conflict with the decorator.
    """
    return _kubectl("kubectl label", args)


@kubectl_func_register(KUBECTL_RW)
def kubectl_annotate(args: str) -> str:
    """
    Run a `kubectl annotate` command and return the output.

    Args:
        args (str): The specific resource and options to pass to `kubectl annotate`.
                       For example:
                       - "pod nginx description='my nginx'" to annotate a pod.
                       - "pods --all description='production pods'" to annotate all pods.

    Returns:
        str: The output of the `kubectl annotate` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl annotate` prefix; it is added automatically.
    """
    return _kubectl("kubectl annotate", args)


# ----- Other Commands -----


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_api_resources(args: str) -> str:
    """
    Run a `kubectl api-resources` command and return the output.

    Args:
        args (str): The specific options to pass to `kubectl api-resources`.
                       For example:
                       - "" (empty string) to list all resources.
                       - "--namespaced=true" to show only namespaced resources.
                       - "--verbs=get" to show resources that can be retrieved.

    Returns:
        str: The output of the `kubectl api-resources` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl api-resources` prefix; it is added automatically.
    """
    return _kubectl("kubectl api-resources", args)


@kubectl_func_register(KUBECTL_READONLY)
def kubectl_api_versions(args: str) -> str:
    """
    Run a `kubectl api-versions` command and return the output.

    Args:
        args (str): The specific options to pass to `kubectl api-versions`.
                       For example:
                       - "" (empty string) to list all API versions.

    Returns:
        str: The output of the `kubectl api-versions` command, or an error message if the command is invalid.

    Notes:
        - The `args` argument should not include the `kubectl api-versions` prefix; it is added automatically.
    """
    return _kubectl("kubectl api-versions", args)


# @kubectl_func_register(KUBECTL_RW)
# def kubectl_config(args: str) -> str:
#     """
#     Run a `kubectl config` command and return the output.

#     Args:
#         args (str): The specific subcommand and options to pass to `kubectl config`.
#                        For example:
#                        - "current-context" to show the current context.
#                        - "use-context my-context" to switch to a different context.
#                        - "set-context --current --namespace=default" to set the default namespace.

#     Returns:
#         str: The output of the `kubectl config` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl config` prefix; it is added automatically.
#     """
#     return _kubectl("kubectl config", args)


# @kubectl_func_register(KUBECTL_READONLY)
# def kubectl_plugin(args: str) -> str:
#     """
#     Run a `kubectl plugin` command and return the output.

#     Args:
#         args (str): The specific subcommand and options to pass to `kubectl plugin`.
#                        For example:
#                        - "list" to list available plugins.

#     Returns:
#         str: The output of the `kubectl plugin` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl plugin` prefix; it is added automatically.
#     """
#     return _kubectl("kubectl plugin", args)


# @kubectl_func_register(KUBECTL_READONLY)
# def kubectl_version(args: str) -> str:
#     """
#     Run a `kubectl version` command and return the output.

#     Args:
#         args (str): The specific options to pass to `kubectl version`.
#                        For example:
#                        - "" (empty string) to show both client and server versions.
#                        - "--client" to show only the client version.
#                        - "-o json" to get the version information in JSON format.

#     Returns:
#         str: The output of the `kubectl version` command, or an error message if the command is invalid.

#     Notes:
#         - The `args` argument should not include the `kubectl version` prefix; it is added automatically.
#     """
#     return _kubectl("kubectl version", args)

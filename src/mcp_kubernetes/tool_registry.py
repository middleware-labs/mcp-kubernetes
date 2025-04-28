from .kubectl import *

KUBECTL_READ_ONLY_TOOLS = [
    kubectl_get,
    kubectl_describe,
    kubectl_explain,
    kubectl_logs,
    kubectl_api_resources,
    kubectl_api_versions,
    kubectl_diff,
    kubectl_cluster_info,
    kubectl_top,
    kubectl_events,
    kubectl_auth,
]

KUBECTL_RW_TOOLS = [
    kubectl_create,
    kubectl_delete,
    kubectl_apply,
    kubectl_expose,
    kubectl_run,
    kubectl_set,
    kubectl_rollout,
    kubectl_scale,
    kubectl_autoscale,
    kubectl_label,
    kubectl_annotate,
    kubectl_patch,
    kubectl_replace,
    kubectl_cp,
    kubectl_exec,
    kubectl_attach,
]

KUBECTL_ADMIN_TOOLS = [
    kubectl_cordon,
    kubectl_uncordon,
    kubectl_drain,
    kubectl_taint,
    kubectl_certificate,
    kubectl_debug,
]

KUBECTL_ALL_TOOLS = KUBECTL_READ_ONLY_TOOLS + KUBECTL_RW_TOOLS + KUBECTL_ADMIN_TOOLS

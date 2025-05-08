from .command import ShellProcess
from .security_validator import HELM_READ_OPERATIONS, validate_command


async def helm(command: str) -> str:
    """Run a helm command and return the output."""
    error = validate_command(command, HELM_READ_OPERATIONS, "helm")
    if error:
        return error

    process = ShellProcess(command="helm")
    output = process.run(command)
    return output

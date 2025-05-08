from .command import ShellProcess
from .security_validator import CILIUM_READ_OPERATIONS, validate_command


async def cilium(command: str) -> str:
    """Run a cilium command and return the output."""
    error = validate_command(command, CILIUM_READ_OPERATIONS, "cilium")
    if error:
        return error

    process = ShellProcess(command="cilium")
    output = process.run(command)
    return output

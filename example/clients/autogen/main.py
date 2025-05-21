import asyncio
import yaml
import os
from autogen_agentchat.agents import AssistantAgent
from autogen_agentchat.ui import Console
from autogen_agentchat.teams import RoundRobinGroupChat
from autogen_agentchat.conditions import TextMentionTermination
from autogen_ext.auth.azure import AzureTokenProvider
from autogen_ext.models.openai import AzureOpenAIChatCompletionClient
from autogen_ext.tools.mcp import StdioServerParams, mcp_server_tools

from azure.identity import DefaultAzureCredential

answer_addressed = "ANSWER-ADDRESSED"
user_input_required = "USER-INPUT-REQUIRED"

sys_prompt_agent = """
You are a Kubernetes Expert AI Agent specializing in cluster operations, troubleshooting, and maintenance. You possess deep knowledge of Kubernetes architecture, components, and best practices, with a focus on helping users resolve complex operational challenges while maintaining security and stability.
You can use the MCP Kubernetes tool to interact with the cluster and perform various operations. You are capable of executing commands, analyzing logs, and providing detailed explanations of your actions.
Your primary goal is to assist users in diagnosing and resolving issues related to Kubernetes clusters

"""

sys_prompt_critic = f"""
Judge the answer provided by k8sagent. Respond with '{answer_addressed}' if user's question is addressed; respond with '{user_input_required}' when k8sagent is not able to solve the answer, otherwise responed with 'go ahead' to aks k8sgent to go ahead.
"""


def load_config(file_path: str = ".config.yaml") -> dict:
    """Load configuration from a YAML file."""
    with open(file_path, "r") as config_file:
        return yaml.safe_load(config_file)


def validate_config(config: dict):
    """Validate the loaded configuration."""
    mcp_kubernetes_bin = config["mcp_kubernetes_bin"]
    model_config = config["model_config"]

    if not os.path.isfile(mcp_kubernetes_bin) or not os.access(
        mcp_kubernetes_bin, os.X_OK
    ):
        raise FileNotFoundError(
            f"The MCP Kubernetes binary '{mcp_kubernetes_bin}' does not exist or is not executable. Please update .config.yaml."
        )

    required_model_keys = ["deployment", "model", "api_version", "azure_endpoint"]
    missing_keys = [
        key
        for key in required_model_keys
        if key not in model_config or not model_config[key]
    ]
    if missing_keys:
        raise ValueError(
            f"The following keys are missing or empty in model_config: {', '.join(missing_keys)}. Please update .config.yaml."
        )


def initialize():
    """Initialize the application by loading and validating the configuration."""
    config = load_config()
    validate_config(config)
    return config


def get_az_model_client(model_config: dict) -> AzureOpenAIChatCompletionClient:
    """Create and return an AzureOpenAIChatCompletionClient."""
    token_provider = AzureTokenProvider(
        DefaultAzureCredential(),
        "https://cognitiveservices.azure.com/.default",
    )

    return AzureOpenAIChatCompletionClient(
        azure_deployment=model_config["deployment"],
        model=model_config["model"],
        api_version=model_config["api_version"],
        azure_endpoint=model_config["azure_endpoint"],
        azure_ad_token_provider=token_provider,
    )


async def get_tools(
    mcp_kubernetes_bin: str,
    additional_tools: str,
    allow_namespaces: str,
    readonly: bool,
) -> list:
    """Initialize and return tools for the MCP Kubernetes server."""
    server_params = StdioServerParams(
        command=mcp_kubernetes_bin,
        args=[
            f"--additional-tools={additional_tools}",
            f"--allow-namespaces={allow_namespaces}",
            "--readonly" if readonly else "",
        ],
    )

    tools = await mcp_server_tools(server_params)
    for tool in tools:
        print(f"MCP Server Tool: {tool.name}")

    print("=" * 30)

    return tools


async def interactive_mode(team):
    """Run the agent in interactive mode."""
    print("Interactive mode started. Type 'exit' to end the conversation.")

    while True:
        user_input = input("\n>>You(Message or Type 'exit' to end the conversation): ")

        # Exit condition
        if user_input.lower() == "exit":
            print("\nEnding conversation.")
            break

        await Console(
            team.run_stream(task=user_input),
        )

        print()


# Run the agent in interactive mode
async def main() -> None:
    config = initialize()

    mcp_kubernetes_bin = config["mcp_kubernetes_bin"]
    model_config = config["model_config"]
    app_config = config.get("app_config", {})

    additional_tools = app_config.get("additional_tools", "")
    allow_namespaces = app_config.get("allow_namespaces", "")
    readonly = app_config.get("readonly", False)

    az_model_client = get_az_model_client(model_config)

    tools = await get_tools(
        mcp_kubernetes_bin, additional_tools, allow_namespaces, readonly
    )

    agent = AssistantAgent(
        name="k8sagent",
        model_client=az_model_client,
        tools=tools,
        system_message=sys_prompt_agent,
        reflect_on_tool_use=True,
        model_client_stream=True,
    )

    text_termination1 = TextMentionTermination(answer_addressed)
    text_termination2 = TextMentionTermination(user_input_required)

    critic = AssistantAgent(
        "critic",
        model_client=az_model_client,
        system_message=sys_prompt_critic,
    )

    team = RoundRobinGroupChat(
        [agent, critic], termination_condition=text_termination1 | text_termination2
    )

    await interactive_mode(team)

    await az_model_client.close()


# Run the async main function
if __name__ == "__main__":
    asyncio.run(main())

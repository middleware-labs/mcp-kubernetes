import asyncio
import yaml
import os
import shutil
from autogen_core.models import UserMessage
from autogen_ext.auth.azure import AzureTokenProvider
from autogen_ext.models.openai import AzureOpenAIChatCompletionClient
from autogen_ext.tools.mcp import StdioServerParams, mcp_server_tools
from autogen_agentchat.agents import AssistantAgent
from autogen_agentchat.messages import ModelClientStreamingChunkEvent
from azure.identity import DefaultAzureCredential
from autogen_agentchat.ui import Console

# Load configuration from config.yaml
with open(".config.yaml", "r") as config_file:
    config = yaml.safe_load(config_file)

mcp_kubernetes_bin = config["mcp_kubernetes_bin"]
model_config = config["model_config"]

# Extract app configuration
app_config = config.get("app_config", {})
additional_tools = app_config.get("additional_tools", "")
allow_namespaces = app_config.get("allow_namespaces", "")
readonly = app_config.get("readonly", False)

# Validate configuration
if not os.path.isfile(mcp_kubernetes_bin) or not os.access(mcp_kubernetes_bin, os.X_OK):
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

sys_prompt = """

You are a Kubernetes Expert AI Agent specializing in cluster operations, troubleshooting, and maintenance. You possess deep knowledge of Kubernetes architecture, components, and best practices, with a focus on helping users resolve complex operational challenges while maintaining security and stability.

You can use the MCP Kubernetes tool to interact with the cluster and perform various operations. You are capable of executing commands, analyzing logs, and providing detailed explanations of your actions.
Your primary goal is to assist users in diagnosing and resolving issues related to Kubernetes clusters

"""


def get_az_model_client() -> AzureOpenAIChatCompletionClient:
    # Create the token provider
    token_provider = AzureTokenProvider(
        DefaultAzureCredential(),
        "https://cognitiveservices.azure.com/.default",
    )

    return AzureOpenAIChatCompletionClient(
        azure_deployment=model_config["deployment"],
        model=model_config["model"],
        api_version=model_config["api_version"],
        azure_endpoint=model_config["azure_endpoint"],
        azure_ad_token_provider=token_provider,  # Optional if you choose key-based authentication.
        # api_key="sk-...", # For key-based authentication.
    )


async def get_tools() -> list:
    server_params = StdioServerParams(
        command=mcp_kubernetes_bin,
        args=[
            f"--additional-tools={additional_tools}",
            f"--allow-namespaces={allow_namespaces}",
            "--readonly" if readonly else "",
        ],
    )

    # Initialize tools
    tools = await mcp_server_tools(server_params)
    for tool in tools:
        print(f"MCP Server Tool: {tool.name}")

    print("=" * 30)

    return tools


# Run the agent in interactive mode
async def main() -> None:
    az_model_client = get_az_model_client()

    tools = await get_tools()

    # Create agent with the tools
    agent = AssistantAgent(
        name="k8sagent",
        model_client=az_model_client,
        tools=tools,
        system_message=sys_prompt,
        reflect_on_tool_use=True,
        model_client_stream=True,  # Enable streaming tokens from the model client.
    )

    # Initialize the console
    print("Interactive mode started. Type 'exit' to end the conversation.")

    while True:
        # Get user input
        user_input = input("\n>>You: ")

        # Exit condition
        if user_input.lower() == "exit":
            print("\nEnding conversation.")
            break

        # Process the message and stream the response
        print("\n>>K8sAgent: ", end="", flush=False)

        # Stream the response using the task parameter
        async for message in agent.run_stream(task=user_input):
            # Skip if this is not a message with content
            if not hasattr(message, "content"):
                continue

            if not isinstance(message, ModelClientStreamingChunkEvent):
                # Skip duplicate messages
                continue

            content = message.content

            # Skip if the content looks like debug info or is empty
            if (
                content.startswith("messages=")
                or "stop_reason=" in content
                or not content.strip()
            ):
                continue

            # Print the content
            print(content, end="", flush=True)

        print()  # Add a newline after the response

    # Close the connection to the model client
    await az_model_client.close()


# Run the async main function
if __name__ == "__main__":
    asyncio.run(main())

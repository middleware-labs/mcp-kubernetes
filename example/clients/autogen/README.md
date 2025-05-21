# Example Project for Kubernetes Debugger

This project demonstrates how to use `mcp-kubernetes` to solve Kubernetes-related issues. It leverages `autogen` to create an agent that can easily interact with `mcp`.

## Environment Setup

Follow these steps to set up your environment:

1. **Ensure Python 3.12 or Higher**
   Make sure you have Python 3.12 or a later version installed on your system.

2. **Create a Virtual Environment**
   Use `uv` to create a virtual environment:
   ```bash
   uv venv .venv
   ```

3. **Activate the Virtual Environment**
   Activate the virtual environment:
   ```bash
   source .venv/bin/activate
   ```

4. **Sync Dependencies**
   Install the required dependencies listed in `uv.lock`:
   ```bash
   uv sync
   ```

5. **Set Up Configuration**
   Copy the `.config.template.yaml` file to `.config.yaml` and update it with your environment-specific values:
   ```bash
   cp .config.template.yaml .config.yaml
   ```
   Edit `.config.yaml` to set the correct values for `mcp_kubernetes_bin` and `model_config`.

6. **Run the Application**
   Execute the main script to start the application:
   ```bash
   python main.py
   ```

This will launch the application and allow you to begin debugging Kubernetes issues.



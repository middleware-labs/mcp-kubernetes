# Makefile for mcp-kubernetes project

# Variables
PYTHON := python3
UV := uv
DOCKER := docker
IMAGE_NAME := mcp-kubernetes
IMAGE_TAG := latest
REGISTRY := ghcr.io
OWNER := $(shell git config --get remote.origin.url | sed -n 's/.*github.com[:/]\([^/]*\).*/\1/p')
FULL_IMAGE := $(REGISTRY)/$(OWNER)/$(IMAGE_NAME):$(IMAGE_TAG)
PYTHONPATH := PYTHONPATH="$(PWD)/src"

# Default target
.PHONY: all
all: install test build

# Install dependencies
.PHONY: install
install:
	$(UV) pip install --editable .

# Install development dependencies
.PHONY: install-dev
install-dev:
	$(UV) pip install --editable '.[dev]'

# Uninstall the package
.PHONY: uninstall
uninstall:
	$(UV) pip uninstall -y mcp-kubernetes

# Run tests
.PHONY: test
test:
	$(PYTHONPATH) $(PYTHON) -m unittest discover

# Run tests with coverage
.PHONY: coverage
coverage:
	$(UV) pip install coverage
	$(PYTHONPATH) coverage run -m unittest discover
	coverage report
	coverage xml

# Build the Python package
.PHONY: build
build:
	$(UV) build

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf build/
	rm -rf dist/
	rm -rf *.egg-info/
	rm -rf .coverage
	rm -rf coverage.xml
	find . -type d -name __pycache__ -exec rm -rf {} +

# Run the application locally
.PHONY: run
run:
	$(PYTHONPATH) $(UV) run -m mcp_kubernetes.main

# Run with inspector for debugging
.PHONY: inspect
inspect:
	$(PYTHONPATH) npx @modelcontextprotocol/inspector $(UV) run -m mcp_kubernetes.main

# Build Docker image
.PHONY: docker-build
docker-build:
	$(DOCKER) build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Build and push Docker image with platform support
.PHONY: docker-buildx
docker-buildx:
	$(DOCKER) buildx build --platform linux/amd64,linux/arm64 -t $(FULL_IMAGE) --push .

# Run the Docker container
.PHONY: docker-run
docker-run:
	$(DOCKER) run -it --rm \
		-v $(HOME)/.kube/config:/home/mcp/.kube/config \
		$(IMAGE_NAME):$(IMAGE_TAG)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all            : Install dependencies, run tests, and build package"
	@echo "  install        : Install the package in development mode"
	@echo "  install-dev    : Install development dependencies"
	@echo "  uninstall      : Uninstall the package"
	@echo "  test           : Run all tests"
	@echo "  coverage       : Run tests with coverage report"
	@echo "  build          : Build the Python package"
	@echo "  clean          : Clean build artifacts"
	@echo "  run            : Run the application locally"
	@echo "  inspect        : Run with MCP inspector for debugging"
	@echo "  docker-build   : Build Docker image"
	@echo "  docker-buildx  : Build and push multi-platform Docker image"
	@echo "  docker-run     : Run the Docker container with local kubeconfig"
	@echo "  help           : Show this help message"
.PHONY: help update apply-templates build build-version build-all \
	list-versions list-variants clean test licenses licenses-check

# Variables
IMAGE_NAME ?= openclaw
DOCKERFILES_DIR ?= ./dockerfiles
VERSIONS_FILE ?= $(or $(XDG_CACHE_HOME),$(HOME)/.cache)/openclaw-docker/versions.json
DOCKER_USERNAME ?= $(shell echo $$DOCKER_USERNAME)

# Default versions to update (latest)
VERSIONS ?= latest

help:
	@echo "OpenClaw Dockerfile CLI (local use)"
	@echo ""
	@echo "Update targets:"
	@echo "  update              Fetch version info and generate Dockerfiles (default: latest)"
	@echo "  update VERSIONS='latest beta'  Update specific dist-tags"
	@echo "  apply-templates     Re-generate Dockerfiles from $(VERSIONS_FILE)"
	@echo ""
	@echo "Local build targets:"
	@echo "  build VERSION=x.x.x VARIANT=variant  Build one local image tag"
	@echo "  build-version VERSION=x.x.x          Build all local tags for one version"
	@echo "  build-all                            Build all local tags"
	@echo ""
	@echo "Info targets:"
	@echo "  list-versions       List available versions in $(VERSIONS_FILE)"
	@echo "  list-variants       List variants for a VERSION"
	@echo ""
	@echo "Other targets:"
	@echo "  clean               Remove generated Dockerfiles"
	@echo ""
	@echo "Scope note:"
	@echo "  This Makefile does not publish or push images to registries."
	@echo ""
	@echo "Examples:"
	@echo "  make update"
	@echo "  make update VERSIONS='latest beta'"
	@echo "  make build VERSION=<version> VARIANT=alpine3.23"
	@echo "  make build-version VERSION=<version>"
	@echo "  make build-all"

# Update versions manifest and generate Dockerfiles
update:
	@echo "Updating versions: $(VERSIONS)"
	go run . $(foreach v,$(VERSIONS),--version $(v)) --output $(DOCKERFILES_DIR) --versions-file $(VERSIONS_FILE)

# Re-apply templates without fetching new version info
apply-templates:
	@echo "Generating Dockerfiles from versions manifest..."
	go run . render --versions-file $(VERSIONS_FILE) --output $(DOCKERFILES_DIR)

# Build a specific version/variant
build:
ifndef VERSION
	$(error VERSION is required. Usage: make build VERSION=x.x.x VARIANT=variant)
endif
ifndef VARIANT
	$(error VARIANT is required. Usage: make build VERSION=x.x.x VARIANT=variant)
endif
	@if [ ! -f "$(DOCKERFILES_DIR)/$(VERSION)/$(VARIANT)/Dockerfile" ]; then \
		echo "Error: Dockerfile not found at $(DOCKERFILES_DIR)/$(VERSION)/$(VARIANT)/Dockerfile"; \
		echo "Run 'make list-variants VERSION=$(VERSION)' to see available variants"; \
		exit 1; \
	fi
	@echo "Building $(IMAGE_NAME):$(VERSION)-$(VARIANT)..."
	docker build -t $(IMAGE_NAME):$(VERSION)-$(VARIANT) \
		-f $(DOCKERFILES_DIR)/$(VERSION)/$(VARIANT)/Dockerfile .

# Build all variants for a specific version
build-version:
ifndef VERSION
	$(error VERSION is required. Usage: make build-version VERSION=x.x.x)
endif
	@if [ ! -d "$(DOCKERFILES_DIR)/$(VERSION)" ]; then \
		echo "Error: Version $(VERSION) not found in $(DOCKERFILES_DIR)"; \
		echo "Run 'make list-versions' to see available versions"; \
		exit 1; \
	fi
	@echo "Building all variants for version $(VERSION)..."
	@for variant in $$(ls $(DOCKERFILES_DIR)/$(VERSION)); do \
		echo "Building $(IMAGE_NAME):$(VERSION)-$$variant..."; \
		docker build -t $(IMAGE_NAME):$(VERSION)-$$variant \
			-f $(DOCKERFILES_DIR)/$(VERSION)/$$variant/Dockerfile . || exit 1; \
	done
	@echo "All variants for $(VERSION) built successfully!"

# Build all versions and variants
build-all:
	@echo "Building all versions and variants..."
	@for version in $$(ls $(DOCKERFILES_DIR)); do \
		for variant in $$(ls $(DOCKERFILES_DIR)/$$version); do \
			echo "Building $(IMAGE_NAME):$$version-$$variant..."; \
			docker build -t $(IMAGE_NAME):$$version-$$variant \
				-f $(DOCKERFILES_DIR)/$$version/$$variant/Dockerfile . || exit 1; \
		done; \
	done
	@echo "All images built successfully!"

# List available versions
list-versions:
	@echo "Available versions:"
	@jq -r 'keys[]' $(VERSIONS_FILE) 2>/dev/null || ls $(DOCKERFILES_DIR) 2>/dev/null || echo "No versions found. Run 'make update' first."

# List variants for a version
list-variants:
ifndef VERSION
	$(error VERSION is required. Usage: make list-variants VERSION=x.x.x)
endif
	@echo "Variants for version $(VERSION):"
	@jq -r '.["$(VERSION)"].variants | keys[]' $(VERSIONS_FILE) 2>/dev/null || \
		ls $(DOCKERFILES_DIR)/$(VERSION) 2>/dev/null || \
		echo "Version $(VERSION) not found."

# Clean generated Dockerfiles
clean:
	@echo "Removing generated Dockerfiles..."
	rm -rf $(DOCKERFILES_DIR)/*
	@echo "Cleanup complete!"

# Run unit tests
test:
	go test ./...

# ============================================================================
# License Targets
# ============================================================================

# Validate required license documentation exists
licenses:
	@echo "Validating license documentation..."
	@test -f LICENSE || (echo "ERROR: LICENSE is missing" >&2; exit 1)
	@test -f README.md || (echo "ERROR: README.md is missing" >&2; exit 1)
	@echo "License documentation is present."

# CI license check target
licenses-check:
	@$(MAKE) licenses

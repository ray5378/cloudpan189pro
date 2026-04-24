PROJECT_NAME=cloudpan189-share
MODULE_NAME=github.com/xxcheng123/cloudpan189-share
VAR_COMMIT ?= $(shell git rev-parse HEAD)
VAR_BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VAR_GIT_SUMMARY ?= $(shell git describe --tags --dirty --always)
VAR_GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)

OUTPUT_DIR=output
BINARY_NAME=share
DOCKER_IMAGE=$(PROJECT_NAME):latest

GO_IMAGE ?= golang:1.25-alpine
GO_DOCKER_WORKDIR ?= /src
GO_DOCKER_ENV ?= -e CGO_ENABLED=1
GO_CMD ?= /usr/local/go/bin/go
USE_DOCKER_GO ?= 1
DOCKER_GO_RUN = docker run --rm -v "$(CURDIR):$(GO_DOCKER_WORKDIR)" -w $(GO_DOCKER_WORKDIR) $(GO_DOCKER_ENV) $(GO_IMAGE) /bin/sh -lc
DOCKER_GO_PREPARE = if ! command -v gcc >/dev/null 2>&1; then apk add --no-cache build-base git >/dev/null; fi;

ifeq ($(USE_DOCKER_GO),1)
GO_RUNNER = $(DOCKER_GO_RUN) "$(DOCKER_GO_PREPARE) $(GO_CMD)
else
GO_RUNNER = sh -lc "go
endif

.PHONY: build build-frontend build-backend build-multi-arch clean clean-all
.PHONY: docker-build docker-run docker-stop docker-clean docker-logs
.PHONY: dev test lint help go-env

build: build-frontend build-backend
	@echo "✅ Build completed successfully!"

build-frontend:
	@echo "🎨 Building frontend..."
	@if [ -d "fe" ]; then \
		cd fe && npm install && npm run build; \
		echo "✅ Frontend build completed"; \
	else \
		echo "⚠️  Frontend directory not found, skipping..."; \
	fi

build-backend:
	@echo "🔨 Building backend... (USE_DOCKER_GO=$(USE_DOCKER_GO))"
	@mkdir -p $(OUTPUT_DIR)
	@$(GO_RUNNER) mod tidy"
	@$(GO_RUNNER) build -ldflags='-X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/main.go"
	@echo "✅ Backend build completed: $(OUTPUT_DIR)/$(BINARY_NAME)"

build-multi-arch:
	@echo "🔨 Building for multiple architectures... (USE_DOCKER_GO=$(USE_DOCKER_GO))"
	@mkdir -p $(OUTPUT_DIR)
	@$(GO_RUNNER) mod tidy"
	@echo "📦 Building for Linux AMD64..."
	@$(GO_RUNNER) env GOOS=linux GOARCH=amd64 $(GO_CMD) build -ldflags='-s -w -X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/main.go"
	@echo "📦 Building for Linux ARM64..."
	@$(GO_RUNNER) env GOOS=linux GOARCH=arm64 $(GO_CMD) build -ldflags='-s -w -X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/main.go"
	@echo "📦 Building for Linux ARMv7a..."
	@$(GO_RUNNER) env GOOS=linux GOARCH=arm GOARM=7 $(GO_CMD) build -ldflags='-s -w -X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-armv7a ./cmd/main.go"
	@echo "📦 Building for Windows AMD64..."
	@$(GO_RUNNER) env GOOS=windows GOARCH=amd64 $(GO_CMD) build -ldflags='-s -w -X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/main.go"
	@echo "📦 Building for macOS AMD64..."
	@$(GO_RUNNER) env GOOS=darwin GOARCH=amd64 $(GO_CMD) build -ldflags='-s -w -X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/main.go"
	@echo "📦 Building for macOS ARM64..."
	@$(GO_RUNNER) env GOOS=darwin GOARCH=arm64 $(GO_CMD) build -ldflags='-s -w -X $(MODULE_NAME)/configs.Commit=$(VAR_COMMIT) -X $(MODULE_NAME)/configs.BuildDate=$(VAR_BUILD_DATE) -X $(MODULE_NAME)/configs.GitSummary=$(VAR_GIT_SUMMARY) -X $(MODULE_NAME)/configs.GitBranch=$(VAR_GIT_BRANCH)' -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/main.go"
	@echo "✅ Multi-architecture build completed!"
	@ls -la $(OUTPUT_DIR)/

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@if [ -d "fe" ]; then rm -rf fe/dist fe/node_modules; fi

lint-clean:
	@echo "🧹 Cleaning linter cache..."
	golangci-lint cache clean

docker-build:
	@echo "🐳 Building Docker image..."
	docker build \
		--build-arg MODULE_NAME=$(MODULE_NAME) \
		--build-arg VAR_COMMIT=$(VAR_COMMIT) \
		--build-arg VAR_BUILD_DATE=$(VAR_BUILD_DATE) \
		--build-arg VAR_GIT_SUMMARY=$(VAR_GIT_SUMMARY) \
		--build-arg VAR_GIT_BRANCH=$(VAR_GIT_BRANCH) \
		-t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

docker-run: docker-stop
	@echo "🚀 Starting Docker container..."
	docker run -d \
		-p 12395:12395 \
		--name $(PROJECT_NAME) \
		$(DOCKER_IMAGE)
	@echo "✅ Container started: http://localhost:12395"

docker-stop:
	@echo "🛑 Stopping Docker container..."
	@docker stop $(PROJECT_NAME) 2>/dev/null || true
	@docker rm $(PROJECT_NAME) 2>/dev/null || true

docker-logs:
	@echo "📋 Docker container logs:"
	docker logs -f $(PROJECT_NAME)

docker-clean: docker-stop
	@echo "🧹 Cleaning Docker resources..."
	@docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@docker image prune -f

clean-all: clean docker-clean lint-clean
	@echo "✅ Complete cleanup finished!"

dev:
	@echo "🔧 Starting development server... (USE_DOCKER_GO=$(USE_DOCKER_GO))"
	@$(GO_RUNNER) run ./cmd/main.go"

test:
	@echo "🧪 Running tests... (USE_DOCKER_GO=$(USE_DOCKER_GO))"
	@$(GO_RUNNER) test -v ./..."

lint:
	@echo "Running linter..."
	golangci-lint run

go-env:
	@echo "🐹 Go build env"
	@$(GO_RUNNER) version"

swag-init:
	@echo "📚 Generating Swagger documentation..."
	swag init -g cmd/main.go -o internal/docs --parseDependency --parseInternal
	@echo "✅ Swagger documentation generated in internal/docs/"

swag-fmt:
	@echo "🎨 Formatting Swagger comments..."
	swag fmt
	@echo "✅ Swagger comments formatted"

info:
	@echo "📊 Build Information:"
	@echo "  Project: $(PROJECT_NAME)"
	@echo "  Module:  $(MODULE_NAME)"
	@echo "  Commit:  $(VAR_COMMIT)"
	@echo "  Date:    $(VAR_BUILD_DATE)"
	@echo "  Summary: $(VAR_GIT_SUMMARY)"
	@echo "  Branch:  $(VAR_GIT_BRANCH)"
	@echo "  Output:  $(OUTPUT_DIR)/$(BINARY_NAME)"
	@echo "  USE_DOCKER_GO: $(USE_DOCKER_GO)"
	@echo "  GO_IMAGE: $(GO_IMAGE)"
	@echo ""
	@echo "🏗️ Supported Architectures:"
	@echo "  - linux/amd64"
	@echo "  - linux/arm64"
	@echo "  - linux/arm/v7 (ARMv7a)"
	@echo "  - windows/amd64"
	@echo "  - darwin/amd64"
	@echo "  - darwin/arm64"

help:
	@echo "🚀 Available commands:"
	@echo ""
	@echo "📦 Build Commands:"
	@echo "  build            - Build frontend and backend (默认 Docker Go)"
	@echo "  build-frontend   - Build frontend only"
	@echo "  build-backend    - Build backend only (默认 Docker Go)"
	@echo "  build-multi-arch - Build for multiple architectures (默认 Docker Go)"
	@echo "  go-env           - Show active Go version in current build env"
	@echo ""
	@echo "🔧 Development Commands:"
	@echo "  dev              - Start development server (默认 Docker Go)"
	@echo "  test             - Run tests (默认 Docker Go)"
	@echo "  lint             - Run golangci-lint"
	@echo ""
	@echo "🛠️ Overrides:"
	@echo "  USE_DOCKER_GO=0 make build-backend"
	@echo "  GO_IMAGE=golang:1.25-alpine make build-backend"
	@echo ""
	@echo "🐳 Docker Commands:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  docker-stop      - Stop Docker container"
	@echo "  docker-logs      - Show container logs"
	@echo "  docker-clean     - Clean Docker resources"
	@echo ""
	@echo "🧹 Cleanup Commands:"
	@echo "  clean            - Clean build artifacts"
	@echo "  lint-clean       - Clean linter cache"
	@echo "  clean-all        - Complete cleanup"
	@echo ""
	@echo "📚 Docs:"
	@echo "  swag-init        - Generate Swagger docs"
	@echo "  swag-fmt         - Format Swagger comments"
	@echo "  info             - Show build information"

.DEFAULT_GOAL := help

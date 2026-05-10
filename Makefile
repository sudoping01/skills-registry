BINARY      := skill
CLI_IMAGE   := skillhub/skill
CLI_TAG     := latest
INSTALL_DIR := /usr/local/bin

.PHONY: help
help:
	@echo ""
	@echo "  SkillHub CLI"
	@echo ""
	@echo "  Setup"
	@echo "    make install       Build and install the CLI to $(INSTALL_DIR)"
	@echo "    make uninstall     Remove the CLI from $(INSTALL_DIR)"
	@echo ""
	@echo "  Development"
	@echo "    make build         Build the CLI binary to ./bin/skill"
	@echo "    make test          Run all tests"
	@echo "    make clean         Remove build artifacts"
	@echo ""
	@echo "  Docker"
	@echo "    make docker-build  Build the CLI Docker image"
	@echo "    make docker-push   Push the CLI Docker image"
	@echo ""
	@echo "  Release"
	@echo "    make release       Build binaries for linux/mac/windows"
	@echo ""

.PHONY: install
install: build
	@echo "→ Installing skill to $(INSTALL_DIR)..."
	cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "✓ Installed: $(INSTALL_DIR)/$(BINARY)"
	@echo "  Run: skill --help"

.PHONY: uninstall
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "✓ Uninstalled"

.PHONY: build
build:
	@mkdir -p bin
	@echo "→ Building CLI..."
	go build -o bin/$(BINARY) ./cmd/skill
	@echo "✓ Built: ./bin/$(BINARY)"

.PHONY: test
test:
	go test ./... -v

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: docker-build
docker-build:
	docker build -f Dockerfile.cli -t $(CLI_IMAGE):$(CLI_TAG) .
	@echo "✓ CLI image built: $(CLI_IMAGE):$(CLI_TAG)"

.PHONY: docker-push
docker-push:
	docker push $(CLI_IMAGE):$(CLI_TAG)

.PHONY: release
release:
	@mkdir -p dist
	@echo "→ Building release binaries..."
	GOOS=linux   GOARCH=amd64 go build -o dist/skill-linux-amd64    ./cmd/skill
	GOOS=linux   GOARCH=arm64 go build -o dist/skill-linux-arm64    ./cmd/skill
	GOOS=darwin  GOARCH=amd64 go build -o dist/skill-darwin-amd64   ./cmd/skill
	GOOS=darwin  GOARCH=arm64 go build -o dist/skill-darwin-arm64   ./cmd/skill
	GOOS=windows GOARCH=amd64 go build -o dist/skill-windows-amd64.exe ./cmd/skill
	@echo "✓ Release binaries in ./dist/"
	@ls -lh dist/

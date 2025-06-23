.PHONY: build clean windows macos all
BINARY_NAME=public_docs_server
VERSION=1.0.0
BUILD_DIR=bin
DIST_DIR=dist
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	mkdir -p $(BUILD_DIR)
	mkdir -p $(DIST_DIR)
build: clean
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .
windows: clean
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY_NAME)_windows_amd64.exe .
macos: clean
	GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY_NAME)_darwin_amd64 .
macos-arm64: clean
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/$(BINARY_NAME)_darwin_arm64 .
linux: clean
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY_NAME)_linux_amd64 .
linux-arm64: clean
	GOOS=linux GOARCH=arm64 go build -o $(DIST_DIR)/$(BINARY_NAME)_linux_arm64 .
all: clean windows macos macos-arm64 linux linux-arm64
run:
	go run . .
test:
	go test ./...
deps:
	go mod tidy
	go mod download
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)
release: all
	@echo "Creating release packages..."
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)_windows_amd64.tar.gz $(BINARY_NAME)_windows_amd64.exe
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)_darwin_amd64.tar.gz $(BINARY_NAME)_darwin_amd64
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)_darwin_arm64.tar.gz $(BINARY_NAME)_darwin_arm64
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)_linux_amd64.tar.gz $(BINARY_NAME)_linux_amd64
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)_linux_arm64.tar.gz $(BINARY_NAME)_linux_arm64
	@echo "Release packages created in $(DIST_DIR)/"
list:
	@echo "Available binaries:"
	@ls -la $(DIST_DIR)/ 2>/dev/null || echo "No binaries found. Run 'make all' to build." 
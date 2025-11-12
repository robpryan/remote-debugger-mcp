build:
	@echo "Building the project..."
	@go build -o build/go-debugger-mcp ./cmd/debugger/*.go

build-dir:
	@if [ ! -d build/ ]; then mkdir -p build; fi

test: build-dir
	@echo "Running tests with coverage..."
	@go test -v -short -race -coverprofile=build/coverage.out ./...
	@go tool cover -html=build/coverage.out -o build/coverage.html

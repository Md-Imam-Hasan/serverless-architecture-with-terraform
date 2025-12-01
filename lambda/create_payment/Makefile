.PHONY: build clean test lint

# Build the Lambda function for Linux ARM64 (default for AWS Lambda)
build:
	cd src && GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap main.go

# Build for AMD64 (alternative architecture)
build-amd64:
	cd src && GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap main.go

# Clean build artifacts
clean:
	rm -f bootstrap

# Run tests
test:
	go test -v ./...

# Run linting
lint:
	gofmt -l .
	go vet ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run all quality checks
check: lint test

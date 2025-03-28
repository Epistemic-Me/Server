# Start with the official Golang 1.22 image
FROM golang:1.22.4-alpine AS base

RUN apk update && apk add --no-cache make protobuf-dev git

# Install Go plugins for protoc and connectrpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2 && \
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# Add GOPATH/bin to PATH
ENV PATH=$PATH:/go/bin

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Allows us to fetch submodules via HTTPS vs SSH in dockerfile
RUN sed -i 's|git@github.com:|https://github.com/|' .gitmodules && \
    git submodule sync && \
    git submodule update --init --recursive

# Delete any trailing generated protobufs
RUN rm -rf pb


# Generate protobufs
RUN find ./proto -name "*.proto" -print0 | xargs -0 protoc \
  --proto_path=./ \
  --go_out=. --go_opt=module=epistemic-me-core \
  --go-grpc_out=. --go-grpc_opt=module=epistemic-me-core \
  --connect-go_out=. --connect-go_opt=module=epistemic-me-core

# Fix import paths in generated Go files
RUN find . -type f -name '*.go' -exec sh -c 'if grep -l "epistemic-me-core/pb/" "$1"; then echo "=== $1 ==="; grep "epistemic-me-core/pb/" "$1"; fi' sh {} \;

# Fix import paths in generated Go files - handle all cases while preserving valid paths
RUN for file in $(find . -type f -name '*.go'); do \
    # Fix base pb import
    sed -i 's|pb "epistemic-me-core/pb/"|pb "epistemic-me-core/pb"|g' $file; \
    # Fix other imports while preserving models and pbconnect
    sed -i '/pb\/models/!{/pb\/pbconnect/!{s|"epistemic-me-core/pb/"|"epistemic-me-core/pb"|g}}' $file; \
    done

# Print remaining problematic imports for debugging
RUN find . -type f -name '*.go' -exec sh -c 'if grep "epistemic-me-core/pb/" "$1" | grep -v "pb/models\|pb/pbconnect" > /dev/null; then echo "=== $1 ==="; grep "epistemic-me-core/pb/" "$1" | grep -v "pb/models\|pb/pbconnect"; fi' sh {} \;

# Verify no trailing slashes remain in imports (excluding valid paths)
RUN find . -type f -name '*.go' -exec sh -c 'if grep "epistemic-me-core/pb/" "$1" | grep -v "pb/models\|pb/pbconnect" > /dev/null; then echo "Found problematic import in $1:"; grep "epistemic-me-core/pb/" "$1" | grep -v "pb/models\|pb/pbconnect"; exit 1; fi' sh {} \;

# Add an explicit check of the go.mod file
RUN cat go.mod

# Clean go mod cache and remove any cached packages
RUN go clean -modcache && rm -rf /go/pkg/mod/epistemic-me-core*

# Tidy up the dependencies
RUN go mod tidy

# =============================
# Base Test Stage
# =============================
FROM base AS basetest

WORKDIR /app

# Create a symbolic link to make /Server point to /app
# Tests expect kvStore to be in the Server directory
RUN ln -s /app /Server

# =============================
# SDK Test Stage
# =============================
FROM basetest AS sdktest

# Command to run SDK tests
CMD ["go", "test", "-v", "./tests/sdk/..."]

# =============================
# Integration Test Stage
# =============================
FROM basetest AS inttest

# Command to run unit tests
CMD ["go", "test", "-v", "./tests/integration/..."]


# =============================
# Development Stage
# =============================
FROM base AS dev

# Set the working directory
WORKDIR /app

# Build the Go app
RUN go build -o main .

# Expose ports 8080
EXPOSE 8080

# Command to run the application
CMD ["./main"]


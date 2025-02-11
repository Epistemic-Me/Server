#!/bin/bash

# Function to check if Docker daemon is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        echo "Docker daemon is not running"
        return 1
    fi
    return 0
}

# Function to start Docker on macOS
start_docker_macos() {
    echo "Attempting to start Docker daemon..."
    if [ -x "/Applications/Docker.app/Contents/MacOS/Docker" ]; then
        open -a Docker
        echo "Waiting for Docker daemon to start..."
        while ! docker info > /dev/null 2>&1; do
            echo "."
            sleep 1
        done
        echo "Docker daemon started"
    else
        echo "Docker.app not found in Applications"
        return 1
    fi
}

# Step 1: Generate protobuf files
echo "Generating protobuf files..."
# Remove existing protobuf generated files
rm -rf pb

# Find all .proto files and generate Go, gRPC, and Connect code
find ./proto -name "*.proto" -print0 | xargs -0 protoc \
  --proto_path=./ \
  --go_out=. --go_opt=module=epistemic-me-core \
  --go-grpc_out=. --go-grpc_opt=module=epistemic-me-core \
  --connect-go_out=. --connect-go_opt=module=epistemic-me-core

echo "Protobuf files generated in ./pb"

# Step 2: Fix import paths in generated files
echo "Fixing import paths..."
for file in $(find pb -type f -name '*.go'); do
  sed 's|pb "epistemic-me-core/pb/"|pb "epistemic-me-core/pb"|' $file > temp.go
  mv temp.go $file
done

# Step 3: Build the application
echo "Building application..."

# Check if running on macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Check if Docker is running
    if ! check_docker; then
        # Try to start Docker
        if ! start_docker_macos; then
            echo "Failed to start Docker daemon"
            exit 1
        fi
    fi
fi

# Build the Docker image
echo "Building Docker image..."
docker build -t epistemic-me-core .

echo "Build completed successfully!" 
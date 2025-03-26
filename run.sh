#!/bin/zsh

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Test"

# Function to build the Docker image
build_image() {
    echo "Building Docker image..."
    docker build --build-arg OPENAI_API_KEY="$OPENAI_API_KEY" -t epistemic-me-core .
}

# Function to run integration tests
run_tests() {
    echo "Running tests..."
    # Wait for server to be ready
    while ! curl -s http://localhost:8080 > /dev/null; do
        echo "Waiting for server to be ready..."
        sleep 1
    done
    
    # Export the OPENAI_API_KEY for tests
    export OPENAI_API_KEY="$OPENAI_API_KEY"
    
    local test_failed=false
    
    # Run the integration tests if --test or --integration-test flag is set
    if [ "$TEST_MODE" = true ] || [ "$INTEGRATION_TEST_MODE" = true ]; then
        echo "Running integration tests..."
        (cd "$SCRIPT_DIR/tests/integration" && OPENAI_API_KEY="$OPENAI_API_KEY" go test -v ./...)
        INTEGRATION_TEST_EXIT_CODE=$?
        if [ $INTEGRATION_TEST_EXIT_CODE -ne 0 ]; then
            echo "Integration tests failed with exit code $INTEGRATION_TEST_EXIT_CODE"
            test_failed=true
        else
            echo "Integration tests passed successfully"
        fi
    fi
    
    # Run the SDK tests if --test or --sdk-test flag is set
    if [ "$TEST_MODE" = true ] || [ "$SDK_TEST_MODE" = true ]; then
        echo "Running SDK tests..."
        (cd "$SCRIPT_DIR/tests/sdk" && OPENAI_API_KEY="$OPENAI_API_KEY" go test -v ./...)
        SDK_TEST_EXIT_CODE=$?
        if [ $SDK_TEST_EXIT_CODE -ne 0 ]; then
            echo "SDK tests failed with exit code $SDK_TEST_EXIT_CODE"
            test_failed=true
        else
            echo "SDK tests passed successfully"
        fi
    fi
    
    # Handle test failures in non-daemon mode
    if [ "$test_failed" = true ] && [ "$DAEMON_MODE" = false ]; then
        docker stop epistemic-me-core
        exit 1
    fi
    
    if [ "$test_failed" = false ]; then
        if [ "$TEST_MODE" = true ]; then
            echo "All tests passed successfully"
        elif [ "$INTEGRATION_TEST_MODE" = true ]; then
            echo "Integration tests completed successfully"
        elif [ "$SDK_TEST_MODE" = true ]; then
            echo "SDK tests completed successfully"
        fi
    fi
}

# Function to run the container
run_container() {
    local LIVE_RELOAD=$1
    echo "Starting container..."
    # Stop any existing container
    docker stop epistemic-me-core 2>/dev/null || true
    docker rm epistemic-me-core 2>/dev/null || true
    
    if [ "$LIVE_RELOAD" = true ]; then
        # Run with volume mount for live reload
        docker run --name epistemic-me-core -p 8080:8080 \
            -v "$(pwd):/app" \
            -e OPENAI_API_KEY="$OPENAI_API_KEY" \
            epistemic-me-core &
    else
        # Run normally without volume mount
        docker run --name epistemic-me-core -p 8080:8080 \
            -e OPENAI_API_KEY="$OPENAI_API_KEY" \
            epistemic-me-core &
    fi

    # If test mode is enabled, run tests
    if [ "$TEST_MODE" = true ]; then
        run_tests
    fi
}

# Function to watch for changes and rebuild
watch_and_rebuild() {
    echo "Watching for changes..."
    while true; do
        if find . -type f -name "*.go" -o -name "*.proto" -o -name "Dockerfile" | 
           entr -d echo "Change detected"; then
            echo "Rebuilding due to changes..."
            "$SCRIPT_DIR/build.sh"
            run_container true
            
            # Run tests again if in test mode
            if [ "$TEST_MODE" = true ]; then
                run_tests
            fi
        fi
    done
}

# Check if entr is installed when in daemon mode
check_entr() {
    if ! command -v entr >/dev/null 2>&1; then
        echo "Installing entr for file watching..."
        if [[ "$OSTYPE" == "darwin"* ]]; then
            brew install entr
        else
            echo "Please install 'entr' manually for your system"
            exit 1
        fi
    fi
}

# Read the OPENAI_API_KEY from .env file
if [ ! -f "$PROJECT_ROOT/.env" ]; then
    echo "Error: .env file not found in $PROJECT_ROOT"
    exit 1
fi

OPENAI_API_KEY=$(grep OPENAI_API_KEY "$PROJECT_ROOT/.env" | cut -d '=' -f2)
if [ -z "$OPENAI_API_KEY" ]; then
    echo "Error: OPENAI_API_KEY not found in .env file"
    exit 1
fi

# Parse command line arguments
DAEMON_MODE=false
TEST_MODE=false
SDK_TEST_MODE=false
INTEGRATION_TEST_MODE=false

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --daemon) 
            DAEMON_MODE=true
            check_entr
            ;;
        --test)
            TEST_MODE=true
            ;;
        --sdk-test)
            SDK_TEST_MODE=true
            ;;
        --integration-test)
            INTEGRATION_TEST_MODE=true
            ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Initial build using build.sh
"$SCRIPT_DIR/build.sh"

if [ "$DAEMON_MODE" = true ]; then
    # Run in daemon mode with live updates
    run_container true
    watch_and_rebuild
else
    # Run normally without live updates
    run_container false
    
    # Run tests if any test mode is enabled
    if [ "$TEST_MODE" = true ] || [ "$SDK_TEST_MODE" = true ] || [ "$INTEGRATION_TEST_MODE" = true ]; then
        run_tests
    fi
    
    # In non-daemon mode, wait for container to finish
    wait
fi 
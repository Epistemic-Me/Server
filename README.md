# Epistemic Me Server

The Epistemic Me server is a Go-based backend service that provides belief and dialectic management capabilities through a gRPC interface.

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (`protoc`)
- OpenAI API key for AI-powered features
- Docker for containerization

### Installing Dependencies

1. Install the Protocol Buffers compiler and Go plugins:

```bash
brew install protobuf
brew install protoc-gen-go
brew install protoc-gen-go-grpc
brew install protoc-gen-connect-go
```

2. Initialize the git submodules (proto):

```bash
git submodule init && git submodule update
```

## Building and Running

### Local Development

1. Set your OpenAI API key in `.env` file at the project root:

```bash
OPENAI_API_KEY=your_api_key_here
```

2. Run the server using the `run.sh` script:

```bash
./run.sh [flags]
```

The server will start on port 8080 by default.

### Run Script (run.sh)

The `run.sh` script is the main entry point for building, running, and testing the server. Before executing any command, it automatically runs a pre-build step using `build.sh` which:
- Generates protobuf files
- Fixes import paths
- Builds the Docker image

#### Available Flags

- No flags: Just builds and runs the server
- `--daemon`: Runs in daemon mode with live reload on code changes
- `--test`: Runs both integration and SDK tests
- `--integration-test`: Runs only integration tests
- `--sdk-test`: Runs only SDK tests

#### Examples

```bash
# Just run the server
./run.sh

# Run in daemon mode with live reload
./run.sh --daemon

# Run all tests (integration and SDK)
./run.sh --test

# Run only integration tests
./run.sh --integration-test

# Run only SDK tests
./run.sh --sdk-test
```

#### Test Execution Flow

When running tests:
1. The script first builds the project using `build.sh`
2. Starts the server in a Docker container
3. Waits for the server to be ready
4. Executes the specified test suite(s)
5. Reports test results
6. Exits with appropriate status code

### Docker Deployment

1. Build the Docker image:

```bash
docker build --build-arg OPENAI_API_KEY={OPEN_API_KEY} -t epistemic-me-core .
```

2. Run the container:

```bash
docker run -p 8080:8080 epistemic-me-core
```

## Project Structure

- `proto/`: Protocol Buffer definitions
- `pb/`: Generated Go code from Protocol Buffers
- `svc/`: Core service implementations
- `server/`: gRPC server implementation
- `tests/`: Integration and unit tests
  - `integration/`: Integration tests
  - `sdk/`: SDK tests
- `ai/`: AI helper implementations
- `db/`: Database and storage implementations

## Development Workflow

1. Make changes to the Protocol Buffer definitions in `proto/`
2. Run `./build.sh` to regenerate the Go code
3. Implement your changes in the relevant service package
4. Add tests for your changes
5. Run tests to verify everything works
6. Submit your PR

## API Documentation

The API is defined using Protocol Buffers. You can find the service definitions in the `proto/` directory.

For detailed API documentation, please refer to the proto files:
- `proto/models/*.proto`: Data models
- `proto/service.proto`: Service definitions

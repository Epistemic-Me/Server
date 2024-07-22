# Start with the official Golang 1.22 image
FROM golang:1.22.4-alpine

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

# Delete any trailing generated protobufs
RUN rm -rf pb

# Generate protobufs
RUN find ./proto -name "*.proto" -print0 | xargs -0 protoc \
  --proto_path=./ \
  --go_out=. --go_opt=module=epistemic-me-backend \
  --go-grpc_out=. --go-grpc_opt=module=epistemic-me-backend \
  --connect-go_out=. --connect-go_opt=module=epistemic-me-backend

# Fix import paths in generated Go files
RUN for file in $(find pb -type f -name '*.go'); do \
  sed 's|pb "epistemic-me-backend/pb/"|pb "epistemic-me-backend/pb"|' $file > temp.go; \
  mv temp.go $file; \
  done

# Tidy up the dependencies
RUN go mod tidy

# Build the Go app
RUN go build -o main .

# Expose ports 8080 and 9090 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]

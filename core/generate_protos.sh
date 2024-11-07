#!/bin/bash

# Remove existing protobuf generated files
rm -rf pb

# Find all .proto files and generate Go, gRPC, and Connect code
find ./proto -name "*.proto" -print0 | xargs -0 protoc \
  --proto_path=./ \
  --go_out=. --go_opt=module=epistemic-me-core \
  --go-grpc_out=. --go-grpc_opt=module=epistemic-me-core \
  --connect-go_out=. --connect-go_opt=module=epistemic-me-core

echo "Protobuf files generated in ./pb"

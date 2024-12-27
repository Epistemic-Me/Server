### Backend Setup

## Summary

The current backend setup is a golang grpc server. 

Proto models are stored in the proto/ folder, and the protoc command is used to generate golang stubs for use by the grpc server. 

## Developing Locally

Setup your environment for local protobuf generation

```
brew install protoc-gen-go
brew install protoc-gen-go-grpc
```

Initialize the git submodules (proto)

```git submodule init && git submodule update```

Navigate to the root directory, and run `sh generate_protos.sh` to generate the golang files in the epistemic-me-core/ directory from the
proto files. 

## Docker Instructions

Initialize the git submodules (proto)

```git submodule init && git submodule update```

Download [Docker](https://www.docker.com/products/docker-desktop/)

Build the docker image

`docker build --build-arg OPENAI_API_KEY={OPEN_API_KEY} -t epistemic-me-core .`

Run the server and expose on port 8080 to mirror goland default port 8080

`docker run -p 8080:8080 epistemic-me-core`

Run Integration Tests Against The Server

go test -v integration_test.go

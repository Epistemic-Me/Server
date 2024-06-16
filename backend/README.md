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

Navigate to the `./backend` directory, and run `sh generate_protos.sh` to generate the golang files in the pb/ directory from the
proto files. 

## Docker Instructions

Download [Docker](https://www.docker.com/products/docker-desktop/)

Build the docker image

`docker build -t epistemic-me-backend .`

Run the server and expose on port 8080 to mirror goland default port 8080

`docker run -p 8080:8080 -p 9090:9090 epistemic-me-backend`

Use Brew on Mac OS to download the grpcurl CLI command

`brew install grpcurl`

Test the server by querying an API

`grpcurl -plaintext localhost:9090 EpistemicMeService/ListBeliefs`
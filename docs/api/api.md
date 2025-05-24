# Blobcast REST API Documentation

This directory contains the REST API documentation for the Blobcast node.

## OpenAPI Specification

The REST API is documented using the OpenAPI 3.1 standard in `openapi.yaml`.

### Viewing the Documentation

You can view the API documentation in several ways:

1. **Swagger UI** - Upload the `openapi.yaml` file to [Swagger Editor](https://editor.swagger.io/)

2. **Postman** - Import the `openapi.yaml` file directly into Postman to create a collection

3. **Scalar** - The REST API server hosts the API docs via Scalar at the `/docs` endpoint

### API Overview

The REST API provides the following endpoints:

#### System
- `GET /v1/health` - Health check endpoint

#### Chain
- `GET /v1/chain/info` - Get chain information (chain ID, finalized height, etc.)

#### Blocks
- `GET /v1/blocks/latest` - Get the latest finalized block
- `GET /v1/blocks/{heightOrHash}` - Get a block by height or hash

#### Manifests
- `GET /v1/manifests/file/{id}` - Get a file manifest
- `GET /v1/manifests/directory/{id}` - Get a directory manifest

### Running the API Server

The REST API server needs to connect to a Blobcast full node gRPC. You may connect to a public node or run your own.

To run your own Blobcast node:

```bash
# Start a blobcast node
blobcast start --verbose
```

To start the REST API server:

```bash
# In another terminal, start the REST API server
blobcast api --port 8081 --node-grpc localhost:50051
```

### Example Usage

```bash
# Check health
curl http://localhost:8081/v1/health

# Get chain info
curl http://localhost:8081/v1/chain/info

# Get latest block
curl http://localhost:8081/v1/blocks/latest

# Get block by height
curl http://localhost:8081/v1/blocks/123

# Get block by hash (with 0x prefix)
curl http://localhost:8081/v1/blocks/0x076692e52e2e607ac72aedb377fa494b55584c3145e8c1d4bd4a87675f281db5

# Get file manifest
curl http://localhost:8081/v1/manifests/file/bc46nekbubvwcyg34in9p85rfbzd38p6tmg3ep9z9asn2jw5eaefznmzv5yyql
```

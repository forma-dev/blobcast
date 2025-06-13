![Blobcast](docs/blobcast-banner.png)

> [!IMPORTANT]
> Blobcast is experimental beta software. Use at your own risk and do not rely on it for production workloads.

# Blobcast

Blobcast is a minimal based rollup for publishing and retrieving files on top of Celestia's data-availability (DA) layer.

* **Based Rollup** - ordering and finality come from Celestia headers; every Blobcast node derives identical state without a VM or separate consensus layer.
* **Content addressable** - files are chunked, hashed, and referenced through manifests with cryptographic verification.
* **Deterministic state** - nodes sync Celestia blocks to derive identical state with cryptographic integrity guarantees.
* **Simple state machine** - each Celestia block is projected into a `Block` that lists valid `Chunk`, `FileManifest`, and `DirectoryManifest` identifiers.  Merkle roots are committed to an on-disk Merkle-Mountain-Range (MMR).
* **No VM** - state transition follows a simple set of static rules without requiring a virtual machine (see [`docs/architecture.md`](docs/architecture.md))
* **gRPC + REST APIs** - nodes expose both gRPC APIs and HTTP/JSON REST APIs for integration.
* **Web gateway** - built-in HTTP server for browsing and serving files directly from manifests.

> **⚠️ Beta Software**: Blobcast is currently experimental.

---

## Active Networks

### Blobcast Testnet

| Property | Value |
| ---- | ---- |
| Chain ID | blobcast-mocha-1 |
| Explorer | https://explorer.blobcast-mocha-1.forma.art |
| Gateway | https://gateway.blobcast-mocha-1.forma.art |
| REST API | https://api.blobcast-mocha-1.forma.art |
| Node gRPC | https://grpc.blobcast-mocha-1.forma.art |

---

## Prerequisites

* A running [Celestia node](https://docs.celestia.org/nodes/light-node) (light node is sufficient)

---

## Install

<details>
<summary>Install a pre-built binary (recommended)</summary>

```bash
bash -c "$(curl -sL https://raw.githubusercontent.com/forma-dev/blobcast/refs/heads/main/install.sh)"
```
</details>

<details>
<summary>Building from source</summary>

**Prerequisites:**

* Go 1.23.6 or higher
* [Just](https://just.systems/man/en/packages.html)

```bash
# Clone, build, and install
$ git clone https://github.com/forma-dev/blobcast.git
$ cd blobcast
$ just install
```
</details>

## Quick Start: Submitting files

```bash
# Start celestia light node on mocha testnet
$ celestia light start \
    --p2p.network mocha \
    --core.ip rpc-mocha.pops.one \
    --core.port 9090 \
    --rpc.skip-auth

# Submit a directory
$ blobcast files submit --dir ./my-data --verbose

# Submit a file
$ blobcast files submit --file ./my-file.png --verbose

# View submitted files on the explorer
$ open https://explorer.blobcast-mocha-1.forma.art/

# Browse files or directories via web gateway
$ open https://gateway.blobcast-mocha-1.forma.art/bchw4geg5w2xkeh7aciat4iikpevg0ihjnagj5j1yfppw4zex0vxj8sf4lswbllv
$ open https://gateway.blobcast-mocha-1.forma.art/bckbg87lw3brjt5go3ig522yiy7udjbkofy6fhf6nuqpdz31n41389yn9rdgu1fo
```

## Quick start: Running a node

```bash
# Start celestia light node on mocha testnet
$ celestia light start \
    --p2p.network mocha \
    --core.ip rpc-mocha.pops.one \
    --core.port 9090 \
    --rpc.skip-auth

# Rapid sync from trusted node
$ blobcast node sync --remote-grpc-address https://grpc.blobcast-mocha-1.forma.art

# Start blobcast node
$ blobcast node start \
    --celestia-auth $CELESTIA_NODE_AUTH_TOKEN \
    --celestia-rpc ws://localhost:26658 \
    --verbose
```

## Quick start: Running gateway & API

```bash
# Start the web gateway (http://localhost:8080)
$ blobcast gateway start

# Start the REST API (http://localhost:8081)
$ blobcast api start

# View submitted files on the explorer
$ open https://explorer.blobcast-mocha-1.forma.art/

# Browse files or directories via local web gateway
$ open http://localhost:8080/bchw4geg5w2xkeh7aciat4iikpevg0ihjnagj5j1yfppw4zex0vxj8sf4lswbllv
$ open http://localhost:8080/bckbg87lw3brjt5go3ig522yiy7udjbkofy6fhf6nuqpdz31n41389yn9rdgu1fo
```

## Quick start: Running all services (via docker-compose)

```bash
$ docker compose -f ./docker-compose/blobcast-mocha-1.yaml up
```

See `docs/cli.md` for a full command reference.

---

## Configuration

Blobcast can be configured via command-line flags or environment variables:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `BLOBCAST_DATA_DIR` | `~/.blobcast` | Blobcast storage directory |
| `BLOBCAST_CELESTIA_NETWORK` | `mocha` | Network: `mammoth`, `mocha`, or `celestia` (mainnet) |
| `BLOBCAST_CELESTIA_NODE_RPC` | `ws://localhost:26658` | Celestia node RPC websocket endpoint |
| `BLOBCAST_CELESTIA_NODE_AUTH_TOKEN` | - | Celestia node auth token |
| `BLOBCAST_NODE_GRPC` | `127.0.0.1:50051` | gRPC address for blobcast node |

---

## Documentation

Additional design & developer docs live in the `docs/` folder:

* [`docs/architecture.md`](docs/architecture.md) - high-level architecture overview
* [`docs/api.md`](docs/api.md) - REST API
* [`docs/cli.md`](docs/cli.md) - detailed CLI reference

---

## Why no VM?

Blobcast does not attempt to be a general-purpose smart-contract platform. It has exactly one *transaction* type - the `BlobcastEnvelope`. All valid state transitions can be expressed by a simple set of static rules, making a VM unnecessary and allowing an extremely compact implementation.

---

## Contributing

Issues and pull-requests are welcome!

---

## License

MIT - see `LICENSE` for details.

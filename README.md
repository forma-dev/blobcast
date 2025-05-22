# Blobcast

Blobcast is a minimal based rollup for publishing and retrieving files on top of Celestia's data-availability (DA) layer.

* **Based Rollup** – ordering and finality come from Celestia headers; every Blobcast node derives identical state without a VM or separate consensus layer.
* **Content addressable** – files are chunked, hashed and referenced through manifests. Every cryptographic hash is verified end-to-end.
* **Self-hosted storage** – long-term availability of chunks/manifests is provided by Blobcast nodes.
* **Simple state machine** – each Celestia block is projected into a `Block` that lists valid `Chunk`, `FileManifest`, and `DirectoryManifest` identifiers.  Merkle roots are committed to an on-disk Merkle-Mountain-Range (MMR).
* **No smart-contract VM** – all state transition logic lives in Go code (see `pkg/sync/chain.go`).

---

## Quick start

```bash
# Clone & build
$ git clone https://github.com/forma-dev/blobcast.git
$ cd blobcast
$ go build -o build/bin/blobcast .

# Start a node (assumes a local celestia light node running on mocha testnet)
$ ./build/bin/blobcast start \
    --auth $CELESTIA_NODE_AUTH_TOKEN \
    --rpc  ws://localhost:26658 \
    --grpc-port 50051

# Start the gateway (http://localhost:8080)
$ ./build/bin/blobcast serve --port 8080

# Upload a directory
$ ./build/bin/blobcast upload --dir ./my-data \
    --auth $CELESTIA_NODE_AUTH_TOKEN

# Upload a file
$ ./build/bin/blobcast upload --file ./my-file.png \
    --auth $CELESTIA_NODE_AUTH_TOKEN

# Browse the file/directory on the gateway
$ open http://localhost:8080/bc48ktvz8ajakmwu2o6vkdhg4b5j1c3nveq11f0y29xkqe7hzdjmvsxdfdnco4
```

See `docs/cli.md` for a full command reference.

---

## Repository layout

```text
.
├── cmd/        – CLI commands
├── pkg/
│   ├── celestia/   – Thin wrapper around celestia-node RPC API
│   ├── sync/       – State machine, uploader & downloader logic
│   ├── state/      – Pebble-backed local databases (chain, manifests, chunks…)
│   ├── types/      – Core domain types (Block, BlobIdentifier, …)
│   └── ...
├── proto/      – Protobuf definitions
└── main.go     – Entrypoint
```

---

## Documentation

Additional design & developer docs live in the `docs/` folder:

* `docs/architecture.md` – high-level architecture overview

---

## Why no VM?

Blobcast does not attempt to be a general-purpose smart-contract platform. It has exactly one *transaction* type – the `BlobcastEnvelope`. All valid state transitions can be expressed by a simple set of static rules, making a VM unnecessary and allowing an extremely compact implementation.

---

## Contributing

Issues and pull-requests are welcome!

---

## License

MIT – see `LICENSE` for details.

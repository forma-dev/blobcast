# Command-line Reference

Blobcast CLI is organized into logical subcommands:

- **`node`** - Run a blobcast node
- **`files`** - Submit and export files/directories
- **`gateway`** - Web gateway for browsing files
- **`api`** - REST API server

All commands share a common set of *persistent* flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--verbose, -v` | off | Enable debug logging |

---

## `blobcast node`

Commands for running a blobcast node.

### `blobcast node start`

Runs a full Blobcast node: synchronizes the chain and exposes the **gRPC API**.

```
blobcast node start \
  --grpc-address 127.0.0.1 \
  --grpc-port    50051
```

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--data-dir` | `BLOBCAST_DATA_DIR` | `~/.blobcast` | Blobcast storage directory |
| `--network` | `BLOBCAST_CELESTIA_NETWORK` | `mocha` | Celestia network (`mocha`, `mammoth`, `celestia`) |
| `--celestia-auth` | `BLOBCAST_CELESTIA_NODE_AUTH_TOKEN` | – | Celestia node auth token |
| `--celestia-rpc` | `BLOBCAST_CELESTIA_NODE_RPC` | `ws://localhost:26658` | Celestia node RPC websocket endpoint |
| `--grpc-address` | – | `127.0.0.1` | gRPC server bind address |
| `--grpc-port` | – | `50051` | gRPC server port |

The first time you run `start`, a Pebble database will be created at `<data-dir>/<network>/`.

---

## `blobcast files`

Commands for submitting and exporting files/directories.

### `blobcast files submit`

Submits a file *or* an entire directory tree to Celestia. Each file is optionally
encrypted, chunked, the manifests are generated and all data is published to
Celestia.

```
# Submit a directory
blobcast files submit --dir ./my-data

# Submit a single file
blobcast files submit --file ./my-file.png
```

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--dir, -d` | – | – | Path to directory (mutually exclusive with `--file`) |
| `--file, -f` | – | – | Single file upload |
| `--max-blob-size` | – | `384KB` | Max chunk size (e.g. `1.5MB`, `1024KB`) |
| `--max-tx-size` | – | `2MB` | Max Celestia transaction size |
| `--celestia-auth` | `BLOBCAST_CELESTIA_NODE_AUTH_TOKEN` | – | Celestia node auth token |
| `--celestia-rpc` | `BLOBCAST_CELESTIA_NODE_RPC` | `ws://localhost:26658` | Celestia node RPC endpoint |
| `--key` | – | – | 32-byte AES-256 key (hex) for encryption/decryption |

On success the command prints a **Blobcast URL**:

```
blobcast://bchw4geg5w2xkeh7aciat4iikpevg0ihjnagj5j1yfppw4zex0vxj8sf4lswbllv
```

Keep this URL - it points to the manifest and is all you need to retrieve the data.

### `blobcast files export`

Exports a file or directory from Blobcast to disk.

```
blobcast files export --url blobcast://bc... --dir ./output
```

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--url, -u` | – | – | Blobcast URL of the file or directory manifest (required) |
| `--dir, -d` | – | – | Directory to export into |
| `--file, -f` | – | – | File to export |
| `--node-grpc` | `BLOBCAST_NODE_GRPC` | `127.0.0.1:50051` | gRPC address of blobcast node |
| `--key` | – | – | 32-byte AES-256 key (hex) for encryption/decryption |

The directory will be recreated exactly as it was uploaded. If the data was
encrypted you must supply the same `--key` that was used during submission.

### `blobcast files identify`

Parses a Blobcast URL and prints the decoded height & share commitment.

```
blobcast files identify --url blobcast://bchw4geg5w2xkeh7aciat4iikpevg0ihjnagj5j1yfppw4zex0vxj8sf4lswbllv
```

| Flag | Description |
|------|-------------|
| `--url, -u` | Blobcast URL to parse (required) |

---

## `blobcast gateway`

Commands for running the web gateway.

### `blobcast gateway start`

Starts a web gateway for browsing directories and files.

```
blobcast gateway start --port 8080
```

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--addr, -a` | – | `127.0.0.1` | Address to listen on |
| `--port, -p` | – | `8080` | HTTP server port |
| `--node-grpc` | `BLOBCAST_NODE_GRPC` | `127.0.0.1:50051` | gRPC address of blobcast node |

Open `http://localhost:8080/<manifest_id>` to view a file, browse a directory structure, or append the relative path to download a file directly.

---

## `blobcast api`

Commands for running the REST API server.

### `blobcast api start`

Starts the REST API server.

```
blobcast api start --port 8081
```

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--addr, -a` | – | `127.0.0.1` | Address to listen on |
| `--port, -p` | – | `8081` | REST API server port |
| `--node-grpc` | `BLOBCAST_NODE_GRPC` | `127.0.0.1:50051` | gRPC address of blobcast node |

See [`docs/api/`](api/) for API documentation.

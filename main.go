package main

import (
	"github.com/forma-dev/blobcast/cmd"
	_ "github.com/forma-dev/blobcast/cmd/api"
	_ "github.com/forma-dev/blobcast/cmd/files"
	_ "github.com/forma-dev/blobcast/cmd/gateway"
	_ "github.com/forma-dev/blobcast/cmd/indexer"
	_ "github.com/forma-dev/blobcast/cmd/node"
)

func main() {
	cmd.Execute()
}

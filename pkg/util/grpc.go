package util

import (
	"crypto/tls"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func NewGRPCClient(address string) (*grpc.ClientConn, error) {
	keepaliveParams := keepalive.ClientParameters{
		Time:                15 * time.Minute,
		Timeout:             60 * time.Second,
		PermitWithoutStream: true,
	}

	var creds credentials.TransportCredentials
	if strings.HasPrefix(address, "https://") || strings.Contains(address, ":443") {
		creds = credentials.NewTLS(&tls.Config{})
		address = strings.TrimPrefix(address, "https://")
		address = strings.TrimSuffix(address, "/")
		if !strings.Contains(address, ":") {
			address += ":443"
		}
	} else {
		creds = insecure.NewCredentials()
	}

	return grpc.NewClient(
		address,
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)), // 1GB for now
	)
}

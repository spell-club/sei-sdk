package sdk

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getGrpcOptions(endpointURL string) (string, []grpc.DialOption, error) {
	parsed, err := url.Parse(endpointURL)
	if err != nil {
		return "", nil, fmt.Errorf("url.Parse: %w", err)
	}

	username := strings.Split(parsed.Host, ".")[0]
	password := strings.Trim(parsed.Path, "/")
	grpcOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 1024)),
	}

	grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	port := 9090

	target := fmt.Sprintf("%s:%d", parsed.Hostname(), port)
	if password == "" {
		return target, grpcOpts, nil
	}

	// create rpc credentials using basicAuth implementation of credentials.PerRPCCredentials
	creds := basicAuth{username: username, password: password}
	grpcOpts = append(grpcOpts, grpc.WithPerRPCCredentials(creds))

	return target, grpcOpts, nil
}

// basicAuth is a simple implementation of credentials.PerRPCCredentials
// to support basic authentication for grpc requests
// customers may copy/paste this or implement with their own struct
type basicAuth struct {
	username string
	password string
}

func (b basicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (basicAuth) RequireTransportSecurity() bool {
	return false
}

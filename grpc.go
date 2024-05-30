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
	"google.golang.org/grpc/credentials/insecure"
)

// basicAuth is a simple implementation of credentials.PerRPCCredentials
// to support basic authentication for grpc requests
// customers may copy/paste this or implement with their own struct
type basicAuth struct {
	username string
	password string
}

func (b basicAuth) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (basicAuth) RequireTransportSecurity() bool {
	return false
}

func getGRPCConn(cfg Config) (*grpc.ClientConn, error) { //nolint:gocritic
	grpcHost := cfg.GRPCHost
	grpcDialOptions := []grpc.DialOption{grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 1024))}

	if cfg.UseBasicAuth { //nolint:gocritic
		parsed, err := url.Parse(grpcHost)
		if err != nil {
			return nil, fmt.Errorf("url.Parse: %w", err)
		}

		usernameSplitted := strings.Split(parsed.Host, ".")
		var username string
		if len(usernameSplitted) > 0 {
			username = strings.Split(parsed.Host, ".")[0]
		}
		password := strings.Trim(parsed.Path, "/")

		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))) //nolint:gosec
		grpcHost = fmt.Sprintf("%s:9090", parsed.Hostname())
		if password != "" {
			// create rpc credentials using basicAuth implementation of credentials.PerRPCCredentials
			grpcDialOptions = append(grpcDialOptions, grpc.WithPerRPCCredentials(basicAuth{username: username, password: password}))
		}
	} else if cfg.InsecureGRPC {
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	conn, err := grpc.NewClient(grpcHost, grpcDialOptions...)
	if err != nil {
		return nil, fmt.Errorf("grpc.Dial: %s", err)
	}

	return conn, nil
}

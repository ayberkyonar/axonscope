// Package eventstore reads from an Axon Server event store that uses
// Dynamic Consistency Boundaries (DCB), the event store model of Axon Framework 5.
//
// It hides all gRPC and protobuf details and only offers read operations.
package eventstore

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/ayberkyonar/axonscope/internal/eventstore/internal/axonpb"
)

// Config describes how to reach an Axon Server event store.
type Config struct {
	// Address is the host:port of Axon Server's gRPC endpoint, e.g. "localhost:8124".
	Address string
	// Context is the Axon Server context holding the events, e.g. "default".
	Context string

	// Token is the access token, needed when Axon Server has access control
	// enabled. Use a token that is only allowed to read.
	Token string

	// TLS encrypts the connection. The server's certificate is verified
	// against the system's certificate authorities, or against CACertFile.
	TLS bool
	// CACertFile is a PEM file with the certificate authority that signed the
	// server's certificate, for servers with a company-internal CA. Setting it
	// enables TLS.
	CACertFile string
}

// Store is a read-only connection to one Axon Server context.
// Close it when you are done.
type Store struct {
	cfg    Config
	conn   *grpc.ClientConn
	events axonpb.DcbEventStoreClient
}

// Open prepares a connection to the event store described by cfg.
// It does not contact the server yet; the first read does.
func Open(cfg Config) (*Store, error) {
	creds, err := transportCredentials(cfg)
	if err != nil {
		return nil, err
	}

	// Axon Server reads the context and token from headers on every call,
	// so we attach them here once instead of in every method.
	headers := metadata.Pairs("AxonIQ-Context", cfg.Context)
	if cfg.Token != "" {
		headers.Set("AxonIQ-Access-Token", cfg.Token)
	}
	withHeaders := func(ctx context.Context) context.Context {
		return metadata.NewOutgoingContext(ctx, headers)
	}

	conn, err := grpc.NewClient(cfg.Address,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any,
			cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(withHeaders(ctx), method, req, reply, cc, opts...)
		}),
		grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc,
			cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return streamer(withHeaders(ctx), desc, cc, method, opts...)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", cfg.Address, err)
	}

	return &Store{cfg: cfg, conn: conn, events: axonpb.NewDcbEventStoreClient(conn)}, nil
}

// transportCredentials returns plain-text credentials, or TLS credentials
// when cfg asks for TLS.
func transportCredentials(cfg Config) (credentials.TransportCredentials, error) {
	if !cfg.TLS && cfg.CACertFile == "" {
		return insecure.NewCredentials(), nil
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if cfg.CACertFile != "" {
		pem, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("read CA certificate: no PEM certificates found in %s", cfg.CACertFile)
		}
		tlsConfig.RootCAs = pool
	}
	return credentials.NewTLS(tlsConfig), nil
}

// Head returns the sequence number the next appended event will get.
// For an empty event store this is 0.
func (s *Store) Head(ctx context.Context) (int64, error) {
	resp, err := s.events.GetHead(ctx, &axonpb.GetHeadRequest{})
	if err != nil {
		return 0, s.rpcError("get head", err)
	}
	return resp.GetSequence(), nil
}

// Close releases the connection.
func (s *Store) Close() error {
	return s.conn.Close()
}

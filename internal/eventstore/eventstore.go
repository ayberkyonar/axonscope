// Package eventstore reads from an Axon Server event store that uses
// Dynamic Consistency Boundaries (DCB), the event store model of Axon Framework 5.
//
// It hides all gRPC and protobuf details and only offers read operations.
package eventstore

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
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
	// Axon Server picks the context from a header on every call,
	// so we attach it here once instead of in every method.
	header := metadata.Pairs("AxonIQ-Context", cfg.Context)
	withContext := func(ctx context.Context) context.Context {
		return metadata.NewOutgoingContext(ctx, header)
	}

	conn, err := grpc.NewClient(cfg.Address,
		// TODO: support TLS for production servers.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any,
			cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(withContext(ctx), method, req, reply, cc, opts...)
		}),
		grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc,
			cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return streamer(withContext(ctx), desc, cc, method, opts...)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", cfg.Address, err)
	}

	return &Store{cfg: cfg, conn: conn, events: axonpb.NewDcbEventStoreClient(conn)}, nil
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

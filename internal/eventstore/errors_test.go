package eventstore

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRPCError(t *testing.T) {
	store := &Store{cfg: Config{Address: "localhost:9999", Context: "orders"}}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "server unreachable",
			err:  status.Error(codes.Unavailable, "connection refused"),
			want: "cannot reach Axon Server at localhost:9999",
		},
		{
			name: "not a DCB context",
			err:  status.Error(codes.InvalidArgument, "[AXONIQ-2308] Operation not supported on non-DCB context"),
			want: `context "orders" is not a DCB event store`,
		},
		{
			name: "other errors keep the operation and server message",
			err:  status.Error(codes.PermissionDenied, "no access"),
			want: "get head: no access (PermissionDenied)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.rpcError("get head", tt.err).Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("got %q, want it to contain %q", got, tt.want)
			}
			if strings.Contains(got, "rpc error") {
				t.Errorf("got %q, which still contains gRPC details", got)
			}
		})
	}
}

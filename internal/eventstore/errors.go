package eventstore

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// rpcError turns an error from a call to Axon Server into one that explains
// the problem in axonscope's terms, without gRPC details. op describes the
// failed operation, for errors without a clearer explanation.
func (s *Store) rpcError(op string, err error) error {
	st := status.Convert(err)
	switch {
	case st.Code() == codes.Unavailable:
		return s.connectionError(st.Message())

	case st.Code() == codes.Unauthenticated || st.Code() == codes.PermissionDenied:
		return fmt.Errorf("access denied by Axon Server at %s: the access token is missing, invalid, "+
			"or not allowed to read context %q", s.cfg.Address, s.cfg.Context)

	// AXONIQ-2308 is Axon Server's error for "not a DCB context".
	case st.Code() == codes.InvalidArgument && strings.Contains(st.Message(), "AXONIQ-2308"):
		return fmt.Errorf("context %q is not a DCB event store: it does not exist, "+
			"or it uses the older aggregate-based format, which axonscope does not support yet", s.cfg.Context)

	default:
		return fmt.Errorf("%s: %s (%s)", op, st.Message(), st.Code())
	}
}

// connectionError explains why no connection could be made, based on the
// message gRPC gives. Unrecognized messages get the most common explanation.
func (s *Store) connectionError(message string) error {
	addr := s.cfg.Address
	switch {
	case strings.Contains(message, "first record does not look like a TLS handshake"):
		return fmt.Errorf("no TLS on Axon Server at %s: connect without TLS", addr)

	case strings.Contains(message, "x509: "):
		_, reason, _ := strings.Cut(message, "x509: ")
		return fmt.Errorf("the TLS certificate of Axon Server at %s is not trusted (%s): "+
			"if a company-internal CA signed it, pass that CA's certificate", addr, strings.TrimSuffix(reason, `"`))

	// The server closed the connection without speaking gRPC.
	case strings.Contains(message, "server preface") && !s.cfg.TLS && s.cfg.CACertFile == "":
		return fmt.Errorf("no gRPC answer from Axon Server at %s: it may require TLS, "+
			"or this is not its gRPC port (default 8124)", addr)

	default:
		return fmt.Errorf("cannot reach Axon Server at %s: is it running, and is that its gRPC port (default 8124)?", addr)
	}
}

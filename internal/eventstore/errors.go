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
		return fmt.Errorf("cannot reach Axon Server at %s: is it running, and is that its gRPC port (default 8124)?",
			s.cfg.Address)

	// AXONIQ-2308 is Axon Server's error for "not a DCB context".
	case st.Code() == codes.InvalidArgument && strings.Contains(st.Message(), "AXONIQ-2308"):
		return fmt.Errorf("context %q is not a DCB event store: it does not exist, "+
			"or it uses the older aggregate-based format, which axonscope does not support yet", s.cfg.Context)

	default:
		return fmt.Errorf("%s: %s (%s)", op, st.Message(), st.Code())
	}
}

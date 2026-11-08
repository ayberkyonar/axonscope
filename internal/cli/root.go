// Package cli implements the axonscope command-line interface.
package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Execute runs the command given on the command line.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	var cfg eventstore.Config

	root := &cobra.Command{
		Use:   "axonscope",
		Short: "Inspect Axon Server event stores (read-only)",
		// Print only the error on failure, not the full usage text.
		SilenceUsage: true,
	}

	flags := root.PersistentFlags()
	flags.StringVar(&cfg.Address, "server", "localhost:8124", "Axon Server gRPC address (host:port)")
	flags.StringVar(&cfg.Context, "context", "default", "Axon Server context")

	root.AddCommand(newHeadCmd(&cfg), newEventsCmd(&cfg))
	return root
}

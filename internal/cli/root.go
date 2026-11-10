// Package cli implements the axonscope command-line interface.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Execute runs the command given on the command line.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

// options holds the global flags, shared by all commands.
type options struct {
	store  eventstore.Config
	output string // "text" or "json"
}

func (o *options) jsonOutput() bool {
	return o.output == "json"
}

func newRootCmd() *cobra.Command {
	opts := &options{}

	root := &cobra.Command{
		Use:   "axonscope",
		Short: "Inspect Axon Server event stores (read-only)",
		// Print only the error on failure, not the full usage text.
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.output != "text" && opts.output != "json" {
				return fmt.Errorf("invalid output %q: want text or json", opts.output)
			}
			return nil
		},
	}

	flags := root.PersistentFlags()
	flags.StringVar(&opts.store.Address, "server", "localhost:8124", "Axon Server gRPC address (host:port)")
	flags.StringVar(&opts.store.Context, "context", "default", "Axon Server context")
	flags.StringVarP(&opts.output, "output", "o", "text", "output format: text or json")

	root.AddCommand(newHeadCmd(opts), newEventsCmd(opts), newEventCmd(opts), newMCPCmd(opts))
	return root
}

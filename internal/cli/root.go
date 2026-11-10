// Package cli implements the axonscope command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Execute runs the command given on the command line.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

// tokenEnv is the environment variable holding the Axon Server access token.
const tokenEnv = "AXONSCOPE_TOKEN"

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
		Long: "Inspect Axon Server event stores (read-only).\n\n" +
			"If Axon Server has access control enabled, set the " + tokenEnv + " environment\n" +
			"variable to an access token. Use a token that is only allowed to read.",
		// Print only the error on failure, not the full usage text.
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.output != "text" && opts.output != "json" {
				return fmt.Errorf("invalid output %q: want text or json", opts.output)
			}
			// Read from the environment, not a flag: flags end up in shell history.
			opts.store.Token = os.Getenv(tokenEnv)
			return nil
		},
	}

	flags := root.PersistentFlags()
	flags.StringVar(&opts.store.Address, "server", "localhost:8124", "Axon Server gRPC address (host:port)")
	flags.StringVar(&opts.store.Context, "context", "default", "Axon Server context")
	flags.BoolVar(&opts.store.TLS, "tls", false, "encrypt the connection with TLS")
	flags.StringVar(&opts.store.CACertFile, "ca-cert", "", "PEM file of the CA that signed the server's certificate (implies --tls)")
	flags.StringVarP(&opts.output, "output", "o", "text", "output format: text or json")

	root.AddCommand(newHeadCmd(opts), newEventsCmd(opts), newEventCmd(opts), newStatsCmd(opts), newTrendCmd(opts), newMCPCmd(opts))
	return root
}

// writeJSON writes v as indented JSON, for output that is one value
// rather than a stream of lines.
func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

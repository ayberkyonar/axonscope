package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func newHeadCmd(cfg *eventstore.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "head",
		Short: "Print the sequence number the next event will get",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := eventstore.Open(*cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			head, err := store.Head(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), head)
			return nil
		},
	}
}

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func newHeadCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "head",
		Short: "Print the sequence number the next event will get",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := eventstore.Open(opts.store)
			if err != nil {
				return err
			}
			defer store.Close()

			head, err := store.Head(cmd.Context())
			if err != nil {
				return err
			}

			if opts.jsonOutput() {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
					Head int64 `json:"head"`
				}{head})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), head)
			return err
		},
	}
}

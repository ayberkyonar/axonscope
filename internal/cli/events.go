package cli

import (
	"fmt"
	"io"
	"iter"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func newEventsCmd(cfg *eventstore.Config) *cobra.Command {
	var from int64
	var limit int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List events, oldest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := eventstore.Open(*cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			return printEvents(cmd.OutOrStdout(), store.Events(cmd.Context(), from), limit)
		},
	}

	cmd.Flags().Int64Var(&from, "from", 0, "sequence of the first event to show")
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum number of events to show, 0 for all")
	return cmd
}

// printEvents writes one aligned line per event, stopping after limit
// events (0 means no limit).
func printEvents(out io.Writer, events iter.Seq2[eventstore.Event, error], limit int) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	// Flushing on every return also shows the events read before an error.
	defer w.Flush()

	shown := 0
	for e, err := range events {
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			e.Sequence, e.Timestamp.Format(time.DateTime), shortName(e.Name), formatTags(e.Tags))

		shown++
		if shown == limit {
			break
		}
	}
	return nil
}

// shortName strips the package from an event name:
// "com.example.events.AccountRegistered" becomes "AccountRegistered".
func shortName(name string) string {
	return name[strings.LastIndex(name, ".")+1:]
}

// formatTags formats tags as "key=value key=value".
func formatTags(tags []eventstore.Tag) string {
	parts := make([]string, len(tags))
	for i, tag := range tags {
		parts[i] = tag.Key + "=" + tag.Value
	}
	return strings.Join(parts, " ")
}

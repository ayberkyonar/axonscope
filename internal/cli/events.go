package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func newEventsCmd(opts *options) *cobra.Command {
	var filter queryFlags
	var limit int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List events, oldest first",
		Example: "  axonscope events --name AccountRegistered\n" +
			"  axonscope events --tag userId=42 --since 7d --limit 0\n" +
			"  axonscope events -o json | jq .payload",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := filter.query(time.Now())
			if err != nil {
				return err
			}

			store, err := eventstore.Open(opts.store)
			if err != nil {
				return err
			}
			defer store.Close()

			events := firstN(store.Events(cmd.Context(), query), limit)
			if opts.jsonOutput() {
				return writeEventsJSON(cmd.OutOrStdout(), events)
			}
			return writeEventsText(cmd.OutOrStdout(), events)
		},
	}

	filter.register(cmd)
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum number of events to show, 0 for all")
	return cmd
}

// firstN stops seq after n values; 0 means no limit.
func firstN[V any](seq iter.Seq2[V, error], n int) iter.Seq2[V, error] {
	if n == 0 {
		return seq
	}
	return func(yield func(V, error) bool) {
		count := 0
		for v, err := range seq {
			if !yield(v, err) {
				return
			}
			count++
			if count == n {
				return
			}
		}
	}
}

// writeEventsText writes one aligned line per event.
func writeEventsText(out io.Writer, events iter.Seq2[eventstore.Event, error]) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	// Flushing on every return also shows the events read before an error.
	defer w.Flush()

	for e, err := range events {
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			e.Sequence, e.Timestamp.Format(time.DateTime), e.ShortName(), formatTags(e.Tags))
	}
	return nil
}

// writeEventsJSON writes one JSON object per line (JSON Lines), so output
// streams and works with line-based tools such as jq.
func writeEventsJSON(out io.Writer, events iter.Seq2[eventstore.Event, error]) error {
	enc := json.NewEncoder(out)
	for e, err := range events {
		if err != nil {
			return err
		}
		if err := enc.Encode(eventjson.From(e)); err != nil {
			return err
		}
	}
	return nil
}

// formatTags formats tags as "key=value key=value".
func formatTags(tags []eventstore.Tag) string {
	parts := make([]string, len(tags))
	for i, tag := range tags {
		parts[i] = tag.Key + "=" + tag.Value
	}
	return strings.Join(parts, " ")
}

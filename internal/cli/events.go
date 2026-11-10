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
	var query eventstore.Query
	var tags []string
	var limit int

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List events, oldest first",
		Example: "  axonscope events --name AccountRegistered\n" +
			"  axonscope events --tag userId=42 --limit 0",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if query.Tags, err = parseTags(tags); err != nil {
				return err
			}

			store, err := eventstore.Open(*cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			return printEvents(cmd.OutOrStdout(), store.Events(cmd.Context(), query), limit)
		},
	}

	flags := cmd.Flags()
	flags.Int64Var(&query.From, "from", 0, "sequence of the first event to show")
	flags.StringArrayVar(&query.Names, "name", nil, "only show events with this name, full or short (repeatable)")
	flags.StringArrayVar(&tags, "tag", nil, "only show events with this tag, as key=value (repeatable)")
	flags.IntVar(&limit, "limit", 20, "maximum number of events to show, 0 for all")
	return cmd
}

// parseTags parses tags written as "key=value".
func parseTags(raw []string) ([]eventstore.Tag, error) {
	tags := make([]eventstore.Tag, len(raw))
	for i, s := range raw {
		key, value, ok := strings.Cut(s, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid tag %q: want key=value", s)
		}
		tags[i] = eventstore.Tag{Key: key, Value: value}
	}
	return tags, nil
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
			e.Sequence, e.Timestamp.Format(time.DateTime), e.ShortName(), formatTags(e.Tags))

		shown++
		if shown == limit {
			break
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

package cli

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/insight"
)

func newStatsCmd(opts *options) *cobra.Command {
	var filter queryFlags
	var byTag string
	var top, maxEvents int

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Count events per name, or per tag value",
		Long: "Count events per name, most frequent first. With --by-tag, count them per value of a tag instead.\n\n" +
			"This reads every selected event, so on a large event store narrow it down, for example with --since.\n" +
			"Counting by tag needs each event's tags, which costs one call to Axon Server per event,\n" +
			"so it stops after --max-events events and says so.",
		Example: "  axonscope stats --since 7d\n" +
			"  axonscope stats --tag userId=42\n" +
			"  axonscope stats --since 7d --name OrderPlaced --by-tag customerId",
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

			out := cmd.OutOrStdout()
			if byTag != "" {
				stats, err := insight.GroupByTag(store.Events(cmd.Context(), query), byTag, top, maxEvents)
				if err != nil {
					return err
				}
				if opts.jsonOutput() {
					return writeJSON(out, eventjson.FromTagStats(stats))
				}
				return writeTagStatsText(out, stats)
			}

			query.SkipTags = true // counting per name needs no tags
			stats, err := insight.Summarize(store.Events(cmd.Context(), query))
			if err != nil {
				return err
			}
			if opts.jsonOutput() {
				return writeJSON(out, eventjson.FromStats(stats))
			}
			return writeStatsText(out, stats)
		},
	}

	filter.register(cmd)
	cmd.Flags().StringVar(&byTag, "by-tag", "", "count per value of this tag key, like customerId")
	cmd.Flags().IntVar(&top, "top", 10, "with --by-tag: how many of the most frequent values to show")
	cmd.Flags().IntVar(&maxEvents, "max-events", insight.DefaultMaxEvents, "with --by-tag: stop after this many events, 0 for no limit")
	return cmd
}

// writeStatsText writes a summary line, then one line per event name with
// the counts right-aligned.
func writeStatsText(out io.Writer, stats insight.Stats) error {
	if stats.Total == 0 {
		_, err := fmt.Fprintln(out, "no events")
		return err
	}

	fmt.Fprintf(out, "%d events from %s to %s\n\n",
		stats.Total, stats.First.Format(time.DateTime), stats.Last.Format(time.DateTime))

	// The first count is the largest, so it sets the column width.
	width := len(strconv.Itoa(stats.ByName[0].Count))
	for _, nc := range stats.ByName {
		if _, err := fmt.Fprintf(out, "%*d  %s\n", width, nc.Count, nc.ShortName()); err != nil {
			return err
		}
	}
	return nil
}

// writeTagStatsText writes a summary line, a warning when the counts are
// incomplete, then one line per tag value with the counts right-aligned.
func writeTagStatsText(out io.Writer, stats insight.TagStats) error {
	if stats.Total == 0 {
		_, err := fmt.Fprintln(out, "no events")
		return err
	}

	values := "values"
	if stats.Distinct == 1 {
		values = "value"
	}
	fmt.Fprintf(out, "%d events from %s to %s, %d different %s %s\n",
		stats.Total, stats.First.Format(time.DateTime), stats.Last.Format(time.DateTime), stats.Distinct, stats.Key, values)
	if stats.Partial {
		fmt.Fprintf(out, "Only the first %d events were counted. For a complete answer, narrow it down "+
			"with --since, --name or --tag, or raise --max-events.\n", stats.Total)
	}
	fmt.Fprintln(out)

	most := stats.Untagged
	if len(stats.Top) > 0 {
		most = max(most, stats.Top[0].Count)
	}
	width := len(strconv.Itoa(most))
	for _, tc := range stats.Top {
		fmt.Fprintf(out, "%*d  %s\n", width, tc.Count, tc.Value)
	}
	if stats.Untagged > 0 {
		fmt.Fprintf(out, "%*d  (no %s tag)\n", width, stats.Untagged, stats.Key)
	}
	return nil
}

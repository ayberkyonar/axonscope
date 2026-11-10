package cli

import (
	"encoding/json"
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

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Count events per name",
		Long: "Count events per name, most frequent first.\n\n" +
			"This reads every selected event, so on a large event store narrow it down, for example with --since.",
		Example: "  axonscope stats --since 7d\n" +
			"  axonscope stats --tag userId=42",
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

			query.SkipTags = true // counting needs no tags
			stats, err := insight.Summarize(store.Events(cmd.Context(), query))
			if err != nil {
				return err
			}

			if opts.jsonOutput() {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(eventjson.FromStats(stats))
			}
			return writeStatsText(cmd.OutOrStdout(), stats)
		},
	}

	filter.register(cmd)
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

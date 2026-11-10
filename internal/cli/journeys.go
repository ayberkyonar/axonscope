package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/insight"
)

// journeyExamples is how many example tag values to collect per path.
const journeyExamples = 3

func newJourneysCmd(opts *options) *cobra.Command {
	var filter queryFlags
	var jopts insight.JourneyOptions

	cmd := &cobra.Command{
		Use:   "journeys",
		Short: "Find the common paths that orders, users or other entities take",
		Long: "Find the common paths through the events. All events with the same value of the --by-tag tag form\n" +
			"one journey, like everything that happened to one order. Repeats in a row are merged, so adding\n" +
			"two or three items is the same path.\n\n" +
			"Journeys need each event's tags, which costs one call to Axon Server per event, so reading stops\n" +
			"after --max-events events and says so.",
		Example: "  axonscope journeys --by-tag orderId --since 7d\n" +
			"  axonscope journeys --by-tag orderId --since 7d --containing OrderCancelled\n" +
			"  axonscope journeys --by-tag userId --starting-with AccountRegistered",
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

			jopts.Examples = journeyExamples
			journeys, err := insight.FindJourneys(store.Events(cmd.Context(), query), jopts)
			if err != nil {
				return err
			}
			if opts.jsonOutput() {
				return writeJSON(cmd.OutOrStdout(), eventjson.FromJourneys(journeys))
			}
			return writeJourneysText(cmd.OutOrStdout(), journeys)
		},
	}

	filter.register(cmd)
	flags := cmd.Flags()
	flags.StringVar(&jopts.Key, "by-tag", "", "the tag that ties events into one journey, like orderId (required)")
	flags.StringVar(&jopts.Containing, "containing", "", "only journeys with an event of this name")
	flags.StringVar(&jopts.StartingWith, "starting-with", "", "only journeys whose first event has this name")
	flags.IntVar(&jopts.Top, "top", 10, "how many of the most common paths to show")
	flags.IntVar(&jopts.MaxEvents, "max-events", insight.DefaultMaxEvents, "stop after this many events, 0 for no limit")
	cmd.MarkFlagRequired("by-tag")
	return cmd
}

// writeJourneysText writes a summary line, a warning when the journeys are
// incomplete, then one line per path: count, share, steps, median duration
// and examples.
func writeJourneysText(out io.Writer, js insight.Journeys) error {
	if js.Matching == 0 {
		_, err := fmt.Fprintf(out, "no journeys (%d events read)\n", js.Events)
		return err
	}

	summary := fmt.Sprintf("%d events, %d journeys by %s", js.Events, js.Entities, js.Key)
	if js.Matching != js.Entities {
		summary += fmt.Sprintf(", %d matching", js.Matching)
	}
	paths := "paths"
	if js.Distinct == 1 {
		paths = "path"
	}
	fmt.Fprintf(out, "%s, %d different %s\n", summary, js.Distinct, paths)
	if js.Partial {
		fmt.Fprintf(out, "Only the first %d events were read, so journeys are incomplete. For a complete answer, "+
			"narrow it down with --since, --name or --tag, or raise --max-events.\n", js.Events)
	}
	fmt.Fprintln(out)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, j := range js.Top {
		share := 100 * j.Count / js.Matching
		fmt.Fprintf(w, "%d\t%d%%\t%s\tmedian %s\te.g. %s\n",
			j.Count, share, formatSteps(j), j.MedianDuration.Round(time.Second), strings.Join(j.Examples, ", "))
	}
	if js.Other > 0 {
		fmt.Fprintf(w, "%d\t%d%%\t(other paths)\n", js.Other, 100*js.Other/js.Matching)
	}
	return w.Flush()
}

// formatSteps writes a path like "Started → ItemAdded (repeated) → Placed".
func formatSteps(j insight.Journey) string {
	names := make([]string, len(j.Steps), len(j.Steps)+1)
	for i, s := range j.Steps {
		names[i] = eventstore.ShortName(s.Name)
		if s.Repeated {
			names[i] += " (repeated)"
		}
	}
	if j.MoreSteps {
		names = append(names, "…")
	}
	return strings.Join(names, " → ")
}

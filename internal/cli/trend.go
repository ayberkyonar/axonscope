package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/insight"
)

func newTrendCmd(opts *options) *cobra.Command {
	var filter queryFlags
	var by, timeZone string

	cmd := &cobra.Command{
		Use:   "trend",
		Short: "Count events per hour, day or week",
		Long: "Count events per hour, day or week, including periods without events.\n\n" +
			"Days and weeks (starting Monday) follow the calendar of your local time zone, or of --timezone.\n" +
			"This reads every selected event, so narrow it down with --since.",
		Example: "  axonscope trend --since 7d\n" +
			"  axonscope trend --since 24h --by hour --name OrderPlaced\n" +
			"  axonscope trend --since 8w --by week --timezone Europe/Amsterdam",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			unit, err := insight.ParseUnit(by)
			if err != nil {
				return fmt.Errorf("--by: %w", err)
			}
			loc := time.Local
			if timeZone != "" {
				if loc, err = time.LoadLocation(timeZone); err != nil {
					return fmt.Errorf("--timezone: unknown time zone %q: want a name like Europe/Amsterdam", timeZone)
				}
			}
			// Read dates in --since and --until in the trend's time zone, so they line up with its days.
			now := time.Now().In(loc)
			query, err := filter.query(now)
			if err != nil {
				return err
			}

			store, err := eventstore.Open(opts.store)
			if err != nil {
				return err
			}
			defer store.Close()

			query.SkipTags = true // counting needs no tags
			periods := insight.Periods{Unit: unit, Location: loc, From: query.Since, To: query.Until}
			if periods.To.IsZero() {
				periods.To = now // show the periods up to now, also when they are empty
			}
			buckets, err := insight.Trend(store.Events(cmd.Context(), query), periods)
			if err != nil {
				return err
			}

			if opts.jsonOutput() {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(eventjson.FromTrend(unit, loc, buckets))
			}
			return writeTrendText(cmd.OutOrStdout(), unit, buckets)
		},
	}

	filter.register(cmd)
	cmd.Flags().StringVar(&by, "by", "day", "period length: hour, day or week")
	cmd.Flags().StringVar(&timeZone, "timezone", "", "time zone for day and week boundaries, like Europe/Amsterdam (default: local)")
	return cmd
}

// barWidth is the length of the bar for the busiest period.
const barWidth = 30

// writeTrendText writes one line per period: its start, its count and a bar
// scaled to the busiest period.
func writeTrendText(out io.Writer, unit insight.Unit, buckets []insight.Bucket) error {
	if len(buckets) == 0 {
		_, err := fmt.Fprintln(out, "no events")
		return err
	}

	most := 0
	for _, b := range buckets {
		most = max(most, b.Total)
	}
	width := len(strconv.Itoa(most))

	for _, b := range buckets {
		bar := ""
		if b.Total > 0 {
			// Every period with events gets at least one block, so it never looks empty.
			bar = strings.Repeat("█", max(1, b.Total*barWidth/most))
		}
		line := strings.TrimRight(fmt.Sprintf("%s  %*d  %s", periodLabel(unit, b.Start), width, b.Total, bar), " ")
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func periodLabel(unit insight.Unit, start time.Time) string {
	switch unit {
	case insight.Hour:
		return start.Format("Mon 2006-01-02 15:04")
	case insight.Week:
		return "week of " + start.Format(time.DateOnly)
	default:
		return start.Format("Mon 2006-01-02")
	}
}

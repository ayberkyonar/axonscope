package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/timespec"
)

// queryFlags are the flags that select events, shared by the commands that read them.
type queryFlags struct {
	from         int64
	names        []string
	tags         []string
	since, until string
}

func (f *queryFlags) register(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.Int64Var(&f.from, "from", 0, "sequence of the first event")
	flags.StringArrayVar(&f.names, "name", nil, "only events with this name, full or short (repeatable)")
	flags.StringArrayVar(&f.tags, "tag", nil, "only events with this tag, as key=value (repeatable)")
	flags.StringVar(&f.since, "since", "", "only events at or after this time: "+timespec.Help)
	flags.StringVar(&f.until, "until", "", "only events before this time, same formats as --since")
}

// query turns the flags into a query. Relative times count back from now.
func (f *queryFlags) query(now time.Time) (eventstore.Query, error) {
	tags, err := parseTags(f.tags)
	if err != nil {
		return eventstore.Query{}, err
	}
	since, err := timespec.Parse(f.since, now)
	if err != nil {
		return eventstore.Query{}, fmt.Errorf("--since: %w", err)
	}
	until, err := timespec.Parse(f.until, now)
	if err != nil {
		return eventstore.Query{}, fmt.Errorf("--until: %w", err)
	}
	return eventstore.Query{From: f.from, Names: f.names, Tags: tags, Since: since, Until: until}, nil
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

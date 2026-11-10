package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func newEventCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "event <sequence>",
		Short: "Show one event in full, including its payload",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sequence, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sequence %q: want a number", args[0])
			}

			store, err := eventstore.Open(opts.store)
			if err != nil {
				return err
			}
			defer store.Close()

			e, err := store.Event(cmd.Context(), sequence)
			if err != nil {
				return err
			}

			if opts.jsonOutput() {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(eventjson.From(e))
			}
			return writeEventText(cmd.OutOrStdout(), e)
		},
	}
}

// writeEventText writes the event's fields, then its payload,
// indented if it is JSON.
func writeEventText(out io.Writer, e eventstore.Event) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Sequence\t%d\n", e.Sequence)
	fmt.Fprintf(w, "Name\t%s\n", e.Name)
	fmt.Fprintf(w, "Version\t%s\n", e.Version)
	fmt.Fprintf(w, "Time\t%s\n", e.Timestamp.Format(time.DateTime))
	fmt.Fprintf(w, "ID\t%s\n", e.ID)
	fmt.Fprintf(w, "Tags\t%s\n", formatTags(e.Tags))
	fmt.Fprintf(w, "Metadata\t%s\n", formatMetadata(e.Metadata))
	if err := w.Flush(); err != nil {
		return err
	}

	var payload bytes.Buffer
	if err := json.Indent(&payload, e.Payload, "", "  "); err != nil {
		payload.Reset()
		payload.Write(e.Payload) // not JSON: show it as it is
	}
	_, err := fmt.Fprintf(out, "\n%s\n", payload.Bytes())
	return err
}

// formatMetadata formats metadata as "key=value key=value", sorted by key.
// Go randomizes map order, so sorting keeps the output stable.
func formatMetadata(metadata map[string]string) string {
	parts := make([]string, 0, len(metadata))
	for _, key := range slices.Sorted(maps.Keys(metadata)) {
		parts = append(parts, key+"="+metadata[key])
	}
	return strings.Join(parts, " ")
}

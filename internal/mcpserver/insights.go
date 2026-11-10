package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/insight"
)

type trendInput struct {
	filterInput
	By       string `json:"by,omitempty" jsonschema:"period length: hour, day or week; default day"`
	TimeZone string `json:"timeZone,omitempty" jsonschema:"IANA time zone for day and week boundaries, like Europe/Amsterdam; default UTC"`
}

func eventStats(ctx context.Context, store *eventstore.Store, in filterInput, now time.Time) (eventjson.Stats, error) {
	query, err := in.query(now)
	if err != nil {
		return eventjson.Stats{}, err
	}
	query.SkipTags = true // counting needs no tags
	stats, err := insight.Summarize(store.Events(ctx, query))
	if err != nil {
		return eventjson.Stats{}, err
	}
	return eventjson.FromStats(stats), nil
}

func eventTrend(ctx context.Context, store *eventstore.Store, in trendInput, now time.Time) (eventjson.Trend, error) {
	query, periods, err := in.trend(now)
	if err != nil {
		return eventjson.Trend{}, err
	}
	buckets, err := insight.Trend(store.Events(ctx, query), periods)
	if err != nil {
		return eventjson.Trend{}, err
	}
	return eventjson.FromTrend(periods.Unit, periods.Location, buckets), nil
}

// trend turns the input into a query and the periods to split it into.
// Dates in 'since' and 'until' are read in the trend's time zone, so they
// line up with its days. Without 'until', the trend runs up to now, so
// recent quiet periods show.
func (in trendInput) trend(now time.Time) (eventstore.Query, insight.Periods, error) {
	by := in.By
	if by == "" {
		by = "day"
	}
	unit, err := insight.ParseUnit(by)
	if err != nil {
		return eventstore.Query{}, insight.Periods{}, fmt.Errorf("by: %w", err)
	}

	loc := time.UTC
	if in.TimeZone != "" {
		if loc, err = time.LoadLocation(in.TimeZone); err != nil {
			return eventstore.Query{}, insight.Periods{}, fmt.Errorf("timeZone: unknown time zone %q: want an IANA name like Europe/Amsterdam", in.TimeZone)
		}
	}

	query, err := in.query(now.In(loc))
	if err != nil {
		return eventstore.Query{}, insight.Periods{}, err
	}
	query.SkipTags = true // counting needs no tags

	periods := insight.Periods{Unit: unit, Location: loc, From: query.Since, To: query.Until}
	if periods.To.IsZero() {
		periods.To = now
	}
	return query, periods, nil
}

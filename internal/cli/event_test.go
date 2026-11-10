package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func TestWriteEventText(t *testing.T) {
	e := eventstore.Event{
		Sequence:  12,
		ID:        "abc",
		Name:      "com.example.LoggedOut",
		Version:   "1",
		Timestamp: time.Date(2026, 11, 5, 20, 9, 58, 0, time.Local),
		Tags:      []eventstore.Tag{{Key: "userId", Value: "42"}},
		Metadata:  map[string]string{"z": "last", "a": "first"},
		Payload:   []byte(`{"userId":"42"}`),
	}

	var out strings.Builder
	if err := writeEventText(&out, e); err != nil {
		t.Fatalf("writeEventText returned error: %v", err)
	}

	want := "" +
		"Sequence  12\n" +
		"Name      com.example.LoggedOut\n" +
		"Version   1\n" +
		"Time      2026-11-05 20:09:58\n" +
		"ID        abc\n" +
		"Tags      userId=42\n" +
		"Metadata  a=first z=last\n" +
		"\n" +
		"{\n" +
		"  \"userId\": \"42\"\n" +
		"}\n"
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestWriteEventTextShowsNonJSONPayloadAsIs(t *testing.T) {
	var out strings.Builder
	if err := writeEventText(&out, eventstore.Event{Payload: []byte("<xml/>")}); err != nil {
		t.Fatalf("writeEventText returned error: %v", err)
	}

	if !strings.HasSuffix(out.String(), "\n<xml/>\n") {
		t.Errorf("payload missing or changed:\n%s", out.String())
	}
}

// Package mcpserver exposes an event store to AI agents over the
// Model Context Protocol (MCP). Every tool it offers is read-only.
package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Run serves MCP over stdin and stdout until the client disconnects
// or ctx is cancelled. Nothing else may write to stdout while it runs,
// because stdout carries the protocol messages.
func Run(ctx context.Context, store *eventstore.Store) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "axonscope", Version: "dev"}, nil)
	addTools(server, store)
	return server.Run(ctx, &mcp.StdioTransport{})
}

// readOnly marks a tool as safe to call: it never changes the event store.
var readOnly = &mcp.ToolAnnotations{ReadOnlyHint: true}

func addTools(server *mcp.Server, store *eventstore.Store) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "head",
		Description: "Get the head of the event store: the sequence number the next event will get. " +
			"Sequence numbers start at 0, so this is also the number of events stored.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, headOutput, error) {
		head, err := store.Head(ctx)
		return nil, headOutput{Head: head}, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_events",
		Description: "List events oldest first, starting at a sequence number. " +
			"Each event has a type name, timestamp, tags (domain identifiers such as userId), metadata and a payload. " +
			"If more events exist, the result contains 'next': pass it as 'from' to read the following page.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listEventsInput) (*mcp.CallToolResult, listEventsOutput, error) {
		out, err := listEvents(ctx, store, in)
		return nil, out, err
	})
}

type headOutput struct {
	Head int64 `json:"head" jsonschema:"the sequence number the next event will get"`
}

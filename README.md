# axonscope

A read-only CLI and MCP server for inspecting [Axon Server](https://www.axoniq.io/products/axon-server) event stores.

Browse, filter and count events from your terminal, or connect an AI assistant through the
[Model Context Protocol](https://modelcontextprotocol.io) and ask it what happened.
axonscope never changes your event store.

It supports event stores that use Dynamic Consistency Boundaries (DCB), the event store
model of Axon Framework 5.

## Install

With [Go](https://go.dev/dl/) 1.27.3 or newer:

```sh
go install github.com/ayberkyonar/axonscope/cmd/axonscope@latest
```

## Quick start

With Axon Server running on `localhost:8124`:

```sh
axonscope head                                # number of events stored
axonscope events                              # first 20 events
axonscope events --name OrderPlaced --since 7d
axonscope events --tag customerId=42
axonscope event 12                            # one event, with its payload
```

Overviews:

```sh
axonscope stats --since 24h                   # events per type in the last day
axonscope stats --since 7d --by-tag customerId          # most active customers
axonscope trend --since 7d                    # events per day, including quiet days
axonscope trend --since 24h --by hour --name OrderPlaced
axonscope journeys --by-tag orderId --since 7d --containing OrderCancelled
                                              # what happens before orders get cancelled
```

Counting by tag and journeys need each event's tags, which costs one call to Axon Server per
event. They stop after 10,000 events (`--max-events`) and say so when the answer is incomplete.

Event names can be full (`com.example.OrderPlaced`) or short (`OrderPlaced`). Times can be
a date (`2026-11-05`), a date and time (`2026-11-05T21:00`), an RFC 3339 timestamp, or a
duration ago (`90m`, `24h`, `7d`, `2w`).

Add `-o json` to any command for JSON output. `events` prints one JSON object per line:

```sh
axonscope events -o json | jq .payload
```

Run `axonscope --help` or `axonscope <command> --help` for all flags.

## Connecting to Axon Server

| Flag / variable | Default | Purpose |
|---|---|---|
| `--server` | `localhost:8124` | Axon Server gRPC address |
| `--context` | `default` | Axon Server context |
| `AXONSCOPE_TOKEN` | | Access token, when access control is enabled |
| `--tls` | off | Encrypt the connection |
| `--ca-cert` | | CA certificate (PEM) for servers with a company-internal CA; implies `--tls` |

When Axon Server has access control enabled, give axonscope its own token with only the
`READ_EVENTS` role. With the CLI that comes with Axon Server:

```sh
java -jar axonserver-cli.jar register-application -a axonscope -r READ_EVENTS@default -t <admin-token>
```

It prints the new token; set it as `AXONSCOPE_TOKEN`.

## Use with AI assistants

`axonscope mcp` runs an MCP server over stdin/stdout. Your AI client starts it; you don't
run it yourself.

**Claude Code:**

```sh
claude mcp add axonscope -e AXONSCOPE_TOKEN=your-token -- axonscope mcp --server localhost:8124
```

**Claude Desktop** and other clients that use a JSON config:

```json
{
  "mcpServers": {
    "axonscope": {
      "command": "axonscope",
      "args": ["mcp", "--server", "localhost:8124"],
      "env": { "AXONSCOPE_TOKEN": "your-token" }
    }
  }
}
```

If the client can't find `axonscope`, use the full path to the executable as `command`.

The server offers these tools, all read-only:

| Tool | Purpose |
|---|---|
| `head` | Number of events stored |
| `event_stats` | Events per type in a period, or per tag value: a quick overview |
| `event_trend` | Events per hour, day or week, in the user's time zone |
| `event_journeys` | Common paths that orders, users or other entities take |
| `list_events` | Events filtered by name, tags and time, with paging |
| `get_event` | One event with its payload |

Event payloads can contain personal or sensitive data. Only connect axonscope to event
stores whose data you are allowed to share with your AI assistant.

## License

axonscope is licensed under the [MIT License](LICENSE).

It includes code generated from the [Axon Server API](https://github.com/AxonIQ/axon-server-api)
(release 2026.1.0), which is licensed under the Apache License 2.0; see
[`internal/eventstore/internal/axonpb/LICENSE.txt`](internal/eventstore/internal/axonpb/LICENSE.txt).
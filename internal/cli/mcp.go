package cli

import (
	"github.com/spf13/cobra"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/mcpserver"
)

func newMCPCmd(cfg *eventstore.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server over stdin/stdout, for AI agents",
		Long: "Run a read-only Model Context Protocol server over stdin/stdout.\n" +
			"An MCP client such as Claude Code starts this command itself; you don't run it by hand.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := eventstore.Open(*cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			return mcpserver.Run(cmd.Context(), store)
		},
	}
}

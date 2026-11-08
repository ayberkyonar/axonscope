// Command axonscope inspects Axon Server event stores.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/ayberkyonar/axonscope/internal/cli"
)

func main() {
	// Cancel running requests when the user presses Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := cli.Execute(ctx)
	stop()
	if err != nil {
		os.Exit(1) // Cobra has already printed the error.
	}
}

// Command axonscope inspects Axon Server event stores.
package main

import (
	"context"
	"os"
	"os/signal"
	// Embeds the time zone database, so names like Europe/Amsterdam work on
	// systems without one, such as Windows and the minimal Docker image.
	_ "time/tzdata"

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

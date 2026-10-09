package projectcli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
)

func runDeliver(opts options, out io.Writer) error {
	if !opts.write {
		return fmt.Errorf("project deliver requires --write authorization")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	report, deliverErr := projectrun.Deliver(ctx, projectRunHost(), projectRunInvoker(), opts.repo, projectrun.DeliverRequest{
		ExplorationID: opts.explorationID, ScopeID: opts.scope, RunID: opts.run, ExecuteAuthorized: opts.write,
	})
	if err := writeJSON(out, report); err != nil {
		return err
	}
	return deliverErr
}

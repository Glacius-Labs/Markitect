package bootstrap

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
)

// cockpitCommand serves the cockpit over the selected state directories. A run
// configuration contributes its state directory; nothing else of it is used.
func cockpitCommand(ctx context.Context, request cli.CockpitRequest, ready func(url string)) error {
	states := append([]string(nil), request.States...)
	if request.RunConfig != "" {
		cfg, err := loadRunConfig(ctx, request.RunConfig)
		if err != nil {
			return err
		}
		states = append(states, cfg.StateDirectory)
	}
	server, err := cockpit.New(cockpit.Config{States: states})
	if err != nil {
		return err
	}
	return server.Serve(ctx, request.Listen, ready)
}

// Package checks prepares pinned container-check metadata and observes bounded
// exact-candidate checks. Structural preparation invokes no project command.
package checks

import (
	"context"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

type MetadataPreparer struct{ factory ImageReaderFactory }

func (p MetadataPreparer) PrepareCheckProfile(ctx context.Context, input runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
	if input.Section() != "container" {
		return runtimeconfig.ProfileObservation{InputBinding: input.Binding(), Status: model.Failed}, fmt.Errorf("only pinned container check profiles are supported")
	}
	return p.prepareContainer(ctx, input)
}

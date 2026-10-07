package canonical

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/core"
)

// ValidateProjectionRequest reuses BindProjection to check a detached request
// against normalized canonical meaning and its own declared runtime binding.
// It does not authenticate that runtime or prove it is installed. Host must
// independently resolve the exact activated Module before executing it.
func ValidateProjectionRequest(model core.Model, request ProjectionRequest) error {
	activation := Activation{
		Modules: []Pin{request.ModulePin}, Schemas: model.Schemas,
		Projectors:  []RegisteredProjector{{Module: request.ModulePin, Registration: request.Projector}},
		ModuleTypes: map[Pin]string{request.ModulePin: ModuleTypeProjection},
	}
	fresh, err := BindProjection(model, activation, []ProjectionBinding{request.Binding}, request.Projection.Identity(), request.TargetFiles)
	if err != nil {
		return err
	}
	left, err := json.Marshal(request)
	if err != nil {
		return err
	}
	right, err := json.Marshal(fresh)
	if err != nil {
		return err
	}
	if !bytes.Equal(left, right) {
		return fmt.Errorf("detached Projection request differs from its canonical meaning or digest-bound inputs")
	}
	return nil
}

package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// VolumeObservation records engine existence and labels, never inferred from a
// failed inspect. Missing volumes are established by a successful exact census.
type VolumeObservation struct {
	Name   string
	Exists bool
	Labels map[string]string
}

// InspectVolume observes one named volume. Engine/census failures are errors,
// rather than evidence that the volume is absent.
func (e *Engine) InspectVolume(ctx context.Context, name string) (VolumeObservation, error) {
	v := VolumeObservation{Name: name}
	if !namePattern.MatchString(name) {
		return v, fmt.Errorf("invalid volume name %q", name)
	}
	out, err := e.output(ctx, "volume", "ls", "--format", "{{.Name}}")
	if err != nil {
		return v, err
	}
	found := false
	for _, n := range strings.Fields(string(out)) {
		if n == name {
			found = true
		}
	}
	if !found {
		return v, nil
	}
	out, err = e.output(ctx, "volume", "inspect", name)
	if err != nil {
		return v, err
	}
	var rows []struct {
		Name   string
		Labels map[string]string
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return v, fmt.Errorf("decode volume observation: %w", err)
	}
	if len(rows) != 1 || rows[0].Name != name {
		return v, fmt.Errorf("volume observation identity mismatch")
	}
	v.Exists, v.Labels = true, rows[0].Labels
	return v, nil
}

const creationLabel = "io.markitect.volume-creation"

// CreateVolumeExclusive refuses preexisting volumes and verifies a fresh random
// ownership label after Docker's idempotent create. Docker has no atomic exclusive
// create; a competing creation or ambiguous readback is refused before any writes.
// On error the volume may exist and is deliberately not removed.
func (e *Engine) CreateVolumeExclusive(ctx context.Context, name string, labels map[string]string) (VolumeObservation, error) {
	v, err := e.InspectVolume(ctx, name)
	if err != nil {
		return v, err
	}
	if v.Exists {
		return v, fmt.Errorf("volume %s already exists", name)
	}
	if len(labels) == 0 {
		return v, fmt.Errorf("exclusive volume needs ownership labels")
	}
	if _, ok := labels[creationLabel]; ok {
		return v, fmt.Errorf("reserved volume creation label")
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return v, err
	}
	want := make(map[string]string, len(labels)+1)
	for k, value := range labels {
		want[k] = value
	}
	want[creationLabel] = hex.EncodeToString(token)
	if err := e.CreateVolume(ctx, name, want); err != nil {
		return v, err
	}
	v, err = e.InspectVolume(ctx, name)
	if err != nil {
		return v, err
	}
	if !v.Exists {
		return v, fmt.Errorf("created volume %s is absent", name)
	}
	for k, value := range want {
		if v.Labels[k] != value {
			return v, fmt.Errorf("volume %s creation ownership is uncertain", name)
		}
	}
	return v, nil
}

// VolumeMounts returns every container ID mounting the exact named volume,
// including stopped containers and read-only mounts. Any incomplete inspect
// fails the census; it cannot establish that the volume is unused.
func (e *Engine) VolumeMounts(ctx context.Context, name string) ([]string, error) {
	if !namePattern.MatchString(name) {
		return nil, fmt.Errorf("invalid volume name %q", name)
	}
	out, err := e.output(ctx, "ps", "--all", "--quiet", "--no-trunc")
	if err != nil {
		return nil, err
	}
	var mounted []string
	for _, id := range strings.Fields(string(out)) {
		if !containerIDPattern.MatchString(id) {
			return nil, fmt.Errorf("invalid container census ID %q", id)
		}
		out, err := e.output(ctx, "inspect", "--type", "container", id)
		if err != nil {
			return nil, err
		}
		var rows []struct {
			ID     string
			Mounts *[]struct{ Type, Name string }
		}
		if err := json.Unmarshal(out, &rows); err != nil {
			return nil, fmt.Errorf("decode mount census: %w", err)
		}
		if len(rows) != 1 || rows[0].ID != id || rows[0].Mounts == nil {
			return nil, fmt.Errorf("container mount census identity mismatch")
		}
		for _, m := range *rows[0].Mounts {
			if m.Type == "" || (m.Type == "volume" && !namePattern.MatchString(m.Name)) {
				return nil, fmt.Errorf("container mount census has an incomplete mount entry")
			}
			if m.Type == "volume" && m.Name == name {
				mounted = append(mounted, id)
				break
			}
		}
	}
	return mounted, nil
}

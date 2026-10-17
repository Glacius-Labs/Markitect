// Package codexappserver prepares explicitly selected offline file metadata.
// It starts no process and establishes no runtime or provider capability.
package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const metadataPinName = "codex-cli-0.162.0-alpha.17.2-offline-v1"

// MetadataPreparer implements Apply's process-free AgentProfilePreparer.
// Its only admitted pin is historical offline evidence, not runtime support.
// The zero value is usable and has the same fixed pin as the constructor.
type MetadataPreparer struct{}

func NewMetadataPreparer() *MetadataPreparer { return &MetadataPreparer{} }

var _ apply.AgentProfilePreparer = (*MetadataPreparer)(nil)

func (*MetadataPreparer) PrepareAgentProfile(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	return prepareMetadata(ctx, input, historicalPin())
}

type preparation struct {
	input apply.ProfileInput
	out   apply.ProfileObservation
	err   error
}

func (p *preparation) issue(status model.Status, code, field, message string) {
	if status == model.Failed || p.out.Status != model.Failed {
		p.out.Status = status
	}
	doc := p.input.Document()
	locator, _ := json.Marshal([2]string{doc.Location.Boundary, doc.Location.Path.String()})
	p.out.Diagnostics = append(p.out.Diagnostics, apply.HostProfileDiagnostic{
		Code:    "codex_metadata." + code,
		Source:  organization.SettingSource{File: "source:" + string(locator), ContentID: doc.File.ContentID(), Field: p.input.Field() + field},
		Message: message,
	})
}

func (p *preparation) acquisition(err error, field string) {
	if err == nil {
		return
	}
	p.err = errors.Join(p.err, err)
	var invalid *invalidMetadata
	if errors.As(err, &invalid) {
		p.issue(model.Failed, "invalid_file", field, err.Error())
	} else {
		p.issue(model.NotCompleted, "acquisition", field, err.Error())
	}
}

func prepareMetadata(ctx context.Context, input apply.ProfileInput, pin offlinePin) (apply.ProfileObservation, error) {
	p := preparation{input: input, out: apply.ProfileObservation{
		InputBinding: input.Binding(), Status: model.Passed,
		Contract: apply.ProfileContract{Name: "markitect.codex-app-server.offline-metadata", Version: 1},
	}}
	if ctx == nil {
		p.acquisition(fmt.Errorf("nil preparation context"), "")
		return p.out, p.err
	}
	if err := ctx.Err(); err != nil {
		p.acquisition(err, "")
		return p.out, p.err
	}
	config, ok := p.decode()
	if !ok {
		return p.out, p.err
	}
	for _, location := range config.instructions {
		found := false
		for _, file := range input.Instructions() {
			if file.Location == location {
				found = true
				break
			}
		}
		if found {
			p.out.Instructions = append(p.out.Instructions, location)
		} else {
			p.out.RequiredInstructions = append(p.out.RequiredInstructions, location)
			p.issue(model.NotCompleted, "instruction_uncaptured", "/instructions", "Selected instruction needs a new complete acquisition; no instruction file was read.")
		}
	}
	disk := newMetadataDisk(ctx)
	defer disk.close()
	executable, _, err := disk.readAbsolute(config.executable, maxExecutableBytes, false)
	if err == nil {
		p.out.Files = append(p.out.Files, executable)
		if executable.ContentID != pin.executable {
			p.issue(model.Failed, "executable_pin", "/executable", "Actual executable bytes do not match the fixed historical offline pin.")
		}
	}
	p.acquisition(err, "/executable")
	for i, boundary := range []string{config.stable, config.experimental} {
		field := []string{"/schemas/stable", "/schemas/experimental"}[i]
		inventory, complete, inventoryErr := disk.inventory(boundary, pin.selected)
		p.out.Inventories = append(p.out.Inventories, inventory)
		p.acquisition(inventoryErr, field)
		if complete && inventoryDigest(inventory.Files) != pin.inventories[i] {
			p.issue(model.Failed, "inventory_pin", field, "Complete observed schema inventory does not match the fixed historical offline pin.")
		}
	}
	p.acquisition(disk.verify(), "")
	return p.out, p.err
}

type offlinePin struct {
	executable  snapshot.ContentID
	inventories [2]snapshot.ContentID
	selected    []string
}

// These identities were reviewed from the 2026-10-17 offline inventory. Version
// is bound ONLY by the known executable hash. No fresh export/version is claimed.
func historicalPin() offlinePin {
	return offlinePin{
		executable: digestLiteral("e98c516f1c8935789aabe9122ee7e05ec4c3efd1ba6acf179242e031c495b99e"),
		inventories: [2]snapshot.ContentID{
			digestLiteral("77e83bf79a8aa71a8747e3d3744f38ab4d82fb33ba619cc04b05716a85c31fca"),
			digestLiteral("6e8dd84c70d64c848a8e11fd7a61e2ed9dbdbda7a45ef65a1f1a295007288a27"),
		},
		selected: []string{"v1/InitializeParams.json", "v1/InitializeResponse.json", "v2/ThreadStartParams.json", "v2/ThreadStartResponse.json", "v2/TurnStartParams.json", "ServerRequest.json", "DynamicToolCallParams.json", "DynamicToolCallResponse.json"},
	}
}

func overlaps(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	return err == nil && (rel == "." || rel != ".." && !filepath.IsAbs(rel) && len(rel) > 0 && rel[:min(len(rel), 3)] != ".."+string(filepath.Separator))
}

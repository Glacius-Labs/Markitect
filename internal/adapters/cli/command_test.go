package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func TestRejectionLimitUsesCurrentFieldNameInReport(t *testing.T) {
	projected, ok := reportValue(organization.ManagerSettings{
		RejectionLimit: organization.Setting[int]{Value: 3},
	}).(map[string]any)
	if !ok {
		t.Fatalf("reportValue returned %T", reportValue(organization.ManagerSettings{}))
	}
	if _, exists := projected["ReviewRounds"]; exists {
		t.Fatalf("report retained historical field name: %#v", projected)
	}
	selected, ok := projected["RejectionLimit"].(map[string]any)
	if !ok || selected["Value"] != 3 {
		t.Fatalf("report did not expose the renamed selection: %#v", projected)
	}
}

type resolver struct{ calls int }

func (r *resolver) ResolveRevision(context.Context, string, string) (string, error) {
	r.calls++
	return strings.Repeat("a", 40), nil
}

type unavailableSource struct {
	calls   int
	request apply.PreviewRequest
}

func (s *unavailableSource) CapturePreview(_ context.Context, r apply.PreviewRequest) (apply.PreviewedInputs, error) {
	s.calls++
	s.request = r
	return apply.PreviewedInputs{Revision: r.Revision}, errors.New("controlled acquisition unavailable")
}

func TestExplicitRevisionResolvedOnceWithoutExecution(t *testing.T) {
	source := &unavailableSource{}
	revision := &resolver{}
	c := New(apply.NewPreparationService(apply.NewModelService(nil, source)), nil, nil, revision, nil, nil, nil)
	var out, errout bytes.Buffer
	if code := c.Run(context.Background(), []string{"check", "--repo", t.TempDir(), "--revision", "HEAD"}, &out, &errout); code != 3 {
		t.Fatalf("%d: %s %s", code, out.String(), errout.String())
	}
	if revision.calls != 1 || source.calls != 1 || source.request.Revision != strings.Repeat("a", 40) {
		t.Fatalf("resolver=%d source=%d request=%+v", revision.calls, source.calls, source.request)
	}
	if !strings.Contains(out.String(), "controlled acquisition unavailable") || !strings.Contains(out.String(), `"requested": false`) {
		t.Fatal(out.String())
	}
}

func TestUnavailableApplyDoesNotAcquireOrResolveEvenWithInvalidArguments(t *testing.T) {
	source := &unavailableSource{}
	revision := &resolver{}
	c := New(apply.NewPreparationService(apply.NewModelService(nil, source)), nil, nil, revision, nil, nil, nil)
	var out, errout bytes.Buffer
	if code := c.Run(context.Background(), []string{"apply", "--repo", "absent", "--revision", "HEAD", "--unknown"}, &out, &errout); code != 3 || !strings.Contains(out.String(), "unavailable") {
		t.Fatalf("%d %s", code, out.String())
	}
	if revision.calls != 0 || source.calls != 0 {
		t.Fatal("unavailable Apply acquired a source")
	}
}

func TestRelativeManifestKeepsExplicitRepositoryBoundary(t *testing.T) {
	selection, err := selectManifest(filepath.Join("alias", "repository"), ".markitect/model.yaml")
	if err != nil || selection.Boundary != "." || selection.ManifestPath.String() != ".markitect/model.yaml" {
		t.Fatalf("%+v %v", selection, err)
	}
	for _, bad := range []string{"../model.yaml", "./model.yaml", "a//model.yaml"} {
		if _, err := selectManifest(".", bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

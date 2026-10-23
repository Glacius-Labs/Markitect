package apply_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

func mutateRaw(raw apply.RawInputs) {
	for i := range raw.Local.Source.Entries {
		if len(raw.Local.Source.Entries[i].Bytes) > 0 {
			raw.Local.Source.Entries[i].Bytes[0] = '!'
			break
		}
	}
	raw.Dependencies[0].Source.Entries[0].Path = "mutation"
	for i := range raw.Dependencies[0].Source.Entries {
		if len(raw.Dependencies[0].Source.Entries[i].Bytes) > 0 {
			raw.Dependencies[0].Source.Entries[i].Bytes[0] = '!'
			break
		}
	}
	raw.Dependencies[0].Selection.Boundary = "mutation"
	raw.Execution[0].Location.Boundary = "mutation"
	raw.Registry[0].Location.Boundary = "mutation"
	raw.Ignore[0].Location.Boundary = "mutation"
	raw.RuntimeProfiles[0].Location.Boundary = "mutation"
	raw.Instructions[0].Location.Boundary = "mutation"
}

func mutateSelection(selection apply.InputSelection) {
	selection.Dependencies[0].Boundary = "mutation"
	selection.Execution[0].Boundary = "mutation"
	selection.Registry[0].Boundary = "mutation"
	selection.Ignore[0].Boundary = "mutation"
	selection.RuntimeProfiles[0].Boundary = "mutation"
	selection.Instructions[0].Boundary = "mutation"
}

func TestAcquisitionAndRequestSlicesAreIsolatedAcrossBothPorts(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(map[bool]string{false: "capture", true: "preview"}[preview], func(t *testing.T) {
			selection, raw := libraryInputs(t)
			withSelectedSources(t, &selection, &raw)
			source := controlledSource{
				capture: func(_ context.Context, request apply.CaptureRequest) (apply.CapturedInputs, error) {
					mutateSelection(request.Selection)
					return captured(raw), nil
				},
				preview: func(_ context.Context, request apply.PreviewRequest) (apply.PreviewedInputs, error) {
					mutateSelection(request.Selection)
					return apply.PreviewedInputs{Raw: raw, Head: fixtureHead}, nil
				},
			}
			service := apply.NewModelService(source, source)
			var inputs func() apply.RawInputs
			var request func() apply.InputSelection
			var check func() model.Result
			if preview {
				result, err := service.PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection}, model.Limits{})
				if err != nil {
					t.Fatal(err)
				}
				inputs, request, check = func() apply.RawInputs { return result.Inputs().Raw }, func() apply.InputSelection { return result.Request().Selection }, result.Checked
			} else {
				result, err := service.CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
				if err != nil {
					t.Fatal(err)
				}
				inputs, request, check = func() apply.RawInputs { return result.Inputs().Raw }, func() apply.InputSelection { return result.Request().Selection }, result.Checked
			}
			beforeInputs, beforeRequest, beforeCheck := inputs(), request(), check()
			if !reflect.DeepEqual(beforeRequest, selection) {
				t.Fatal("port mutated retained request or caller")
			}
			mutateRaw(raw)
			mutateSelection(selection)
			mutateRaw(inputs())
			mutateSelection(request())
			if !reflect.DeepEqual(inputs(), beforeInputs) || !reflect.DeepEqual(request(), beforeRequest) || !reflect.DeepEqual(check(), beforeCheck) {
				t.Fatal("input/request alias crossed immutable result boundary")
			}
		})
	}
}

func TestModelCheckResultsCopyNestedContainersAndOptionalSpans(t *testing.T) {
	selection, raw := libraryInputs(t)
	for i := range raw.Local.Source.Entries {
		entry := &raw.Local.Source.Entries[i]
		if strings.HasSuffix(entry.Path, "/borrow.yaml") {
			entry.Bytes = append(entry.Bytes, []byte("\ntitle: Borrow fixture\n")...)
			// Add a display-bearing link without changing any dependency bytes.
			entry.Bytes = []byte(strings.Replace(string(entry.Bytes), "]]", "|fixture display]]", 1))
		}
	}
	refreshLocalFiles(t, &raw)
	result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || result.Checked().Status != model.Passed {
		t.Fatalf("%v %+v", err, result.Checked())
	}
	before, changed := result.Checked(), result.Checked()
	changed.Models[0].Canonical[0] = '!'
	changed.Models[0].Phases[0].Name = "mutation"
	changed.Models[0].Packages[0].Name = "mutation"
	titles, displays, bodies, sources, specs := 0, 0, 0, 0, 0
	for i := range changed.Models[0].Definitions {
		d := &changed.Models[0].Definitions[i]
		if d.Title != nil {
			*d.Title = "mutation"
			titles++
		}
		if len(d.Spec) > 0 {
			d.Spec[0].Name = "mutation"
			specs++
		}
		for j := range d.BodyLinks {
			link := &d.BodyLinks[j]
			if link.Display != nil {
				*link.Display = "mutation"
				displays++
			}
			if link.Coordinates != nil {
				if link.Coordinates.Body != nil {
					link.Coordinates.Body.StartByte = -1
					bodies++
				}
				if link.Coordinates.Source != nil {
					link.Coordinates.Source.Start.Line = -1
					sources++
				}
			}
		}
	}
	if titles == 0 || displays == 0 || bodies == 0 || sources == 0 || specs == 0 {
		t.Fatalf("copy fixture did not exercise nested fields: %d/%d/%d/%d/%d", titles, displays, bodies, sources, specs)
	}
	if !reflect.DeepEqual(before, result.Checked()) {
		t.Fatal("Model check report alias escaped")
	}
}

func TestModelCheckDiagnosticPositionsAreCopied(t *testing.T) {
	selection, raw := libraryInputs(t)
	for i := range raw.Local.Source.Entries {
		if strings.HasSuffix(raw.Local.Source.Entries[i].Path, "/borrow.yaml") {
			raw.Local.Source.Entries[i].Bytes = append(raw.Local.Source.Entries[i].Bytes, []byte("\nID: collision\n")...)
		}
	}
	refreshLocalFiles(t, &raw)
	result, err := captureService(captured(raw), nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || result.Checked().Status != model.Failed {
		t.Fatal("failure fixture did not pass the Model check")
	}
	before, changed := result.Checked(), result.Checked()
	positions := 0
	for i := range changed.Diagnostics {
		changed.Diagnostics[i].Message = "mutation"
		if changed.Diagnostics[i].Position != nil {
			changed.Diagnostics[i].Position.Line = -1
			positions++
		}
	}
	if positions == 0 || !reflect.DeepEqual(before, result.Checked()) {
		t.Fatal("diagnostic position alias escaped or was not tested")
	}
}

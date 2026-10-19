package checks

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"go.yaml.in/yaml/v3"
)

func setupContainerProfile(engine string) ContainerHostProfile {
	return ContainerHostProfile{EngineExecutable: engine, Profile: ContainerProfile{
		ImageID: checkImageID, Executable: "/usr/local/go/bin/go", ArgumentPrefix: nil, Env: nil,
		Limits: run.CheckLimits{MilliCPUs: 2000, MemoryBytes: 4 << 30, Processes: 1024}, MaxOutputBytes: 1 << 20,
	}}
}

func TestContainerHostSectionRoundTripsThroughTheMetadataDecoder(t *testing.T) {
	engine := filepath.Join(testkit.TempDir(t), "engine")
	want := setupContainerProfile(engine)
	section := want.Section()
	text, err := yaml.Marshal(section)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := yaml.Marshal(section)
	if string(text) != string(again) || !strings.Contains(string(text), "env: []\n") || !strings.Contains(string(text), "argumentPrefix: []\n") {
		t.Fatalf("encoding is not deterministic or encodes null lists:\n%s", text)
	}
	prepared, err := preparedContainer(t, string(text), NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { return imageBytesReader([]byte("executable")), nil }))
	if err != nil || prepared.Observation().Status != model.Passed {
		t.Fatalf("the decoder refused the encoding: %v %+v\n%s", err, prepared.Observation().Diagnostics, text)
	}
	got, err := ContainerProfileFromPrepared(prepared)
	if err != nil {
		t.Fatal(err)
	}
	wantProfile := want.Profile
	wantProfile.ArgumentPrefix, wantProfile.Env = []string{}, []string{}
	if !reflect.DeepEqual(got, wantProfile) {
		t.Fatalf("decoded %+v\nwant %+v", got, wantProfile)
	}
}

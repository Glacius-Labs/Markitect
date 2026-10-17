package checks

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const checkImageID = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func containerFields(engine string) string {
	return fmt.Sprintf("profileVersion: 1\nengineExecutable: %s\nimageID: %s\nexecutable: /opt/check\nargumentPrefix: ['literal prefix', '']\nenv: ['A=literal value', 'EMPTY=']\nlimits: {milliCPUs: 1250, memoryBytes: 16777216, processes: 8}\nmaxOutputBytes: 1024\n", strconv.Quote(engine), checkImageID)
}

type imageReaderPeer struct {
	image func(context.Context, string) (string, error)
	file  func(context.Context, string, string, int64) (container.ImageFileObservation, error)
}

func (p imageReaderPeer) ImageID(ctx context.Context, image string) (string, error) {
	return p.image(ctx, image)
}
func (p imageReaderPeer) ObserveImageFile(ctx context.Context, image, file string, max int64) (container.ImageFileObservation, error) {
	return p.file(ctx, image, file, max)
}

func imageBytesReader(data []byte) imageReaderPeer {
	return imageReaderPeer{
		image: func(_ context.Context, image string) (string, error) { return image, nil },
		file: func(context.Context, string, string, int64) (container.ImageFileObservation, error) {
			return container.ImageFileObservation{SHA256: sha256.Sum256(data), Size: int64(len(data))}, nil
		},
	}
}

type checkPreparerFunc func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error)

func (f checkPreparerFunc) PrepareCheckProfile(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	return f(ctx, input)
}

func preparedContainer(t *testing.T, fields string, preparer apply.CheckProfilePreparer) (apply.PreparedProfile, error) {
	t.Helper()
	files, defaults := m1SectionParents(t, "container", fields)
	result, err := files.PrepareSettings(context.Background(), defaults, nil, preparer)
	for _, profile := range result.ProfileObservations() {
		if profile.Input().Namespace() == "checkProfiles" {
			return profile, err
		}
	}
	t.Fatalf("no actual selected Check profile: %v %+v", err, result.Diagnostics())
	return apply.PreparedProfile{}, err
}

func TestContainerMetadataBindsWholeImageExecutableAndProjectsExistingRequest(t *testing.T) {
	engine := filepath.Join(testkit.TempDir(t), "explicit-engine-not-executed")
	// The executable is larger than the separate 1 MiB agent metadata reader.
	data := []byte(strings.Repeat("whole executable bytes\x00", 60000))
	peer := imageBytesReader(data)
	imageCalls, fileCalls, factoryCalls := 0, 0, 0
	peer.image = func(_ context.Context, image string) (string, error) {
		imageCalls++
		if image != checkImageID {
			t.Fatal("image reference changed")
		}
		return image, nil
	}
	peer.file = func(_ context.Context, image, file string, max int64) (container.ImageFileObservation, error) {
		fileCalls++
		if image != checkImageID || file != "/opt/check" || max != container.MaxImageFileObservationBytes {
			t.Fatal("image observation changed exact configuration or executable bound")
		}
		return container.ImageFileObservation{SHA256: sha256.Sum256(data), Size: int64(len(data))}, nil
	}
	preparer := NewMetadataPreparer(func(config EngineConfig) (ImageReader, error) {
		factoryCalls++
		if config.Executable != engine {
			t.Fatal("engine path was not passed literally")
		}
		return peer, nil
	})
	prepared, err := preparedContainer(t, containerFields(engine), preparer)
	o := prepared.Observation()
	wantImage := apply.CheckImageObservation{ImageID: checkImageID, Executable: apply.AgentImageMetadataFile{Path: "/opt/check", ContentID: snapshot.ContentID(sha256.Sum256(data)), Size: int64(len(data))}}
	if err != nil || o.Status != model.Passed || o.InputBinding != prepared.Input().Binding() || o.Contract != containerContract || o.CheckImage == nil || *o.CheckImage != wantImage || o.AgentImage != nil || len(o.Files)+len(o.Inventories)+len(o.Instructions)+len(o.RequiredInstructions) != 0 || factoryCalls != 1 || imageCalls != 1 || fileCalls != 1 {
		t.Fatalf("whole image evidence=%+v err=%v calls=%d/%d/%d", o, err, factoryCalls, imageCalls, fileCalls)
	}
	profile, err := ContainerProfileFromPrepared(prepared)
	if err != nil {
		t.Fatal(err)
	}
	wantTree := checkRequest(t).Tree
	_, declaration, _ := containerProfileInputs(t)
	request, err := profile.Request(declaration, wantTree)
	if err != nil || request.ImageID != checkImageID || request.Executable != "/opt/check" || !request.Tree.Equal(wantTree) || !reflect.DeepEqual(request.Args, append([]string{"literal prefix", ""}, declaration.Args.Value...)) || !reflect.DeepEqual(request.Env, []string{"A=literal value", "EMPTY="}) || request.Limits.MilliCPUs != 1250 || request.Limits.MemoryBytes != 16777216 || request.Limits.Processes != 8 || request.MaxOutputBytes != 1024 {
		t.Fatalf("existing request projection=%+v err=%v", request, err)
	}
	profile.ArgumentPrefix[0], profile.Env[0] = "changed", "CHANGED=caller"
	second, err := ContainerProfileFromPrepared(prepared)
	if err != nil || second.ArgumentPrefix[0] != "literal prefix" || second.Env[0] != "A=literal value" || request.Args[0] != "literal prefix" || request.Env[0] != "A=literal value" || factoryCalls != 1 || imageCalls != 1 || fileCalls != 1 {
		t.Fatal("pure projection shared slices or observed runtime again")
	}
}

func TestContainerMetadataRejectsClosedConfigurationBeforeAcquisition(t *testing.T) {
	engine := filepath.Join(testkit.TempDir(t), "engine")
	valid := containerFields(engine)
	for _, tc := range []struct{ name, from, to string }{
		{"version", "profileVersion: 1", "profileVersion: 2"},
		{"version type", "profileVersion: 1", "profileVersion: '1'"},
		{"unknown field", "profileVersion: 1", "unknown: true\nprofileVersion: 1"},
		{"missing prefix", "argumentPrefix: ['literal prefix', '']\n", ""},
		{"prefix type", "argumentPrefix: ['literal prefix', '']", "argumentPrefix: false"},
		{"prefix item", "argumentPrefix: ['literal prefix', '']", "argumentPrefix: [7]"},
		{"missing env", "env: ['A=literal value', 'EMPTY=']\n", ""},
		{"duplicate env", "env: ['A=literal value', 'EMPTY=']", "env: ['A=1', 'A=2']"},
		{"invalid env key", "env: ['A=literal value', 'EMPTY=']", "env: ['1A=x']"},
		{"env without equals", "env: ['A=literal value', 'EMPTY=']", "env: ['A']"},
		{"image tag", checkImageID, "check:latest"},
		{"image uppercase", checkImageID, "sha256:" + strings.Repeat("A", 64)},
		{"image path", "executable: /opt/check", "executable: check"},
		{"unclean path", "executable: /opt/check", "executable: /opt/../check"},
		{"root path", "executable: /opt/check", "executable: /"},
		{"backslash path", "executable: /opt/check", "executable: '/opt\\check'"},
		{"engine path search", "engineExecutable: " + strconv.Quote(engine), "engineExecutable: docker"},
		{"zero cpu", "milliCPUs: 1250", "milliCPUs: 0"},
		{"fractional cpu", "milliCPUs: 1250", "milliCPUs: 1.5"},
		{"negative memory", "memoryBytes: 16777216", "memoryBytes: -1"},
		{"missing processes", ", processes: 8", ""},
		{"resource overflow", "processes: 8", "processes: 9223372036854775808"},
		{"unknown limit", "processes: 8", "processes: 8, extra: 1"},
		{"zero output", "maxOutputBytes: 1024", "maxOutputBytes: 0"},
		{"output overflow", "maxOutputBytes: 1024", "maxOutputBytes: 67108865"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := strings.Replace(valid, tc.from, tc.to, 1)
			calls := 0
			preparer := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { calls++; return imageBytesReader([]byte("not observed")), nil })
			prepared, err := preparedContainer(t, fields, preparer)
			o := prepared.Observation()
			if err != nil || o.Status != model.Failed || calls != 0 || o.CheckImage != nil || len(o.Diagnostics) == 0 {
				t.Fatalf("invalid configuration reached acquisition: %+v err=%v calls=%d", o, err, calls)
			}
			for _, diagnostic := range o.Diagnostics {
				if diagnostic.Source.ContentID != prepared.Input().Document().File.ContentID() || !strings.HasPrefix(diagnostic.Source.Field, prepared.Input().Field()) {
					t.Fatal("diagnostic lost captured source")
				}
			}
		})
	}
}

func TestContainerMetadataAcquisitionFailuresNeverBecomePassingEvidence(t *testing.T) {
	fields := containerFields(filepath.Join(testkit.TempDir(t), "engine"))
	for _, kind := range []string{"no factory", "factory error", "nil reader", "image error", "image mismatch", "file error", "cleanup error with bytes", "empty digest", "empty file", "oversized", "cancelled before", "cancelled during"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			peer := imageBytesReader([]byte("actual regular image file"))
			calls := 0
			factory := ImageReaderFactory(func(EngineConfig) (ImageReader, error) { calls++; return peer, nil })
			switch kind {
			case "no factory":
				factory = nil
			case "factory error":
				factory = func(EngineConfig) (ImageReader, error) { return nil, errors.New("engine unavailable") }
			case "nil reader":
				factory = func(EngineConfig) (ImageReader, error) { return nil, nil }
			case "image error":
				peer.image = func(context.Context, string) (string, error) { return "", errors.New("inspect failed") }
			case "image mismatch":
				peer.image = func(context.Context, string) (string, error) { return "sha256:" + strings.Repeat("b", 64), nil }
			case "file error", "cleanup error with bytes":
				peer.file = func(context.Context, string, string, int64) (container.ImageFileObservation, error) {
					return container.ImageFileObservation{SHA256: sha256.Sum256([]byte("unconfirmed")), Size: 11}, errors.New(kind)
				}
			case "empty digest", "empty file", "oversized":
				peer.file = func(context.Context, string, string, int64) (container.ImageFileObservation, error) {
					file := container.ImageFileObservation{SHA256: sha256.Sum256([]byte("observed")), Size: 8}
					switch kind {
					case "empty digest":
						file.SHA256 = [32]byte{}
					case "empty file":
						file.Size = 0
					case "oversized":
						file.Size = container.MaxImageFileObservationBytes + 1
					}
					return file, nil
				}
			case "cancelled before":
				cancel()
			case "cancelled during":
				original := peer.file
				peer.file = func(ctx context.Context, image, file string, max int64) (container.ImageFileObservation, error) {
					cancel()
					return original(ctx, image, file, max)
				}
			}
			// Acquire an actual opaque ProfileInput through Apply, without I/O.
			files, defaults := m1SectionParents(t, "container", fields)
			recorder := &recordingPreparer{}
			_, err := files.PrepareSettings(context.Background(), defaults, nil, recorder)
			if err != nil || recorder.input.Section() != "container" {
				t.Fatal("selected container input missing", err)
			}
			o, err := NewMetadataPreparer(factory).PrepareCheckProfile(ctx, recorder.input)
			want := model.NotCompleted
			if kind == "image mismatch" || kind == "empty digest" || kind == "empty file" || kind == "oversized" {
				want = model.Failed
			}
			if o.Status != want || o.CheckImage != nil || len(o.Files) != 0 || len(o.Diagnostics) == 0 || (want == model.NotCompleted && err == nil) || (kind == "cancelled before" && calls != 0) {
				t.Fatalf("failure=%s observation=%+v err=%v", kind, o, err)
			}
		})
	}
}

func TestContainerProfileProjectionRefusesDifferentPreparedBindings(t *testing.T) {
	fields := containerFields(filepath.Join(testkit.TempDir(t), "engine"))
	preparer := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { return imageBytesReader([]byte("whole file")), nil })
	for _, kind := range []string{"contract name", "contract version", "image", "executable", "oversized", "input binding", "missing image", "mixed files", "extra instruction", "zero"} {
		t.Run(kind, func(t *testing.T) {
			var prepared apply.PreparedProfile
			if kind != "zero" {
				var err error
				prepared, err = preparedContainer(t, fields, checkPreparerFunc(func(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
					o, err := preparer.PrepareCheckProfile(ctx, input)
					switch kind {
					case "contract name":
						o.Contract.Name = "different-contract"
					case "contract version":
						o.Contract.Version++
					case "image":
						o.CheckImage.ImageID = "sha256:" + strings.Repeat("b", 64)
					case "executable":
						o.CheckImage.Executable.Path = "/other/check"
					case "oversized":
						o.CheckImage.Executable.Size = container.MaxImageFileObservationBytes + 1
					case "input binding":
						o.InputBinding = apply.ProfileBinding{}
					case "missing image":
						o.CheckImage = nil
					case "mixed files":
						o.Files = []apply.ProfileMetadataFile{{}}
					case "extra instruction":
						o.Instructions = []apply.SourceLocation{{Boundary: "uncaptured", Path: fixturePath(t, "instruction.md")}}
					}
					return o, err
				}))
				if err != nil {
					t.Fatal(err)
				}
			}
			profile, err := ContainerProfileFromPrepared(prepared)
			if err == nil || !reflect.DeepEqual(profile, ContainerProfile{}) {
				t.Fatalf("mismatched %s projected: %+v err=%v", kind, profile, err)
			}
		})
	}
}

func TestContainerMetadataRequiresExplicitEmptyListsAndPreservesLegacyProcess(t *testing.T) {
	fields := containerFields(filepath.Join(testkit.TempDir(t), "engine"))
	fields = strings.ReplaceAll(fields, "['literal prefix', '']", "[]")
	fields = strings.ReplaceAll(fields, "['A=literal value', 'EMPTY=']", "[]")
	preparer := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { return imageBytesReader([]byte("file")), nil })
	prepared, err := preparedContainer(t, fields, preparer)
	profile, projectionErr := ContainerProfileFromPrepared(prepared)
	if err != nil || projectionErr != nil || len(profile.ArgumentPrefix) != 0 || len(profile.Env) != 0 {
		t.Fatalf("explicit empty lists refused: %+v %v %v", profile, err, projectionErr)
	}
	// A process section still reaches the unchanged process/v1 validation.
	input := processInput(t, "profileVersion: 2\nexecutable: not-read\nsha256: bad\n")
	preparer.factory = func(EngineConfig) (ImageReader, error) {
		t.Fatal("process profile reached image reader")
		return nil, nil
	}
	o, err := preparer.PrepareCheckProfile(context.Background(), input)
	if err != nil || o.Status != model.Failed || o.Contract.Name != "markitect.process-check" || o.CheckImage != nil {
		t.Fatalf("legacy process path changed: %+v %v", o, err)
	}
}

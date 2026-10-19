package container

import (
	"context"
	"strings"
	"testing"
)

func TestInspectVolumeDoesNotTreatFailureAsAbsence(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Code: 1, Err: "engine unavailable"})
	if _, err := e.InspectVolume(context.Background(), "work"); err == nil {
		t.Fatal("failure became absence")
	}
}

func TestCreateVolumeExclusiveRefusesExistingBeforeCreate(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: "work\n"}, engineStep{Out: `[{"Name":"work","Labels":{"owner":"other"}}]`})
	v, err := e.CreateVolumeExclusive(context.Background(), "work", map[string]string{"owner": "mine"})
	if err == nil || !v.Exists || len(calls()) != 2 {
		t.Fatalf("observation=%+v err=%v calls=%q", v, err, calls())
	}
}

func TestCreateVolumeExclusiveRefusesRacingCreation(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{}, engineStep{Out: "work"}, engineStep{Out: "work"}, engineStep{Out: `[{"Name":"work","Labels":{"owner":"mine","io.markitect.volume-creation":"other"}}]`})
	_, err := e.CreateVolumeExclusive(context.Background(), "work", map[string]string{"owner": "mine"})
	if err == nil || !strings.Contains(err.Error(), "uncertain") || len(calls()) != 4 {
		t.Fatalf("err=%v calls=%q", err, calls())
	}
	if !containsArg(calls()[1], "volume") || !containsArg(calls()[1], "create") {
		t.Fatalf("create calls %q", calls())
	}
}

func TestCreateVolumeExclusiveObservesFreshOwnership(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{}, engineStep{Out: "work"}, engineStep{Out: "work"}, engineStep{Out: `[{"Name":"work","Labels":{"owner":"mine","io.markitect.volume-creation":"CREATION_TOKEN"}}]`})
	v, err := e.CreateVolumeExclusive(context.Background(), "work", map[string]string{"owner": "mine"})
	if err != nil || !v.Exists || v.Name != "work" || v.Labels["owner"] != "mine" || len(v.Labels[creationLabel]) != 64 {
		t.Fatalf("observation=%+v err=%v", v, err)
	}
}

func TestVolumeMountCensusIncludesStoppedReadOnlyAndExactVolume(t *testing.T) {
	other := strings.Repeat("b", 64)
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID + "\n" + other}, engineStep{Out: `[{"Id":"` + commandID + `","State":{"Running":false},"Mounts":[{"Type":"volume","Name":"work","RW":false}]}]`}, engineStep{Out: `[{"Id":"` + other + `","Mounts":[{"Type":"volume","Name":"work-other"},{"Type":"bind","Name":"work"}]}]`})
	ids, err := e.VolumeMounts(context.Background(), "work")
	if err != nil || len(ids) != 1 || ids[0] != commandID {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	// The engine narrows the listing to containers whose mounts match the
	// name; the exact volume mount is still checked on each one.
	if first := calls()[0]; !containsArg(first, "--all") || !containsArg(first, "volume=work") {
		t.Fatalf("census listing %q", first)
	}
}

func TestVolumeMountCensusFailsWhenOneInspectFails(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Code: 1, Err: "inspect failed"}, engineStep{Out: commandID})
	if _, err := e.VolumeMounts(context.Background(), "work"); err == nil {
		t.Fatal("partial census was accepted")
	}
}

// A container removed between the listing and its inspection mounts nothing;
// a fresh listing without it proves that (a session container that was just
// stopped and removed, or any other container on the engine).
func TestVolumeMountCensusSkipsAContainerRemovedAfterTheListing(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Code: 1, Err: "Error response from daemon: No such container"}, engineStep{Out: ""})
	ids, err := e.VolumeMounts(context.Background(), "work")
	if err != nil || len(ids) != 0 || len(calls()) != 3 || !containsArg(calls()[2], "id="+commandID) {
		t.Fatalf("ids=%v err=%v calls=%q", ids, err, calls())
	}
}

func TestVolumeMountCensusRefusesMissingMountObservation(t *testing.T) {
	for _, mounts := range []string{"", `,"Mounts":null`} {
		t.Run(mounts, func(t *testing.T) {
			e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: `[{"Id":"` + commandID + `"` + mounts + `}]`})
			if _, err := e.VolumeMounts(context.Background(), "work"); err == nil {
				t.Fatal("incomplete mount observation accepted")
			}
		})
	}
}

func TestVolumeMountCensusRequiresCompleteEntries(t *testing.T) {
	for _, mounts := range []string{`[{}]`, `[null]`, `[{"Type":"volume"}]`} {
		t.Run(mounts, func(t *testing.T) {
			e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: `[{"Id":"` + commandID + `","Mounts":` + mounts + `}]`})
			if _, err := e.VolumeMounts(context.Background(), "work"); err == nil {
				t.Fatal("incomplete mount entry accepted")
			}
		})
	}
	for _, mounts := range []string{`[]`, `[{"Type":"bind"}]`} {
		t.Run(mounts, func(t *testing.T) {
			e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: `[{"Id":"` + commandID + `","Mounts":` + mounts + `}]`})
			ids, err := e.VolumeMounts(context.Background(), "work")
			if err != nil || len(ids) != 0 {
				t.Fatalf("mounts=%v err=%v", ids, err)
			}
		})
	}
}

func TestRequireVolumeSubpathsNeedsDockerServer26(t *testing.T) {
	for _, tc := range []struct {
		name string
		step engineStep
		ok   bool
	}{
		{"docker 29", engineStep{Out: "29.0.1\n"}, true},
		{"docker 26", engineStep{Out: "26.0.0\n"}, true},
		{"release candidate of 27", engineStep{Out: "27.0.0-rc.1\n"}, true},
		{"docker 25", engineStep{Out: "25.0.5\n"}, false},
		{"docker 20", engineStep{Out: "20.10.24\n"}, false},
		{"not understood", engineStep{Out: "podman\n"}, false},
		{"empty", engineStep{}, false},
		{"command failed", engineStep{Code: 1, Err: "daemon not running"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, calls := fakeCommandEngine(t, tc.step)
			err := e.RequireVolumeSubpaths(context.Background())
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want ok %v", err, tc.ok)
			}
			if got := calls(); len(got) != 1 || strings.Join(got[0], " ") != "version --format {{.Server.Version}}" {
				t.Fatalf("calls %q", got)
			}
			if err != nil && !strings.Contains(err.Error(), "26") && tc.name != "command failed" {
				t.Fatalf("message must name the required version: %v", err)
			}
		})
	}
}

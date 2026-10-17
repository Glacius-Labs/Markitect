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

func TestVolumeMountCensusIncludesStoppedReadOnlyAndExactVolume(t *testing.T) {
	other := strings.Repeat("b", 64)
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID + "\n" + other}, engineStep{Out: `[{"Id":"` + commandID + `","State":{"Running":false},"Mounts":[{"Type":"volume","Name":"work","RW":false}]}]`}, engineStep{Out: `[{"Id":"` + other + `","Mounts":[{"Type":"volume","Name":"work-other"},{"Type":"bind","Name":"work"}]}]`})
	ids, err := e.VolumeMounts(context.Background(), "work")
	if err != nil || len(ids) != 1 || ids[0] != commandID {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}

func TestVolumeMountCensusFailsWhenOneInspectFails(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Code: 1, Err: "container disappeared"})
	if _, err := e.VolumeMounts(context.Background(), "work"); err == nil {
		t.Fatal("partial census was accepted")
	}
}

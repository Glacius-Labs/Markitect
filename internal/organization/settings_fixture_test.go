package organization_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// These are explicit acquired-fact fixtures, not YAML parser or provider proof.
func settingSource(file, field string) organization.SettingSource {
	return organization.SettingSource{File: file, Field: field, ContentID: snapshot.ContentID(sha256.Sum256([]byte(file + " captured fixture bytes")))}
}

func setting[T any](value T, field string) organization.Setting[T] {
	return organization.Setting[T]{Value: value, Source: settingSource("execution.yaml", field)}
}

func settingPointer[T any](value T, field string) *organization.Setting[T] {
	result := setting(value, field)
	return &result
}

func directory(t *testing.T, name string) snapshot.DirectoryBoundary {
	t.Helper()
	if name == "." {
		return snapshot.RootDirectory()
	}
	path, err := snapshot.NewPath(name)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := snapshot.DirectoryAt(path)
	if err != nil {
		t.Fatal(err)
	}
	return boundary
}

func instruction(id string, bytes []byte) organization.InstructionFact {
	return organization.InstructionFact{
		ID: id, Bytes: bytes,
		Source: organization.SettingSource{File: id + ".txt", ContentID: snapshot.ContentID(sha256.Sum256(bytes))},
	}
}

func settingsFixture(t *testing.T) organization.SettingsFacts {
	t.Helper()
	return organization.SettingsFacts{
		Defaults: organization.SettingsDefaults{
			Manager: organization.RoleDefaults{
				Profile: setting("writer", "/defaults/profiles/manager"), Instructions: setting([]string{"style"}, "/defaults/instructions/manager"),
			},
			Reviewer: organization.RoleDefaults{
				Profile: setting("critic", "/defaults/profiles/reviewer"), Instructions: setting([]string{"review"}, "/defaults/instructions/reviewer"),
			},
			StepTimeout: setting(5*time.Minute, "/defaults/stepTimeout"), ReviewRounds: setting(2, "/defaults/reviewRounds"),
			CreationRoots:        setting([]snapshot.DirectoryBoundary{directory(t, "real")}, "/defaults/creationRoots"),
			BatchNoChangeReviews: setting(false, "/defaults/batchNoChangeReviews"),
		},
		Ceilings: organization.SettingsCeilings{
			StepTimeout: setting(10*time.Minute, "/limits/maxStepTimeout"), ReviewRounds: setting(3, "/limits/maxReviewRounds"),
			CreationRoots: setting([]snapshot.DirectoryBoundary{directory(t, "real"), directory(t, "docs")}, "/limits/creationRoots"),
		},
		Profiles: []organization.ProfileFact{
			{Name: "writer", Identity: "prepared-writer-v1", Source: settingSource("operator-profiles.yaml", "/agentProfiles/writer")},
			{Name: "critic", Identity: "prepared-critic-v1", Source: settingSource("operator-profiles.yaml", "/agentProfiles/critic")},
			{Name: "alternate", Identity: "prepared-alternate-v1", Source: settingSource("operator-profiles.yaml", "/agentProfiles/alternate")},
		},
		Instructions: []organization.InstructionFact{
			instruction("style", []byte("\xef\xbb\xbfUse concise reports.\r\nKeep source references.")),
			instruction("review", []byte("Read independently.\n")),
			instruction("local", []byte("Report local observations.")),
			instruction("empty", nil),
		},
	}
}

func settingsTopology(t *testing.T) organization.Topology {
	t.Helper()
	return requireTopology(t, compileFixture(t, localFixture(t, map[string][]string{
		"lending": {"item"}, "lending/reminders": {"item"},
	}, "empty", "software")))
}

func requireSettings(t *testing.T, topology organization.Topology, facts organization.SettingsFacts) organization.Settings {
	t.Helper()
	settings, err := organization.ResolveSettings(topology, facts)
	if err != nil {
		t.Fatal(err)
	}
	return settings
}

func managerSettings(t *testing.T, settings organization.Settings, path string) organization.ManagerSettings {
	t.Helper()
	manager, found := settings.Manager(path)
	if !found {
		t.Fatalf("no settings for local Manager %q", path)
	}
	return manager
}

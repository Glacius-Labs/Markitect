package release

import (
	"bytes"
	"testing"
)

func TestStrategyDefaultsAssetIsPreservedInBothSourcePackages(t *testing.T) {
	root := rootLayoutFixture(t)
	asset := []byte("\xef\xbb\xbfapiVersion: markitect.strategy-defaults/v1\r\nstrategy: recursive-apply-v1\r\ndefaults:\r\n  rejectionLimit: 3\r\n")
	writeLayoutFile(t, root, embeddedStrategyDefaultsPath, string(asset))
	writeLayoutFile(t, root, "internal/apply/nearby.yaml", "not a packaged source\n")

	collectArchives := func() ([]byte, []byte) {
		t.Helper()
		liveFiles, err := collect(root)
		if err != nil {
			t.Fatal(err)
		}
		liveArchive, err := makeArchive(liveFiles)
		if err != nil {
			t.Fatal(err)
		}

		fixedFiles, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
		if err != nil {
			t.Fatal(err)
		}
		selected, err := collectModuleSnapshot(fixedFiles)
		if err != nil {
			t.Fatal(err)
		}
		fixedArchive, err := makeArchive(selected)
		if err != nil {
			t.Fatal(err)
		}
		return liveArchive, fixedArchive
	}

	liveArchive, fixedArchive := collectArchives()
	if !bytes.Equal(liveArchive, fixedArchive) {
		t.Fatal("live and fixed-source packaging selected different files or bytes")
	}
	contents := archiveContents(t, liveArchive)
	if got, ok := contents[embeddedStrategyDefaultsPath]; !ok || !bytes.Equal(got, asset) {
		t.Fatalf("archived strategy defaults = %q, present %t; want exact original bytes %q", got, ok, asset)
	}
	if _, ok := contents["internal/apply/nearby.yaml"]; ok {
		t.Fatal("packaging included unrelated YAML beside the embedded asset")
	}
	before := digestBytes(liveArchive)

	changedAsset := append(append([]byte(nil), asset...), []byte("  batchNoChangeReviews: false\r\n")...)
	writeLayoutFile(t, root, embeddedStrategyDefaultsPath, string(changedAsset))
	changedLive, changedFixed := collectArchives()
	if !bytes.Equal(changedLive, changedFixed) {
		t.Fatal("live and fixed-source packaging diverged after an asset change")
	}
	if digestBytes(changedLive) == before {
		t.Fatal("changing embedded strategy defaults did not change the source archive digest")
	}
}

func TestStrategyDefaultsFixedSnapshotRejectsNonRegularAsset(t *testing.T) {
	root := rootLayoutFixture(t)
	writeLayoutFile(t, root, embeddedStrategyDefaultsPath, "strategy: recursive-apply-v1\n")
	source := fixedSourceSnapshot(t, root)
	source.Modes[embeddedStrategyDefaultsPath] = "120000"
	if _, err := snapshotModuleFiles(source); err == nil {
		t.Fatal("fixed-source packaging accepted a symlink-mode embedded asset")
	}
}

func TestStrategyDefaultsAssetPreservationStillValidatesText(t *testing.T) {
	for name, input := range map[string][]byte{
		"invalid UTF-8": {0xff},
		"NUL":           {'a', 0, 'b'},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeTextSource(embeddedStrategyDefaultsPath, input); err == nil {
				t.Fatal("invalid embedded text was accepted")
			}
		})
	}
}

package libraries

import "testing"

func TestEveryLibraryHasFiles(t *testing.T) {
	for _, name := range Names() {
		files, ok := Files(name)
		if !ok || len(files) == 0 {
			t.Fatalf("standard Library %s has no files", name)
		}
	}
}

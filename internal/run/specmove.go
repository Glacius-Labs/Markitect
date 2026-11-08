package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The move of the spec folder: markitect up to 0.24.0 kept the spec in
// .markitect/model/, from when it was called the canonical model, and update
// moves it to .markitect/spec/. The record keeps its verdicts: a line is keyed by
// its Definition's ID, the path below the folder, and its hashes are of content.
// Only evidence can name a file of the spec; update renames it, which changes no
// fingerprint, since evidence is no part of it. A file input never names one: a
// registry row never names a file of Markitect's folder.

// SpecMove is what MoveSpec did or, in a dry run, would do.
type SpecMove struct {
	// Moved says the spec went from project.OldSpecDir to project.SpecDir.
	Moved bool
	// Lines are the lines of the record whose evidence named a file below
	// project.OldSpecDir and now names it below project.SpecDir.
	Lines int
}

// Changed says whether the move changed anything.
func (m SpecMove) Changed() bool { return m.Moved || m.Lines > 0 }

// MoveSpec moves the spec of the project at root from project.OldSpecDir to
// project.SpecDir and renames the files the record's evidence names there. It
// refuses, and changes nothing, when both folders hold files. The record is
// renamed whenever it names the old folder, so a line that a merge brings in
// from a branch that has not moved yet is renamed by the next update too. With
// dry it changes nothing and says what it would do.
func MoveSpec(root string, dry bool) (SpecMove, error) {
	var mv SpecMove
	from := filepath.Join(root, filepath.FromSlash(project.OldSpecDir))
	to := filepath.Join(root, filepath.FromSlash(project.SpecDir))
	old, err := holdsFiles(from)
	if err != nil {
		return mv, err
	}
	if old {
		both, err := holdsFiles(to)
		if err != nil {
			return mv, err
		}
		if both {
			return mv, fmt.Errorf("both %s and %s hold files, and a project has one spec; %s is where an earlier markitect kept it: move what you still need from there into %s, remove %s, and run markitect update again", project.OldSpecDir, project.SpecDir, project.OldSpecDir, project.SpecDir, project.OldSpecDir)
		}
		mv.Moved = true
	}
	data, err := readRecordFile(root)
	if err != nil {
		if !mv.Moved {
			// Nothing to move: a record update cannot read is status's to report.
			return SpecMove{}, nil
		}
		return SpecMove{}, err
	}
	record, lines := moveRecord(data)
	mv.Lines = lines
	if dry {
		return mv, nil
	}
	if mv.Moved {
		// What is left of the new folder holds no file: empty folders at most.
		if err := os.RemoveAll(to); err != nil {
			return SpecMove{}, err
		}
		if err := renameFolder(from, to); err != nil {
			return SpecMove{}, fmt.Errorf("could not move %s to %s: %w; close what holds a file there and run markitect update again", project.OldSpecDir, project.SpecDir, err)
		}
	}
	if lines > 0 {
		if err := writeAtomic(filepath.Join(root, filepath.FromSlash(project.Dir)), path.Base(project.ReviewFile), record); err != nil {
			return mv, fmt.Errorf("moved the spec, but could not write %s: %w; run markitect update again", project.ReviewFile, err)
		}
	}
	return mv, nil
}

// holdsFiles says whether a folder holds a file at any depth. A missing folder
// holds none, and a file or a symbolic link where the folder should be counts as
// one.
func holdsFiles(dir string) (bool, error) {
	found := false
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}

// renameFolder renames a folder, trying again for a moment: Windows refuses a
// rename while another program, such as a virus scanner, has a file open there.
func renameFolder(from, to string) error {
	var err error
	for i := 0; i < 20; i++ {
		if err = os.Rename(from, to); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

// moveRecord renames the files below project.OldSpecDir that the lines of a
// record name, and keeps every other line as it is, in its place and with its
// line end, one it cannot read included. It returns the record and how many
// lines it changed.
func moveRecord(data []byte) ([]byte, int) {
	raw := strings.SplitAfter(string(data), "\n")
	n := 0
	for i, r := range raw {
		text := strings.TrimRight(r, "\r\n")
		if !strings.Contains(text, project.OldSpecDir+"/") {
			continue
		}
		var l Line
		if json.Unmarshal([]byte(text), &l) != nil || !validLine(l) || !moveLine(&l) {
			continue
		}
		moved, err := marshal(l)
		if err != nil {
			continue
		}
		raw[i] = moved + r[len(text):]
		n++
	}
	return []byte(strings.Join(raw, "")), n
}

// moveLine renames the files below project.OldSpecDir that a line names as
// evidence, in its list and its inputs, and says whether it named any.
func moveLine(l *Line) bool {
	changed := false
	inputs := Inputs{}
	for key, hash := range l.Inputs {
		if file, ok := strings.CutPrefix(key, InputEvidence); ok {
			key = InputEvidence + project.SpecPath(file)
		}
		if _, had := l.Inputs[key]; !had {
			changed = true
		}
		inputs[key] = hash
	}
	l.Inputs = inputs
	for i, e := range l.Evidence {
		if moved := project.SpecPath(e); moved != e {
			l.Evidence[i] = moved
			changed = true
		}
	}
	return changed
}

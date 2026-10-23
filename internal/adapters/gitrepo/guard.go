package gitrepo

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

func (a *acquisition) guard() project.PublicationGuard {
	hash := sha256.New()
	write := func(text string) {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(text)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(text))
	}
	write("gitrepo-capture/1")
	write(a.git.path)
	write(a.git.gitDir)
	write(a.git.commonDir)
	write(a.git.format)
	write(a.head)
	write(a.raw.Census.Digest().String())
	write(strconv.Itoa(len(a.models)))
	for _, m := range a.models {
		s := m.target
		write(s.Selection.Boundary)
		write(s.Selection.ManifestPath.String())
		write(s.RepositoryArea.String())
		write(s.Root.String())
		write(s.Source.Origin)
		write(s.Source.ManifestPath)
		write(strconv.Itoa(len(s.Source.Entries)))
		for _, entry := range s.Source.Entries {
			write(entry.Path)
			write(string(entry.Kind))
			write(string(entry.Bytes))
		}
		write(s.Files.Digest().String())
	}
	for _, group := range [][]project.SourceFile{a.raw.Execution, a.raw.Registry, a.raw.Ignore, a.raw.RuntimeProfiles, a.raw.Instructions} {
		write(strconv.Itoa(len(group)))
		for _, file := range group {
			write(file.Location.Boundary)
			write(file.Location.Path.String())
			write(file.RepositoryPath.String())
			write(file.File.Path().String())
			write(strconv.Itoa(int(file.File.Mode())))
			write(file.File.ContentID().String())
		}
	}
	return project.PublicationGuard("gitrepo-capture/1:" + hex.EncodeToString(hash.Sum(nil)))
}

package gitrepo

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
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
	write("gitrepo-capture/3")
	write(a.git.path)
	write(a.git.gitDir)
	write(a.git.commonDir)
	write(a.git.format)
	write(a.head)
	write(a.branch)
	write(a.index)
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
		// Entry order is an acquisition detail, not source content: a committed
		// Capture lists Git paths in byte order ("a-b.yaml" before "a/c.yaml"),
		// while publication's working-tree walk descends into "a/" first. Bind
		// every entry, duplicates included, in one canonical order.
		entries := slices.Clone(s.Source.Entries)
		slices.SortFunc(entries, func(x, y model.Entry) int {
			return cmp.Or(strings.Compare(x.Path, y.Path), strings.Compare(string(x.Kind), string(y.Kind)), bytes.Compare(x.Bytes, y.Bytes))
		})
		write(strconv.Itoa(len(entries)))
		for _, entry := range entries {
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
	return project.PublicationGuard("gitrepo-capture/3:" + hex.EncodeToString(hash.Sum(nil)))
}

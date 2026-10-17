package gitrepo

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// The trusted runtime image supplies Node. Names stay raw Linux bytes until
// snapshot validates them; lstat never follows symlinks. Directories count
// against the scan budget but have no tree entry. Only the owner execute bit
// contributes to Git's regular-file mode. A terminal frame proves completion.
const workspaceCensusScript = `
const fs = require('fs');
const maxEntries = Number(process.argv[1]), maxBytes = Number(process.argv[2]);
let count = 0, total = 0;
function write(b) { let n=0; while(n<b.length) n+=fs.writeSync(1,b,n,b.length-n); }
function frame(v) { const b=Buffer.from(JSON.stringify(v)); const h=Buffer.alloc(4); h.writeUInt32BE(b.length); write(h); write(b); }
function walk(dir, rel) {
  const directory = fs.opendirSync(dir,{encoding:'buffer',bufferSize:32});
  try { let item; while((item=directory.readSync())!==null) {
    const name = item.name;
    if(name.equals(Buffer.from('.git'))) continue;
    const p = rel.length ? Buffer.concat([rel,Buffer.from('/'),name]) : name;
    const abs = Buffer.concat([Buffer.from('/work/'),p]);
    const st = fs.lstatSync(abs);
    if(st.isDirectory() && p.equals(Buffer.from('.markitect/state'))) continue;
    if(++count > maxEntries) throw Error('workspace entry limit');
    const type = st.isFile() ? 'file' : st.isDirectory() ? 'directory' : 'unsupported';
    const size = st.isFile() ? st.size : 0;
    if(!Number.isSafeInteger(size) || size < 0 || total+size > maxBytes) throw Error('workspace byte limit');
    total += size;
    frame({path:p.toString('base64'),type,size,execute:(st.mode & 64)!==0});
    if(st.isFile()) {
      const fd=fs.openSync(abs,fs.constants.O_RDONLY|fs.constants.O_NOFOLLOW);
      try {
        const opened=fs.fstatSync(fd);
        if(!opened.isFile() || opened.ino!==st.ino || opened.dev!==st.dev || opened.size!==size || opened.mode!==st.mode || opened.mtimeMs!==st.mtimeMs || opened.ctimeMs!==st.ctimeMs) throw Error('workspace changed before read');
        const buf=Buffer.alloc(65536); let remaining=size;
        while(remaining) { const n=fs.readSync(fd,buf,0,Math.min(buf.length,remaining),null); if(!n) throw Error('workspace shortened'); write(buf.subarray(0,n)); remaining-=n; }
        const after=fs.fstatSync(fd);
        if(after.size!==size || after.mode!==st.mode || after.mtimeMs!==st.mtimeMs || after.ctimeMs!==st.ctimeMs) throw Error('workspace changed during read');
      } finally { fs.closeSync(fd); }
    } else if(st.isDirectory()) walk(abs,p);
  }} finally { directory.closeSync(); }
}
walk(Buffer.from('/work'),Buffer.alloc(0)); frame({end:true});
`

type workspaceFrame struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
	Execute bool   `json:"execute"`
	End     bool   `json:"end"`
}

func workspaceExcluded(p string, directory bool) bool {
	for _, part := range strings.Split(p, "/") {
		if part == ".git" {
			return true
		}
	}
	return strings.HasPrefix(p, ".markitect/state/") || directory && p == ".markitect/state"
}

func readWorkspaceCensus(reader io.Reader, request run.HarvestRequest) (result run.WorkspaceHarvest, err error) {
	result.ID = request.ID
	var entries []snapshot.Entry
	defer func() {
		slices.Sort(result.Unsupported)
		var treeErr error
		result.Tree, treeErr = snapshot.NewTree(entries)
		err = errors.Join(err, treeErr)
		if err != nil {
			result.Uncertain, result.Reason = true, err.Error()
		}
	}()
	var total int64
	seen := make(map[string]bool)
	for count := 0; ; count++ {
		var header [4]byte
		if _, err = io.ReadFull(reader, header[:]); err != nil {
			return result, fmt.Errorf("workspace census missing terminal frame: %w", err)
		}
		size := binary.BigEndian.Uint32(header[:])
		if size == 0 || size > 16384 {
			return result, fmt.Errorf("workspace census invalid frame size %d", size)
		}
		data := make([]byte, size)
		if _, err = io.ReadFull(reader, data); err != nil {
			return result, err
		}
		var f workspaceFrame
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&f); err != nil {
			return result, err
		}
		if f.End {
			if f.Path != "" || f.Type != "" || f.Size != 0 || f.Execute {
				return result, fmt.Errorf("workspace census invalid terminal frame")
			}
			var tail [1]byte
			n, tailErr := reader.Read(tail[:])
			if n != 0 || tailErr != io.EOF {
				return result, fmt.Errorf("workspace census has trailing or incomplete output")
			}
			return result, nil
		}
		if count >= request.MaxEntries || f.Size < 0 || f.Size > request.MaxBytes-total || uint64(f.Size) > uint64(^uint(0)>>1) {
			return result, fmt.Errorf("workspace census exceeds observation limit")
		}
		raw, decodeErr := base64.StdEncoding.Strict().DecodeString(f.Path)
		if decodeErr != nil || len(raw) == 0 || seen[string(raw)] {
			return result, fmt.Errorf("workspace census invalid or duplicate name")
		}
		seen[string(raw)] = true
		p := string(raw)
		if workspaceExcluded(p, f.Type == "directory") {
			return result, fmt.Errorf("workspace census unexpectedly included excluded path")
		}
		if f.Type != "file" && f.Size != 0 {
			return result, fmt.Errorf("workspace census non-file has content")
		}
		if f.Type != "file" && f.Type != "directory" && f.Type != "unsupported" {
			return result, fmt.Errorf("workspace census unknown entry type")
		}
		checked, pathErr := snapshot.NewPath(p)
		if pathErr != nil || f.Type == "unsupported" {
			if !utf8.Valid(raw) {
				p = "bytes:" + hex.EncodeToString(raw)
			}
			result.Unsupported = append(result.Unsupported, p)
		}
		if f.Type != "file" {
			continue
		}
		total += f.Size
		if pathErr != nil {
			_, err = io.CopyN(io.Discard, reader, f.Size)
			if err != nil {
				return result, err
			}
			continue
		}
		content := make([]byte, int(f.Size))
		if _, err = io.ReadFull(reader, content); err != nil {
			return result, err
		}
		mode := snapshot.Regular
		if f.Execute {
			mode = snapshot.Executable
		}
		entry, entryErr := snapshot.NewEntry(checked, mode, content)
		if entryErr != nil {
			return result, entryErr
		}
		entries = append(entries, entry)
	}
}

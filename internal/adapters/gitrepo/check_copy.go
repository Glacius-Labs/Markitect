package gitrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type disposableCopy struct {
	location run.CopyLocation
	tree     snapshot.TreeID
	close    func(context.Context) error
}

func (c *disposableCopy) Location() run.CopyLocation      { return c.location }
func (c *disposableCopy) Tree() snapshot.TreeID           { return c.tree }
func (c *disposableCopy) Close(ctx context.Context) error { return c.close(ctx) }

func (c *localCopy) Materialize(ctx context.Context, tree snapshot.Tree) (run.DisposableCopy, error) {
	dir, err := os.MkdirTemp(c.service.root, "check-")
	if err != nil {
		return nil, err
	}
	copy := &disposableCopy{location: run.CopyLocation{Directory: dir}, close: func(ctx context.Context) error { return removePrivate(ctx, c.service.root, dir) }}
	for _, e := range tree.Entries() {
		if err = ctx.Err(); err != nil {
			return copy, err
		}
		if runtime.GOOS == "windows" && e.Mode() == snapshot.Executable {
			return copy, fmt.Errorf("local Windows check copy cannot establish POSIX executable mode")
		}
		file := filepath.Join(dir, filepath.FromSlash(e.Path().String()))
		if err = os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return copy, err
		}
		mode := os.FileMode(0600)
		if e.Mode() == snapshot.Executable {
			mode = 0700
		}
		if err = os.WriteFile(file, e.Bytes(), mode); err != nil {
			return copy, err
		}
	}
	var observed []snapshot.Entry
	err = filepath.WalkDir(dir, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported check input")
		}
		rel, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		path, err := snapshot.NewPath(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		mode := snapshot.Regular
		if info.Mode()&0111 != 0 {
			mode = snapshot.Executable
		}
		e, err := snapshot.NewEntry(path, mode, data)
		if err == nil {
			observed = append(observed, e)
		}
		return err
	})
	if err != nil {
		return copy, err
	}
	actual, err := snapshot.NewTree(observed)
	if err != nil {
		return copy, err
	}
	if !actual.Equal(tree) {
		return copy, fmt.Errorf("check input readback differs from immutable tree")
	}
	copy.tree = actual.Digest()
	return copy, nil
}

func (c *linuxCopy) Materialize(ctx context.Context, tree snapshot.Tree) (run.DisposableCopy, error) {
	key, err := token()
	if err != nil {
		return nil, err
	}
	subpath := "checks/" + key
	directory := linuxRunRoot + "/" + subpath
	copy := &disposableCopy{location: run.CopyLocation{Volume: c.service.volume, Subpath: subpath}, close: func(ctx context.Context) error { return c.service.remove(ctx, directory) }}
	// The held Run service performs filesystem work; arbitrary project check
	// images need neither Node nor Git. Transfer and all observations are bounded.
	type file struct {
		Path       string
		Executable bool
		Data       []byte
	}
	var files []file
	for _, e := range tree.Entries() {
		files = append(files, file{e.Path().String(), e.Mode() == snapshot.Executable, e.Bytes()})
	}
	data, err := json.Marshal(files)
	if err != nil {
		return copy, err
	}
	if int64(len(data)) > c.service.adapter.MaxTransferBytes {
		return copy, fmt.Errorf("check tree exceeds transfer bound")
	}
	out := &boundedOutput{limit: c.service.adapter.MaxTransferBytes}
	script := `const fs=require('fs'),p=require('path');const root=process.argv[1];fs.mkdirSync(root,{mode:0o700});const files=JSON.parse(fs.readFileSync(0,'utf8'))||[];for(const f of files){const path=p.join(root,f.Path);fs.mkdirSync(p.dirname(path),{recursive:true,mode:0o700});fs.writeFileSync(path,Buffer.from(f.Data,'base64'),{flag:'wx',mode:f.Executable?0o755:0o644})}const result=[];function walk(rel){for(const n of fs.readdirSync(p.join(root,rel))){const path=p.join(root,rel,n),s=fs.lstatSync(path),name=p.join(rel,n);if(s.isSymbolicLink()||(!s.isDirectory()&&!s.isFile()))throw Error('unsupported check input');if(s.isDirectory())walk(name);else result.push({Path:name,Executable:!!(s.mode&0o111),Data:fs.readFileSync(path).toString('base64')})}}walk('');process.stdout.write(JSON.stringify(result))`
	if err = c.service.mkdir(ctx, linuxRunRoot+"/checks"); err != nil {
		return copy, err
	}
	if err = c.service.node(ctx, script, []string{directory}, bytes.NewReader(data), out); err != nil {
		return copy, err
	}
	if out.limitExceeded {
		return copy, fmt.Errorf("check readback exceeds transfer bound")
	}
	var readback []file
	if err = json.Unmarshal(out.Bytes(), &readback); err != nil {
		return copy, err
	}
	var entries []snapshot.Entry
	for _, f := range readback {
		p, err := snapshot.NewPath(f.Path)
		if err != nil {
			return copy, err
		}
		mode := snapshot.Regular
		if f.Executable {
			mode = snapshot.Executable
		}
		e, err := snapshot.NewEntry(p, mode, f.Data)
		if err != nil {
			return copy, err
		}
		entries = append(entries, e)
	}
	actual, err := snapshot.NewTree(entries)
	if err != nil {
		return copy, err
	}
	if !actual.Equal(tree) {
		return copy, fmt.Errorf("Linux check tree readback differs from fixed input")
	}
	copy.tree = actual.Digest()
	return copy, nil
}

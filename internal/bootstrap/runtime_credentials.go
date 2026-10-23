package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
)

// This matches the Codex adapter's admitted login/protocol bound. Credentials
// are read whole, never truncated into a different credential.
const maxRuntimeCredentialBytes = 1 << 20

var (
	errRuntimeCredential = errors.New("runtime credential mapping or token refused")
	errCredentialPath    = errors.New("runtime credential path is redirected or unsafe")
	errCredentialType    = errors.New("runtime credential file is not regular")
	errCredentialLimit   = errors.New("runtime credential file is empty or exceeds its byte limit")
	errCredentialChanged = errors.New("runtime credential file or parent changed during reading")
	errCredentialRead    = errors.New("runtime credential file I/O failed")
)

// credentialSource declares dedicated runtime input. A path cannot prove its
// provenance: the operator must supply a runtime credential, never desktop auth.
type credentialSource struct {
	Provider string `json:"provider"`
	Mode     string `json:"mode"`
	Path     string `json:"path"`
}

// newCredentialResolvers validates and copies explicit mappings without any
// filesystem or environment access. Only invoking a returned resolver reads its
// selected file. The adapters own credential validation and session copies;
// neither this constructor nor its resolvers rotate or rewrite source files.
func newCredentialResolvers(sources map[string]credentialSource) (codexappserver.CredentialResolver, claudecode.CredentialResolver, error) {
	if len(sources) == 0 {
		return nil, nil, errRuntimeCredential
	}
	bound := make(map[string]credentialSource, len(sources))
	for ref, source := range sources {
		validMode := source.Provider == "codex-app-server" && source.Mode == "dedicated-runtime-login" || source.Provider == "claude-code" && source.Mode == "dedicated-runtime-token"
		if ref == "" || strings.TrimSpace(ref) != ref || strings.ContainsFunc(ref, unicode.IsControl) || !utf8.ValidString(ref) || !validMode {
			return nil, nil, errRuntimeCredential
		}
		if !credentialPath(source.Path) {
			return nil, nil, errCredentialPath
		}
		bound[ref] = source
	}
	read := func(ctx context.Context, ref, provider string) (credentialSource, []byte, error) {
		source, found := bound[ref]
		if !found || source.Provider != provider || ctx == nil {
			return credentialSource{}, nil, errRuntimeCredential
		}
		data, err := readRuntimeCredential(ctx, source.Path)
		if err != nil {
			// Paths, reference strings and underlying errors may contain sensitive
			// text. No native error or credential byte escapes this boundary.
			return credentialSource{}, nil, credentialReadError(err)
		}
		return source, data, nil
	}
	codex := func(ctx context.Context, ref string) (codexappserver.RuntimeCredential, error) {
		source, data, err := read(ctx, ref, "codex-app-server")
		if err != nil {
			return codexappserver.RuntimeCredential{}, err
		}
		return codexappserver.RuntimeCredential{Mode: source.Mode, RuntimeLogin: json.RawMessage(data)}, nil
	}
	claude := func(ctx context.Context, ref string) (claudecode.RuntimeCredential, error) {
		_, data, err := read(ctx, ref, "claude-code")
		if err != nil {
			return claudecode.RuntimeCredential{}, err
		}
		if !credentialToken(data) {
			return claudecode.RuntimeCredential{}, errRuntimeCredential
		}
		return claudecode.RuntimeCredential{Token: string(data)}, nil
	}
	return codex, claude, nil
}

func credentialToken(data []byte) bool {
	return len(data) > 0 && utf8.Valid(data) && !strings.ContainsAny(string(data), "\r\n\x00")
}

func credentialReadError(err error) error {
	for _, known := range []error{errCredentialPath, errCredentialType, errCredentialLimit, errCredentialChanged, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, known) {
			return known
		}
	}
	return errCredentialRead
}

func credentialPath(literal string) bool {
	if !filepath.IsAbs(literal) || filepath.Clean(literal) != literal || strings.ContainsFunc(literal, unicode.IsControl) || !utf8.ValidString(literal) {
		return false
	}
	volume := filepath.VolumeName(literal)
	// Dedicated native files must not name a Windows device or network share.
	if strings.HasPrefix(volume, `\\`) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(literal, volume+string(filepath.Separator)), string(filepath.Separator))
	if len(parts) > 256 {
		return false
	}
	for _, part := range parts {
		if part == "" || !filepath.IsLocal(part) || strings.ContainsRune(part, ':') || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return false
		}
	}
	return true
}

type credentialDirectory struct {
	root *os.Root
	info os.FileInfo
	name string
}

// readRuntimeCredential holds every directory while opening its exact child,
// rejects redirects, and rechecks identities after the bounded read. Native
// errors stay private so no selected path or raw file content enters diagnostics.
func readRuntimeCredential(ctx context.Context, literal string) (data []byte, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	anchor := filepath.VolumeName(literal) + string(filepath.Separator)
	parts := strings.Split(strings.TrimPrefix(literal, anchor), string(filepath.Separator))
	info, err := os.Lstat(anchor)
	if err != nil {
		return nil, err
	}
	if credentialRedirected(info) || !info.IsDir() {
		return nil, errCredentialPath
	}
	root, err := os.OpenRoot(anchor)
	if err != nil {
		return nil, err
	}
	dirs := []credentialDirectory{{root: root, info: info}}
	defer func() {
		for i := len(dirs) - 1; i >= 0; i-- {
			err = errors.Join(err, dirs[i].root.Close())
		}
		if err != nil {
			data = nil
		}
	}()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, errCredentialChanged
	}
	for _, part := range parts[:len(parts)-1] {
		if err := credentialExactEntry(ctx, root, part); err != nil {
			return nil, err
		}
		before, err := root.Lstat(part)
		if err != nil {
			return nil, err
		}
		if credentialRedirected(before) || !before.IsDir() {
			return nil, errCredentialPath
		}
		next, err := root.OpenRoot(part)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, credentialDirectory{root: next, info: before, name: part})
		after, err := next.Stat(".")
		if err != nil || !os.SameFile(before, after) {
			return nil, errCredentialChanged
		}
		root = next
	}
	name := parts[len(parts)-1]
	if err := credentialExactEntry(ctx, root, name); err != nil {
		return nil, err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if credentialRedirected(before) {
		return nil, errCredentialPath
	}
	if !before.Mode().IsRegular() {
		return nil, errCredentialType
	}
	if before.Size() <= 0 || before.Size() > maxRuntimeCredentialBytes {
		return nil, errCredentialLimit
	}
	f, err := openCredentialFile(root, name)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	opened, err = f.Stat()
	if err != nil || !credentialSameFile(before, opened) || !opened.Mode().IsRegular() {
		return nil, errCredentialChanged
	}
	data, err = io.ReadAll(io.LimitReader(f, maxRuntimeCredentialBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxRuntimeCredentialBytes {
		return nil, errCredentialLimit
	}
	if int64(len(data)) != opened.Size() {
		return nil, errCredentialChanged
	}
	after, err := f.Stat()
	if err != nil || !credentialSameFile(opened, after) {
		return nil, errCredentialChanged
	}
	atPath, err := root.Lstat(name)
	if err != nil || credentialRedirected(atPath) || !credentialSameFile(after, atPath) {
		return nil, errCredentialChanged
	}
	current, err := os.Lstat(anchor)
	if err != nil || credentialRedirected(current) || !os.SameFile(dirs[0].info, current) {
		return nil, errCredentialChanged
	}
	for i := 1; i < len(dirs); i++ {
		current, err := dirs[i-1].root.Lstat(dirs[i].name)
		if err != nil || credentialRedirected(current) || !os.SameFile(dirs[i].info, current) {
			return nil, errCredentialChanged
		}
	}
	return data, ctx.Err()
}

func credentialSameFile(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.Mode() == b.Mode() && a.ModTime().Equal(b.ModTime())
}

func credentialExactEntry(ctx context.Context, root *os.Root, want string) (err error) {
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	for count := 0; count <= 100000; {
		if err := ctx.Err(); err != nil {
			return err
		}
		names, err := f.Readdirnames(256)
		count += len(names)
		for _, name := range names {
			if name == want {
				return nil
			}
		}
		if err != nil {
			return err
		}
	}
	return errRuntimeCredential
}

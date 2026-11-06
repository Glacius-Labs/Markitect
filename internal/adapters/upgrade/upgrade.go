// Package upgrade finds the newest Markitect release, downloads the binary for
// this platform, checks it against the release's checksums and replaces the
// running executable. It is the one part of Markitect that uses the network, and
// only when the user runs markitect update.
package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// DefaultBase is where the releases live.
const DefaultBase = "https://github.com/Glacius-Labs/Markitect/releases"

// BaseEnv names an environment variable that points the upgrade, and the
// plugin's launcher, to a mirror of the releases page with the same layout.
const BaseEnv = "MARKITECT_RELEASES_URL"

const (
	maxBinary = 256 << 20
	maxSums   = 1 << 20
)

// ErrNoRelease is returned when the project has published no release.
var ErrNoRelease = errors.New("there is no release yet")

// platforms are the operating system and architecture pairs a release has a
// binary for.
var platforms = map[string]bool{
	"linux/amd64": true, "linux/arm64": true,
	"darwin/amd64": true, "darwin/arm64": true,
	"windows/amd64": true, "windows/arm64": true,
}

// Client talks to a releases page. The zero value uses GitHub and this platform.
type Client struct {
	Base         string       // the releases URL; DefaultBase when empty
	HTTP         *http.Client // a client with a generous timeout when nil
	GOOS, GOARCH string       // this platform when empty
}

// New returns a Client for GitHub, or for the mirror BaseEnv names.
func New() Client {
	return Client{Base: strings.TrimRight(os.Getenv(BaseEnv), "/")}
}

func (c Client) base() string {
	if c.Base == "" {
		return DefaultBase
	}
	return strings.TrimRight(c.Base, "/")
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

func (c Client) platform() (goos, goarch string) {
	goos, goarch = c.GOOS, c.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return goos, goarch
}

// AssetName is the name of the binary of a release for a platform.
func AssetName(version, goos, goarch string) string {
	name := "markitect-v" + version + "-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// ChecksumsName is the name of the file that lists the SHA-256 of every binary
// of a release, in the format of sha256sum.
func ChecksumsName(version string) string { return "markitect-v" + version + "-checksums.txt" }

// tagPattern is a release tag: a version with an optional pre-release part and no
// build metadata, as the release workflow accepts it.
var tagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)

// Latest returns the version of the newest release, such as "0.15.0". It reads
// the tag from the redirect of the releases page, which needs no API token and
// has no rate limit worth knowing.
func (c Client) Latest(ctx context.Context) (string, error) {
	// Every update asks; a server that never answers must not hold it for the
	// ten minutes a download may take.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	hc := *c.http()
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	// A project that moved sends the latest page to the new address of the latest
	// page: that redirect is followed, a few times at most. The redirect that names
	// a tag is the answer.
	next := c.base() + "/latest"
	for hop := 0; hop < 4; hop++ {
		loc, err := c.redirect(ctx, hc, next)
		if err != nil {
			return "", err
		}
		i := strings.LastIndex(loc.Path, "/tag/")
		if i >= 0 {
			tag, err := url.PathUnescape(loc.Path[i+len("/tag/"):])
			if err != nil || !tagPattern.MatchString(tag) {
				return "", fmt.Errorf("the latest release has the tag %q, which is no version", loc.Path[i+len("/tag/"):])
			}
			return strings.TrimPrefix(tag, "v"), nil
		}
		if !strings.HasSuffix(loc.Path, "/latest") {
			return "", ErrNoRelease // GitHub sends the releases page of a project without one
		}
		next = loc.String()
	}
	return "", fmt.Errorf("%s redirects too often", c.base()+"/latest")
}

// redirect asks for the page at address and returns where it redirects to.
func (c Client) redirect(ctx context.Context, hc http.Client, address string) (*url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNoRelease
	case resp.StatusCode/100 != 3:
		return nil, fmt.Errorf("%s answered %s", address, resp.Status)
	}
	loc, err := resp.Location()
	if err != nil {
		return nil, fmt.Errorf("%s redirects nowhere", address)
	}
	return loc, nil
}

// Install downloads the binary of the release version for this platform and
// puts it in the place of the executable exe. It checks the download against
// the release's checksums first and leaves exe alone if anything fails.
func (c Client) Install(ctx context.Context, version, exe string) error {
	goos, goarch := c.platform()
	if !platforms[goos+"/"+goarch] {
		return fmt.Errorf("there is no release binary for %s/%s", goos, goarch)
	}
	asset := AssetName(version, goos, goarch)
	tag := "v" + version

	sums, err := c.fetch(ctx, tag, ChecksumsName(version), maxSums)
	if err != nil {
		return err
	}
	var want string
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == asset && len(f[0]) == sha256.Size*2 {
			want = strings.ToLower(f[0])
		}
	}
	if want == "" {
		return fmt.Errorf("the checksums of release %s do not list %s", tag, asset)
	}

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".markitect-new-*")
	if err != nil {
		return fmt.Errorf("cannot write next to %s: %w", exe, err)
	}
	done := false
	defer func() {
		tmp.Close()
		if !done {
			os.Remove(tmp.Name())
		}
	}()
	got, err := c.download(ctx, tag, asset, tmp)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s does not match its checksum (got %s, the release lists %s); nothing was replaced", asset, got, want)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	mode := os.FileMode(0o755)
	if st, err := os.Stat(exe); err == nil {
		mode = st.Mode().Perm() | 0o111
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	if err := replace(tmp.Name(), exe, goos == "windows"); err != nil {
		return fmt.Errorf("cannot replace %s: %w", exe, err)
	}
	done = true
	return nil
}

// replace puts the file next in the place of exe. Windows lets a program rename
// its own running executable but not overwrite it, so there the old one steps
// aside first and is deleted later, when it is no longer running.
func replace(next, exe string, aside bool) error {
	if !aside {
		return os.Rename(next, exe)
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(next, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	os.Remove(old) // fails while the old program still runs; the next update tries again
	return nil
}

func (c Client) get(ctx context.Context, tag, name string) (*http.Response, error) {
	u := c.base() + "/download/" + url.PathEscape(tag) + "/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("release %s has no %s", tag, name)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s answered %s", u, resp.Status)
	}
	return resp, nil
}

func (c Client) fetch(ctx context.Context, tag, name string, limit int64) ([]byte, error) {
	resp, err := c.get(ctx, tag, name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s is larger than %d bytes", name, limit)
	}
	return data, nil
}

// download copies the asset into w and returns its SHA-256.
func (c Client) download(ctx context.Context, tag, name string, w io.Writer) (string, error) {
	resp, err := c.get(ctx, tag, name)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(resp.Body, maxBinary+1))
	if err != nil {
		return "", err
	}
	if n > maxBinary {
		return "", fmt.Errorf("%s is larger than %d bytes", name, int64(maxBinary))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Run starts the executable exe with args, connected to stdout and stderr, and
// returns its exit code. update uses it to let the new binary finish the work.
func (Client) Run(ctx context.Context, exe string, args []string, stdout, stderr io.Writer) (int, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}

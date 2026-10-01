package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func validateTagTarget(r Runner, ctx context.Context, tag, commit string) error {
	var ref refInfo
	if err := apiJSON(r, ctx, "repos/"+repository+"/git/ref/tags/"+url.PathEscape(tag), &ref); err != nil {
		return fmt.Errorf("read remote tag: %w", err)
	}
	kind, sha := ref.Object.Type, ref.Object.SHA
	for depth := 0; kind == "tag"; depth++ {
		if depth >= 8 {
			return errors.New("annotated tag chain exceeds eight objects")
		}
		var annotated tagInfo
		if err := apiJSON(r, ctx, "repos/"+repository+"/git/tags/"+sha, &annotated); err != nil {
			return fmt.Errorf("read annotated tag: %w", err)
		}
		kind, sha = annotated.Object.Type, annotated.Object.SHA
	}
	if kind != "commit" || sha != commit || !commitPattern.MatchString(sha) {
		return errors.New("current remote tag does not resolve to the selected workflow source commit")
	}
	return nil
}

func getRelease(r Runner, ctx context.Context, tag string) (releaseInfo, error) {
	var x releaseInfo
	err := apiJSON(r, ctx, "repos/"+repository+"/releases/tags/"+url.PathEscape(tag), &x)
	return x, err
}
func checkImmutable(r Runner, ctx context.Context) error {
	var x struct {
		Enabled bool `json:"enabled"`
	}
	if err := apiJSON(r, ctx, "repos/"+repository+"/immutable-releases", &x); err != nil {
		return err
	}
	if !x.Enabled {
		return errors.New("immutable releases are not enabled")
	}
	return nil
}

func createDraft(r Runner, ctx context.Context, tag, commit string) (releaseInfo, error) {
	body := "Private Markitect release for source commit " + commit + ". See the provenance YAML asset for the source, run, toolchain, and release asset digests."
	// Structured gh fields preserve each value as one process argument without
	// shell interpolation or temporary note files.
	prerelease := isPrereleaseTag(tag)
	args := []string{"api", "--hostname", "github.com", "repos/" + repository + "/releases", "-X", "POST", "-f", "tag_name=" + tag, "-f", "target_commitish=" + commit, "-f", "name=Markitect " + tag, "-f", "body=" + body, "-F", "draft=true", "-F", "prerelease=" + strconv.FormatBool(prerelease)}
	out, err := r.Run(ctx, "", args...)
	if err != nil {
		return releaseInfo{}, err
	}
	var x releaseInfo
	if err := json.Unmarshal(out, &x); err != nil {
		return x, fmt.Errorf("decode created draft response: %w", err)
	}
	if x.ID == 0 || x.TagName != tag || !x.Draft || len(x.Assets) != 0 {
		return x, errors.New("GitHub did not create the expected empty draft")
	}
	return x, nil
}

func verifyDraftAssets(r Runner, ctx context.Context, id int64, files namedBytes) error {
	var x []remoteAsset
	if err := apiJSON(r, ctx, fmt.Sprintf("repos/%s/releases/%d/assets", repository, id), &x); err != nil {
		return err
	}
	return compareRemoteAssets(x, files)
}
func compareRemoteAssets(remote []remoteAsset, files namedBytes) error {
	if len(remote) != 4 || len(files) != 4 {
		return errors.New("release must contain exactly four assets")
	}
	got := map[string]string{}
	for _, a := range remote {
		if _, dup := got[a.Name]; dup {
			return fmt.Errorf("duplicate uploaded asset %q", a.Name)
		}
		got[a.Name] = a.Digest
	}
	for name, data := range files {
		want := "sha256:" + digest(data)
		if got[name] != want {
			return fmt.Errorf("uploaded asset %s digest mismatch", name)
		}
	}
	return nil
}
func patchDraft(r Runner, ctx context.Context, id int64, tag string) error {
	out, err := r.Run(ctx, "", "api", "--hostname", "github.com", fmt.Sprintf("repos/%s/releases/%d", repository, id), "-X", "PATCH", "-F", "draft=false")
	if err != nil {
		return err
	}
	var x releaseInfo
	if err := json.Unmarshal(out, &x); err != nil {
		return err
	}
	if x.ID != id || x.TagName != tag {
		return fmt.Errorf("GitHub returned an unexpected release ID %d during publication", x.ID)
	}
	if x.Draft {
		return errDraftStillPending
	}
	return nil
}

func isPrereleaseTag(tag string) bool {
	version := strings.TrimPrefix(tag, "v")
	version = strings.SplitN(version, "+", 2)[0]
	return strings.Contains(version, "-")
}
func verifyPublished(r Runner, ctx context.Context, tag, commit string, id int64, files namedBytes, assetDir string) ([]string, error) {
	var x releaseInfo
	if err := apiJSON(r, ctx, fmt.Sprintf("repos/%s/releases/%d", repository, id), &x); err != nil {
		return nil, err
	}
	if x.ID != id || x.TagName != tag || x.Draft || !x.Immutable {
		return nil, errors.New("published release is not the expected immutable release")
	}
	if err := compareRemoteAssets(x.Assets, files); err != nil {
		return nil, err
	}
	steps := []string{}
	attempts, err := verifyAttestation(r, ctx, "release", "verify", tag, "--repo", "github.com/"+repository)
	if err != nil {
		return nil, fmt.Errorf("release attestation verification failed: %w", err)
	}
	if attempts > 1 {
		steps = append(steps, fmt.Sprintf("release attestation verified on attempt %d", attempts))
	}
	for _, name := range assetNames(tag) {
		attempts, err := verifyAttestation(r, ctx, "release", "verify-asset", tag, filepath.Join(assetDir, name), "--repo", "github.com/"+repository)
		if err != nil {
			return nil, fmt.Errorf("asset attestation verification failed for %s: %w", name, err)
		}
		if attempts > 1 {
			steps = append(steps, fmt.Sprintf("asset %s attestation verified on attempt %d", name, attempts))
		}
	}
	if err := checkImmutable(r, ctx); err != nil {
		return nil, fmt.Errorf("immutable-release setting changed after publication: %w", err)
	}
	if err := validateTagTarget(r, ctx, tag, commit); err != nil {
		return nil, err
	}
	return steps, nil
}

// GitHub may expose an immutable release before its attestation is queryable.
// Retry only read-only attestation checks; mismatched release metadata and asset
// digests above always fail immediately.
func verifyAttestation(r Runner, ctx context.Context, args ...string) (int, error) {
	delays := [...]time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
	failures := make([]error, 0, len(delays)+1)
	for i, delay := range delays {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return i, errors.Join(append(failures, ctx.Err())...)
			}
		}
		if err := ctx.Err(); err != nil {
			return i, errors.Join(append(failures, err)...)
		}
		if err := call(r, ctx, "", args...); err == nil {
			return i + 1, nil
		} else {
			failures = append(failures, fmt.Errorf("attempt %d/%d: %w", i+1, len(delays), err))
		}
		if err := ctx.Err(); err != nil {
			return i + 1, errors.Join(append(failures, err)...)
		}
	}
	return len(delays), errors.Join(failures...)
}

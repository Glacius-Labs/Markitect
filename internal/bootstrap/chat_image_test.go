package bootstrap

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
)

func imageSource(recipe string) fstest.MapFS {
	return fstest.MapFS{
		"go.mod":        {Data: []byte("module x\n")},
		chatImageRecipe: {Data: []byte(recipe)},
	}
}

// fakeImages is an engine that holds tags and records builds.
type fakeImages struct {
	mu       sync.Mutex
	tags     []string
	builds   []container.ImageBuild
	contexts [][]byte
	removed  []string
	listErr  error
	buildErr error
	// during, when set, runs inside a build.
	during func(ctx context.Context, progress func(string))
}

func (f *fakeImages) Images(context.Context, string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.tags...), f.listErr
}

func (f *fakeImages) ImageID(_ context.Context, ref string) (string, error) {
	return "sha256:" + ref, nil
}

func (f *fakeImages) BuildImage(ctx context.Context, b container.ImageBuild) error {
	data, err := io.ReadAll(b.Context)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.builds = append(f.builds, b)
	f.contexts = append(f.contexts, data)
	f.mu.Unlock()
	if f.during != nil {
		f.during(ctx, b.Progress)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.buildErr != nil {
		return f.buildErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tags = append(f.tags, b.Tag)
	return nil
}

func (f *fakeImages) RemoveImage(_ context.Context, ref string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, ref)
	return nil
}

func TestTheFirstChatBuildsTheImageAndLaterOnesDoNot(t *testing.T) {
	engine := &fakeImages{during: func(_ context.Context, progress func(string)) {
		progress("#1 [internal] load build definition from Dockerfile")
		progress("#7 [build 3/5] RUN   go   build " + strings.Repeat("x", 100))
		progress("#7 2.31 compiling")
	}}
	image := newChatImage(imageSource("FROM scratch\n"))
	var said []string
	id, err := image.ensure(context.Background(), engine, func(s string) { said = append(said, s) })
	if err != nil {
		t.Fatal(err)
	}
	tag, _ := image.tag()
	if !strings.HasPrefix(tag, "markitect-runtime:app-") || id != "sha256:"+tag {
		t.Fatalf("image %q, id %q", tag, id)
	}
	if len(engine.builds) != 1 || engine.builds[0].Tag != tag || engine.builds[0].Dockerfile != chatImageRecipe {
		t.Fatalf("builds %+v", engine.builds)
	}
	files := map[string]string{}
	r := tar.NewReader(bytes.NewReader(engine.contexts[0]))
	for {
		h, err := r.Next()
		if err != nil {
			break
		}
		data, _ := io.ReadAll(r)
		files[h.Name] = string(data)
	}
	if files[chatImageRecipe] != "FROM scratch\n" || files["go.mod"] != "module x\n" || len(files) != 2 {
		t.Errorf("the build context holds %v", files)
	}
	if len(said) != 3 || said[0] != chatImagePreparing || !strings.HasPrefix(said[1], chatImagePreparing+": RUN go build x") || !strings.HasSuffix(said[1], "…") || said[2] != "Starting the chat's agent" {
		t.Errorf("the chat said %q", said)
	}

	said = nil
	if _, err := image.ensure(context.Background(), engine, func(s string) { said = append(said, s) }); err != nil {
		t.Fatal(err)
	}
	if len(engine.builds) != 1 || len(said) != 0 {
		t.Errorf("a later chat built %d times and said %q", len(engine.builds), said)
	}
}

func TestANewSourceBuildsANewImageAndRemovesTheOld(t *testing.T) {
	engine := &fakeImages{}
	first := newChatImage(imageSource("FROM scratch\n"))
	if _, err := first.ensure(context.Background(), engine, func(string) {}); err != nil {
		t.Fatal(err)
	}
	old, _ := first.tag()
	second := newChatImage(imageSource("FROM scratch\n# changed\n"))
	if _, err := second.ensure(context.Background(), engine, func(string) {}); err != nil {
		t.Fatal(err)
	}
	now, _ := second.tag()
	if now == old || len(engine.builds) != 2 {
		t.Fatalf("tags %q %q, %d builds", old, now, len(engine.builds))
	}
	if strings.Join(engine.removed, ",") != old {
		t.Errorf("removed %q, want %q", engine.removed, old)
	}
	same := newChatImage(imageSource("FROM scratch\n"))
	if tag, _ := same.tag(); tag != old {
		t.Errorf("the same source got another name: %q %q", tag, old)
	}
}

func TestAFailedBuildSaysWhyAndKeepsTheOldImage(t *testing.T) {
	engine := &fakeImages{tags: []string{"markitect-runtime:app-old"}, buildErr: errors.New("no network")}
	_, err := newChatImage(imageSource("FROM scratch\n")).ensure(context.Background(), engine, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "the agent image") || !strings.Contains(err.Error(), "no network") {
		t.Fatalf("error %v", err)
	}
	if len(engine.removed) != 0 {
		t.Errorf("removed %q", engine.removed)
	}
}

func TestAnEngineThatDoesNotAnswerBuildsNothing(t *testing.T) {
	engine := &fakeImages{listErr: errors.New("the daemon is not running")}
	_, err := newChatImage(imageSource("FROM scratch\n")).ensure(context.Background(), engine, func(string) { t.Error("it said it prepares") })
	if err == nil || !strings.Contains(err.Error(), "the daemon is not running") || len(engine.builds) != 0 {
		t.Fatalf("error %v, %d builds", err, len(engine.builds))
	}
}

func TestStoppingEndsTheBuild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine := &fakeImages{during: func(ctx context.Context, _ func(string)) {
		cancel()
		<-ctx.Done()
	}}
	_, err := newChatImage(imageSource("FROM scratch\n")).ensure(ctx, engine, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
	if len(engine.tags) != 0 {
		t.Errorf("a stopped build left %q", engine.tags)
	}
}

func TestTwoChatsBuildOnceAndAWaitingOneCanStop(t *testing.T) {
	building, finish := make(chan struct{}), make(chan struct{})
	engine := &fakeImages{during: func(context.Context, func(string)) {
		close(building)
		<-finish
	}}
	image := newChatImage(imageSource("FROM scratch\n"))
	first := make(chan error, 1)
	go func() {
		_, err := image.ensure(context.Background(), engine, func(string) {})
		first <- err
	}()
	<-building

	stopped, stop := context.WithCancel(context.Background())
	waiting := make(chan error, 1)
	said := make(chan struct{})
	go func() {
		_, err := image.ensure(stopped, engine, func(string) { close(said) })
		waiting <- err
	}()
	<-said
	stop()
	if err := <-waiting; !errors.Is(err, context.Canceled) {
		t.Fatalf("the waiting chat: %v", err)
	}

	second := make(chan error, 1)
	go func() {
		_, err := image.ensure(context.Background(), engine, func(string) {})
		second <- err
	}()
	close(finish)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if len(engine.builds) != 1 {
		t.Errorf("%d builds", len(engine.builds))
	}
}

func TestASetManagerImageWinsOverBuilding(t *testing.T) {
	// Any executable serves: the engine call fails, and no build may follow.
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip(err)
	}
	login := t.TempDir() + string(os.PathSeparator) + "login.json"
	if err := os.WriteFile(login, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_ENGINE", git)
	t.Setenv("MARKITECT_LOGIN", login)
	t.Setenv("MARKITECT_NETWORK", "chosen")
	t.Setenv("MARKITECT_MANAGER_IMAGE", "mine:1")
	_, _, err = agentConfigWith(context.Background(), func(context.Context, *container.Engine) (string, error) {
		t.Error("it built although an image is set")
		return "", nil
	})
	if err == nil || !strings.Contains(err.Error(), "the Manager image mine:1") {
		t.Fatalf("error %v", err)
	}
}

func TestSettingsNameTheImageTheAppBuilds(t *testing.T) {
	t.Setenv("MARKITECT_MANAGER_IMAGE", "")
	image := newChatImage(imageSource("FROM scratch\n"))
	s := appSettings(image)
	tag, _ := image.tag()
	if s.Agents.Image != "" || s.Agents.ImageBuilt != tag {
		t.Fatalf("settings %+v", s.Agents)
	}
	for _, p := range s.Agents.Problems {
		if strings.Contains(p, "Manager image") {
			t.Errorf("a problem although the app builds it: %q", p)
		}
	}
	t.Setenv("MARKITECT_MANAGER_IMAGE", "mine:1")
	if s := appSettings(image); s.Agents.Image != "mine:1" || s.Agents.ImageBuilt != "" {
		t.Errorf("with one set: %+v", s.Agents)
	}
}

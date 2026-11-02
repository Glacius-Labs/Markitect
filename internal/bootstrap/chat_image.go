package bootstrap

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
)

const (
	// chatImageName is the repository of the image the app builds for its chats.
	chatImageName = "markitect-runtime"
	// chatImageRecipe is the recipe's path inside the source.
	chatImageRecipe = "packaging/runtime/Dockerfile"
	// chatImageLimit bounds one build, which on a first run also downloads its
	// base images.
	chatImageLimit = time.Hour
	// chatImagePreparing is what a chat says while its image is built.
	chatImagePreparing = "Preparing the agent image, first time only"
	// chatImageStepWidth bounds the build step a chat shows after that.
	chatImageStepWidth = 70
)

// chatBuildStep finds the step a line of a plain build log starts, such as
// "#7 [build 3/5] RUN go build"; the engine's own bookkeeping has no count.
var chatBuildStep = regexp.MustCompile(`^#\d+ \[[^\]]*\d+/\d+\] (.+)$`)

// imageEngine is what building the chats' image asks of the container engine.
type imageEngine interface {
	Images(ctx context.Context, pattern string) ([]string, error)
	ImageID(ctx context.Context, ref string) (string, error)
	BuildImage(ctx context.Context, b container.ImageBuild) error
	RemoveImage(ctx context.Context, ref string) error
}

// chatImage is the image the native app's chats run in when no Manager image
// is set. A chat reads and changes the model, so it needs no project's
// toolchain, and one image serves every project. The app builds it on the
// first chat that needs it, from the source it carries, and names it by that
// source, so that it cannot drift from the app: a new version builds a new image
// and removes the old one.
type chatImage struct {
	source fs.FS
	// name is the image's repository.
	name string
	tag  func() (string, error)
	// turn lets one build run at a time; a chat that waits for it can stop.
	turn chan struct{}
}

func newChatImage(source fs.FS) *chatImage {
	if source == nil {
		return nil
	}
	i := &chatImage{source: source, name: chatImageName, turn: make(chan struct{}, 1)}
	i.tag = sync.OnceValues(func() (string, error) {
		context, err := imageContext(source)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(context)
		return i.name + ":app-" + hex.EncodeToString(sum[:8]), nil
	})
	return i
}

// ensure returns the image's ID, building the image first when the engine does
// not have it. doing says what is happening, for the chat that waits.
func (i *chatImage) ensure(ctx context.Context, engine imageEngine, doing func(string)) (string, error) {
	if doing == nil {
		doing = func(string) {}
	}
	tag, err := i.tag()
	if err != nil {
		return "", fmt.Errorf("the agent image: %w", err)
	}
	built, err := engine.Images(ctx, i.name+":app-*")
	if err != nil {
		return "", fmt.Errorf("the agent image: %w", err)
	}
	if !slices.Contains(built, tag) {
		if err := i.build(ctx, engine, tag, doing); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", fmt.Errorf("the agent image: %w", err)
		}
		doing("Starting the chat's agent")
	}
	return engine.ImageID(ctx, tag)
}

func (i *chatImage) build(ctx context.Context, engine imageEngine, tag string, doing func(string)) error {
	doing(chatImagePreparing)
	select {
	case i.turn <- struct{}{}:
		defer func() { <-i.turn }()
	case <-ctx.Done():
		return ctx.Err()
	}
	// Another chat may have built it while this one waited.
	built, err := engine.Images(ctx, i.name+":app-*")
	if err != nil {
		return err
	}
	if slices.Contains(built, tag) {
		return nil
	}
	source, err := imageContext(i.source)
	if err != nil {
		return err
	}
	bounded, cancel := context.WithTimeout(ctx, chatImageLimit)
	defer cancel()
	err = engine.BuildImage(bounded, container.ImageBuild{
		Tag: tag, Dockerfile: chatImageRecipe, Context: bytes.NewReader(source),
		Progress: func(line string) {
			if m := chatBuildStep.FindStringSubmatch(line); m != nil {
				doing(chatImagePreparing + ": " + shortStep(m[1]))
			}
		},
	})
	if err != nil {
		return err
	}
	// An image a running agent still uses stays until a later build; agents
	// run by image ID, so nothing depends on these names. A chat stopped now
	// does not keep the old image.
	remove, done := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer done()
	for _, old := range built {
		_ = engine.RemoveImage(remove, old)
	}
	return nil
}

func shortStep(step string) string {
	step = strings.Join(strings.Fields(step), " ")
	if len(step) > chatImageStepWidth {
		return strings.ToValidUTF8(step[:chatImageStepWidth], "") + "…"
	}
	return step
}

// imageContext packs the source as a tar build context. Its bytes depend only
// on the files, so they also name the image.
func imageContext(source fs.FS) ([]byte, error) {
	var out bytes.Buffer
	w := tar.NewWriter(&out)
	err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		if err := w.WriteHeader(&tar.Header{Name: path, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

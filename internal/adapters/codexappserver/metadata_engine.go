package codexappserver

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
)

// EngineConfig comes only from the selected, source-bound profile section.
type EngineConfig struct{ Executable string }

// ImageReader exposes only inspection and copying from a never-started image.
// ReadImageFile must return complete regular-file bytes, bounded to 1 MiB.
type ImageReader interface {
	ImageID(context.Context, string) (string, error)
	ReadImageFile(context.Context, string, string) ([]byte, error)
}

type ImageReaderFactory func(EngineConfig) (ImageReader, error)

// NewEngineImageReader binds the explicitly configured engine. NewEngine only
// validates its path; actual engine inspection/copy occurs during preparation.
func NewEngineImageReader(config EngineConfig) (ImageReader, error) {
	return container.NewEngine(config.Executable)
}

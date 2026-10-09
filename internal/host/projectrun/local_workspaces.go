package projectrun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/host/projectworkspace"
)

const (
	localWorkspaceMaxFiles      = 10000
	localWorkspaceMaxFileBytes  = 16 << 20
	localWorkspaceMaxTotalBytes = 256 << 20
)

// LocalWorkspaceService lazily owns private Git candidate workspaces in the
// current user's cache. Constructing it does not touch disk or the adopting
// repository; storage is initialized on the first lifecycle operation.
type LocalWorkspaceService struct {
	storageRoot string
	mu          sync.Mutex
	service     *projectworkspace.GitService
	initErr     error
}

func NewLocalWorkspaceService() *LocalWorkspaceService {
	return &LocalWorkspaceService{}
}

func (s *LocalWorkspaceService) Prepare(ctx context.Context, request projectworkspace.Request) (projectworkspace.Handle, error) {
	service, err := s.get()
	if err != nil {
		return projectworkspace.Handle{}, err
	}
	return service.Prepare(ctx, request)
}

func (s *LocalWorkspaceService) PrepareCandidate(ctx context.Context, request projectworkspace.Request, overlay []projectworkspace.Change, digest string) (projectworkspace.Handle, error) {
	service, err := s.get()
	if err != nil {
		return projectworkspace.Handle{}, err
	}
	return service.PrepareCandidate(ctx, request, overlay, digest)
}

func (s *LocalWorkspaceService) Harvest(ctx context.Context, handle projectworkspace.Handle) (projectworkspace.Delta, error) {
	service, err := s.get()
	if err != nil {
		return projectworkspace.Delta{}, err
	}
	return service.Harvest(ctx, handle)
}

func (s *LocalWorkspaceService) Close(ctx context.Context, handle projectworkspace.Handle) error {
	service, err := s.get()
	if err != nil {
		return err
	}
	return service.Close(ctx, handle)
}

// ReopenCandidate reattaches only to a handle accepted by the underlying
// service's persisted ownership record and terminal-state checks.
func (s *LocalWorkspaceService) ReopenCandidate(ctx context.Context, request projectworkspace.Request, handle projectworkspace.Handle, overlay []projectworkspace.Change, digest string, terminalConfirmed bool) error {
	service, err := s.get()
	if err != nil {
		return err
	}
	return service.ReopenCandidate(ctx, request, handle, overlay, digest, terminalConfirmed)
}

func (s *LocalWorkspaceService) get() (*projectworkspace.GitService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.service != nil || s.initErr != nil {
		return s.service, s.initErr
	}
	root := s.storageRoot
	if root == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			s.initErr = fmt.Errorf("resolve user cache for project workspaces: %w", err)
			return nil, s.initErr
		}
		root = filepath.Join(cache, "Markitect", "workspaces")
	}
	service, err := projectworkspace.NewGitService(root, projectworkspace.Limits{
		MaxFiles: localWorkspaceMaxFiles, MaxFileBytes: localWorkspaceMaxFileBytes, MaxTotalBytes: localWorkspaceMaxTotalBytes,
	})
	if err != nil {
		s.initErr = fmt.Errorf("initialize local project workspace service: %w", err)
		return nil, s.initErr
	}
	s.service = service
	return service, nil
}

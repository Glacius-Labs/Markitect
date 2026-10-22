package agentd

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

const (
	// waitDelay bounds how long agentd waits for the provider's output
	// streams after the provider itself exited, for example when a grandchild
	// still holds them open. The process group is killed right afterwards.
	waitDelay = time.Second
	// maxInputBacklog bounds Orchestrator input waiting for a provider that is not
	// reading, so the Orchestrator connection is never blocked by provider input.
	maxInputBacklog = 64 << 20
)

// provider is the running provider CLI. Its standard input is fed by its own
// goroutine, so a provider that stops reading never holds back stop
// requests, tool replies or the end of the Orchestrator connection.
type provider struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	inMu     sync.Mutex
	inReady  *sync.Cond
	queue    [][]byte
	queued   int
	inClosed bool
	inBroken bool

	done     chan struct{}
	reported chan struct{}
	exit     frame.Exited
}

// startProvider starts cmd. The caller holds out's lock, so no provider
// output can reach the Orchestrator before the caller's started report.
func startProvider(cmd *exec.Cmd, out *frameWriter) (*provider, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = streamWriter{out: out, channel: frame.ProviderOut}
	cmd.Stderr = streamWriter{out: out, channel: frame.ProviderErr}
	cmd.WaitDelay = waitDelay
	ownProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &provider{cmd: cmd, stdin: stdin, done: make(chan struct{})}
	p.inReady = sync.NewCond(&p.inMu)
	go p.pumpInput()
	go func() {
		err := cmd.Wait()
		reapProcessGroup(cmd.Process.Pid)
		p.exit = exitOf(cmd, err)
		close(p.done)
		p.inMu.Lock()
		p.inReady.Broadcast()
		p.inMu.Unlock()
	}()
	return p, nil
}

func (p *provider) pid() int { return p.cmd.Process.Pid }

// write queues Orchestrator bytes for the provider's standard input. An empty
// payload ends the input once everything queued before it was written.
// Input after the provider exited is dropped; its exit report follows.
func (p *provider) write(b []byte) error {
	p.inMu.Lock()
	defer p.inMu.Unlock()
	if p.inClosed {
		if len(b) == 0 {
			return nil
		}
		return errors.New("provider input arrived after the Orchestrator ended it")
	}
	if len(b) == 0 {
		p.inClosed = true
		p.inReady.Signal()
		return nil
	}
	if p.inBroken || p.exited() {
		// The provider no longer reads its input; dropping it is an
		// observation, and its exit report follows when it ends.
		return nil
	}
	if p.queued+len(b) > maxInputBacklog {
		return fmt.Errorf("provider input backlog exceeds %d bytes", maxInputBacklog)
	}
	p.queue = append(p.queue, append([]byte(nil), b...))
	p.queued += len(b)
	p.inReady.Signal()
	return nil
}

// pumpInput writes queued input in order. A write error means the provider
// stopped reading or exited; that is an observation, never an Orchestrator error.
func (p *provider) pumpInput() {
	defer p.stdin.Close()
	for {
		p.inMu.Lock()
		for len(p.queue) == 0 && !p.inClosed && !p.exited() {
			p.inReady.Wait()
		}
		if len(p.queue) == 0 || p.exited() {
			p.queue, p.queued = nil, 0
			p.inMu.Unlock()
			return
		}
		b := p.queue[0]
		p.queue = p.queue[1:]
		p.queued -= len(b)
		p.inMu.Unlock()
		if _, err := p.stdin.Write(b); err != nil {
			p.inMu.Lock()
			p.queue, p.queued, p.inBroken = nil, 0, true
			p.inMu.Unlock()
			return
		}
	}
}

// stop asks the provider's process group to end and kills it after grace.
// Every call applies its own grace, so a shorter later grace still wins.
func (p *provider) stop(grace time.Duration) {
	if p.exited() {
		return
	}
	terminate(p.cmd)
	select {
	case <-p.done:
	case <-time.After(grace):
		kill(p.cmd)
	}
}

func (p *provider) exited() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

func (p *provider) wait() frame.Exited {
	<-p.done
	return p.exit
}

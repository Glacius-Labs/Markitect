package agentd

import (
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// waitDelay bounds how long agentd waits for the provider's output streams
// after the provider itself exited, for example when a grandchild still holds
// them open. The process group is killed right afterwards.
const waitDelay = time.Second

// provider is the running provider CLI.
type provider struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	inMu     sync.Mutex
	inClosed bool

	stopOnce sync.Once
	done     chan struct{}
	exit     frame.Exited
}

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
	go func() {
		err := cmd.Wait()
		reapProcessGroup(cmd.Process.Pid)
		p.exit = exitOf(cmd, err)
		close(p.done)
	}()
	return p, nil
}

func (p *provider) pid() int { return p.cmd.Process.Pid }

// write forwards Host bytes to the provider's standard input. An empty
// payload closes the input, which is how the Host signals the end of input.
func (p *provider) write(b []byte) error {
	p.inMu.Lock()
	defer p.inMu.Unlock()
	if p.inClosed {
		if len(b) == 0 {
			return nil
		}
		return errors.New("provider input is already closed")
	}
	if len(b) == 0 {
		p.inClosed = true
		return p.stdin.Close()
	}
	_, err := p.stdin.Write(b)
	if err != nil && p.exited() {
		// Writing to a provider that already ended is not a Host protocol
		// error; its exit report follows on the control channel.
		return nil
	}
	return err
}

// stop asks the provider's process group to end, then kills it after grace.
func (p *provider) stop(grace time.Duration) {
	p.stopOnce.Do(func() {
		if p.exited() {
			return
		}
		terminate(p.cmd)
		select {
		case <-p.done:
		case <-time.After(grace):
			kill(p.cmd)
		}
	})
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

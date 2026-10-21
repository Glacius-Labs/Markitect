package container

import (
	"context"
	"testing"
)

// An instance the engine no longer knows has ended: the removal confirms its
// absence instead of failing.
func TestRemovingAnInstanceThatIsGoneConfirmsItsEnd(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Code: 1, Err: "Error response from daemon: No such container: abc"}, engineStep{Code: 1, Err: "Error: No such container: abc"})
	inst := &Instance{e: e, id: "abc", spec: ContainerSpec{Name: "markitect-x-g1"}}
	if err := inst.Remove(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// The holder's user is reserved: no session runs as it.
func TestTheHolderUserIsNoSessionUser(t *testing.T) {
	spec := ContainerSpec{Name: "markitect-x", Image: "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", User: HoldUser,
		CPUs: 1, MemoryBytes: 1 << 20, Pids: 8, TmpBytes: 1 << 20, Labels: map[string]string{"markitect.store": "s"}}
	if _, err := spec.args(false, []string{"/usr/local/bin/markitect-agentd"}); err == nil {
		t.Fatal("a session ran as the holder's user")
	}
}

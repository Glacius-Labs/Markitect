package container

import (
	"context"
	"testing"
)

func TestEnsureNetworkCreatesAMissingBridge(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{}, engineStep{Out: "id"})
	if err := e.EnsureNetwork(context.Background(), "markitect-provider", map[string]string{"markitect.provider": "1"}); err != nil {
		t.Fatal(err)
	}
	c := calls()
	if len(c) != 2 || !containsArg(c[1], "create") || !containsArg(c[1], "bridge") || containsArg(c[1], "--internal") || !containsArg(c[1], "markitect-provider") {
		t.Fatalf("calls %q", c)
	}
}

func TestEnsureNetworkKeepsAnExistingBridge(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: "markitect-provider\n"}, engineStep{Out: "bridge false\n"})
	if err := e.EnsureNetwork(context.Background(), "markitect-provider", nil); err != nil || len(calls()) != 2 {
		t.Fatalf("err %v calls %q", err, calls())
	}
}

func TestEnsureNetworkRefusesAnInternalNetwork(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Out: "markitect-provider\n"}, engineStep{Out: "bridge true\n"})
	if err := e.EnsureNetwork(context.Background(), "markitect-provider", nil); err == nil {
		t.Fatal("an internal network was taken for the provider")
	}
}

func TestEnsureNetworkLooksAgainAfterARacingCreate(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{}, engineStep{Code: 1, Err: "network with name markitect-provider already exists"},
		engineStep{Out: "markitect-provider\n"}, engineStep{Out: "bridge false\n"})
	if err := e.EnsureNetwork(context.Background(), "markitect-provider", nil); err != nil || len(calls()) != 4 {
		t.Fatalf("err %v calls %q", err, calls())
	}
}

func TestEnsureNetworkRefusesTheEnginesOwnNetworks(t *testing.T) {
	e, _ := fakeCommandEngine(t)
	for _, name := range []string{"bridge", "host", "none", "a b"} {
		if err := e.EnsureNetwork(context.Background(), name, nil); err == nil {
			t.Fatalf("%q accepted", name)
		}
	}
}

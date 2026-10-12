package projectrun

import (
	"context"
	"testing"
)

func TestDeliverRequiresExplicitAuthorizationAndScope(t *testing.T) {
	root := t.TempDir()
	for _, request := range []DeliverRequest{
		{ExplorationID: "x", ScopeID: "y"},
		{ScopeID: "y", ExecuteAuthorized: true},
		{ExplorationID: "x", ExecuteAuthorized: true},
	} {
		if _, err := Deliver(context.Background(), Host{}, nil, root, request); err == nil {
			t.Fatalf("Deliver accepted incomplete or unauthorized request %+v", request)
		}
	}
}

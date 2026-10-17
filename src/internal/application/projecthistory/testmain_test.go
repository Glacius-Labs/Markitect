package projecthistory

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// TestMain shields git in these tests from the machine's Git configuration.
func TestMain(m *testing.M) { testkit.Main(m) }

func readState(root string) (projectbriefing.Store, string, error) {
	result, err := Read(root)
	return result.Accepted, result.PersistedDigest, err
}

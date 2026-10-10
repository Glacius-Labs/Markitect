package projectbriefing

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

// TestMain shields git in these tests from the machine's Git configuration.
func TestMain(m *testing.M) { testkit.Main(m) }

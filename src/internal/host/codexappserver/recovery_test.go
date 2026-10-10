package codexappserver

import "testing"

func TestValidateRecoveryProtocol(t *testing.T) {
	if err := ValidateRecoveryProtocol(RecoveryHandle{Protocol: protocolIdentity}); err != nil {
		t.Fatalf("owned protocol rejected: %v", err)
	}
	if err := ValidateRecoveryProtocol(RecoveryHandle{Protocol: protocolIdentity + "-foreign"}); err == nil {
		t.Fatal("foreign protocol accepted")
	}
}

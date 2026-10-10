package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

func TestValidateRecoveryProtocol(t *testing.T) {
	if err := ValidateRecoveryProtocol(RecoveryHandle{Protocol: protocolIdentity}); err != nil {
		t.Fatalf("owned protocol rejected: %v", err)
	}
	if err := ValidateRecoveryProtocol(RecoveryHandle{Protocol: protocolIdentity + "-foreign"}); err == nil {
		t.Fatal("foreign protocol accepted")
	}
}

func TestRecoverUsesCurrentFullVerifySubjectAliasBinding(t *testing.T) {
	a, cfg, _, _ := fixture(t, "recovery-completed", Options{})
	subjects := []string{`assessment:"quoted"`, "check:smoke"}
	inv := fullVerifyAliasTestInvocation(t, subjects)
	aliases, err := nativeFullVerifySubjectAliases(inv)
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]map[string]string, 0, len(subjects))
	for _, subject := range subjects {
		alias := ""
		for candidate, canonical := range aliases {
			if canonical == subject {
				alias = candidate
				break
			}
		}
		rows = append(rows, map[string]string{"subject": alias, "outcome": "incomplete", "detail": "preserved through recovery"})
	}
	report, _ := json.Marshal(map[string]any{"status": "incomplete", "summary": "recovered exact turn", "assessments": rows, "findings": []string{}, "counterexamples": []any{}})
	semantic, _ := json.Marshal(map[string]any{"outcome": "incomplete", "candidateFiles": []any{}, "verifierObservations": []any{}, "uncertainty": []string{"original turn incomplete"}, "reportJson": json.RawMessage(report)})
	t.Setenv("MARKITECT_P04_RECOVERY_FINAL", string(semantic))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := a.Fingerprint(cfg)
	if err != nil {
		t.Fatal(err)
	}
	handle := RecoveryHandle{
		Protocol:    protocolIdentity,
		Fingerprint: fingerprint,
		Invocation:  inv,
		Workspace: projectworkspace.Handle{
			ID:      "owned-workspace",
			CWD:     cwd,
			BaseSHA: inv.Request.SourceRevision,
		},
		ThreadID:       "thread-1",
		SessionID:      "thread-1",
		TurnID:         "turn-1",
		TurnDispatched: true,
	}
	result, err := a.Recover(context.Background(), cfg, handle)
	if err != nil {
		t.Fatalf("exact terminal invocation did not recover: %v", err)
	}
	if result.Receipt.RunID != inv.RunID || result.Response.RunID != inv.RunID || result.Receipt.Lifecycle.State != "completed" {
		t.Fatalf("recovery changed original invocation binding: receipt=%+v response=%+v", result.Receipt, result.Response)
	}
	var got struct {
		Status      string `json:"status"`
		Assessments []struct {
			Subject string `json:"subject"`
			Outcome string `json:"outcome"`
			Detail  string `json:"detail"`
		} `json:"assessments"`
	}
	if err := json.Unmarshal(result.Response.ReportJSON, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "incomplete" || len(got.Assessments) != len(subjects) {
		t.Fatalf("recovered semantic report changed: %+v", got)
	}
	for i, subject := range subjects {
		if got.Assessments[i].Subject != subject || got.Assessments[i].Outcome != "incomplete" || got.Assessments[i].Detail != "preserved through recovery" {
			t.Fatalf("recovered assessment %d lost its canonical subject/semantics: %+v", i, got.Assessments[i])
		}
	}
}

func TestRecoverUsesCurrentVerifierSubjectAliasBinding(t *testing.T) {
	a, cfg, _, _ := fixture(t, "recovery-completed", Options{})
	subjects := []string{`verification:"quoted"`, "verifier-subject-000000-000000"}
	inv := verifierSubjectAliasTestInvocation(t, subjects, true)
	aliases, err := nativeVerifierSubjectAliases(inv)
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]map[string]string, 0, len(subjects))
	for i, subject := range subjects {
		alias := ""
		for candidate, canonical := range aliases {
			if canonical == subject {
				alias = candidate
				break
			}
		}
		rows = append(rows, map[string]string{"subject": alias, "outcome": []string{"passed", "incomplete"}[i], "detail": "verifier semantics preserved"})
	}
	semantic, _ := json.Marshal(map[string]any{"outcome": "incomplete", "candidateFiles": []any{}, "verifierObservations": rows, "uncertainty": []string{}, "evidenceRefs": []string{}})
	t.Setenv("MARKITECT_P04_RECOVERY_FINAL", string(semantic))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := a.Fingerprint(cfg)
	if err != nil {
		t.Fatal(err)
	}
	handle := RecoveryHandle{
		Protocol:    protocolIdentity,
		Fingerprint: fingerprint,
		Invocation:  inv,
		Workspace:   projectworkspace.Handle{ID: "owned-workspace", CWD: cwd, BaseSHA: inv.Request.SourceRevision},
		ThreadID:    "thread-1", SessionID: "thread-1", TurnID: "turn-1", TurnDispatched: true,
	}
	result, err := a.Recover(context.Background(), cfg, handle)
	if err != nil {
		t.Fatalf("exact terminal verifier invocation did not recover: %v", err)
	}
	if result.Receipt.RunID != inv.RunID || result.Response.RunID != inv.RunID || result.Receipt.Lifecycle.State != "completed" {
		t.Fatalf("recovery changed original verifier invocation: receipt=%+v response=%+v", result.Receipt, result.Response)
	}
	if len(result.Response.VerifierObservations) != len(subjects) || len(result.Response.EvidenceRefs) != 0 {
		t.Fatalf("recovered verifier content changed: %+v", result.Response)
	}
	for i, subject := range subjects {
		got := result.Response.VerifierObservations[i]
		wantOutcome := []string{"passed", "incomplete"}[i]
		if got.Subject != subject || got.Outcome != wantOutcome || got.Detail != "verifier semantics preserved" {
			t.Fatalf("recovered verifier observation %d changed: %+v", i, got)
		}
	}
}

func TestRecoverRejectsPreAliasProtocolBeforeStartingProcess(t *testing.T) {
	a, cfg, _, _ := fixture(t, "version", Options{})
	inv := fullVerifyAliasTestInvocation(t, []string{"check:smoke"})
	fingerprint, err := a.Fingerprint(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	oldProtocols := []string{
		strings.TrimSuffix(protocolIdentity, "/native-verifier-subject-aliases-v1"),
		strings.TrimSuffix(strings.TrimSuffix(protocolIdentity, "/native-verifier-subject-aliases-v1"), "/native-full-verify-subject-aliases-v1"),
	}
	for _, oldProtocol := range oldProtocols {
		t.Run(oldProtocol[len(oldProtocol)-40:], func(t *testing.T) {
			handle := RecoveryHandle{
				Protocol:    oldProtocol,
				Fingerprint: fingerprint,
				Invocation:  inv,
				Workspace: projectworkspace.Handle{
					ID:      "owned-workspace",
					CWD:     cwd,
					BaseSHA: inv.Request.SourceRevision,
				},
				ThreadID:       "thread-1",
				SessionID:      "thread-1",
				TurnID:         "turn-1",
				TurnDispatched: true,
			}
			_, err = a.Recover(context.Background(), cfg, handle)
			if err == nil || !strings.Contains(err.Error(), "recovery handle does not match the owned invocation/configuration") || errors.Is(err, ErrUncertain) {
				t.Fatalf("prior alias-contract protocol did not fail closed before external process startup: %v", err)
			}
		})
	}
}

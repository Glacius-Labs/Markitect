package main

import (
	"strings"
	"testing"
)

const completeProtocolEnvelope = `{"apiVersion":"markitect.example.org/agent-execution/v1alpha1","runId":"run-1","nonce":"nonce-1","inputDigest":"sha256:input","request":{"role":"executor","sourceRevision":"commit-1","modelDigest":"sha256:model","modulePin":"module@1","projectionId":"projection-1","scopeIds":["requirement-1"],"policyIds":["policy-1"],"context":{"phase":"execute"},"artifacts":[]}}`

func TestDecodeInvocationAcceptsCompletePublicProtocolEnvelope(t *testing.T) {
	inv, err := decodeInvocation(strings.NewReader(completeProtocolEnvelope))
	if err != nil {
		t.Fatalf("decode complete public protocol envelope: %v", err)
	}
	if inv.Request.SourceRevision != "commit-1" || inv.Request.ModelDigest != "sha256:model" ||
		inv.Request.ModulePin != "module@1" || inv.Request.ProjectionID != "projection-1" ||
		len(inv.Request.ScopeIDs) != 1 || inv.Request.ScopeIDs[0] != "requirement-1" ||
		len(inv.Request.PolicyIDs) != 1 || inv.Request.PolicyIDs[0] != "policy-1" {
		t.Fatalf("decoded public request envelope lost protocol bindings: %+v", inv.Request)
	}
}

func TestDecodeInvocationRejectsUnknownFieldsAndTrailingInput(t *testing.T) {
	unknown := strings.Replace(completeProtocolEnvelope, `"role":"executor"`, `"role":"executor","unexpected":true`, 1)
	for name, input := range map[string]string{
		"unknown protocol field": unknown,
		"trailing JSON record":   completeProtocolEnvelope + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeInvocation(strings.NewReader(input)); err == nil {
				t.Fatal("decodeInvocation accepted malformed protocol input")
			}
		})
	}
}

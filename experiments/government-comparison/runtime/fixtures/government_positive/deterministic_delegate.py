"""Provider-free deterministic agentexec responder for native G5 mechanics."""
from __future__ import annotations

import argparse
import base64
import json
import sys

API = "markitect.example.org/agent-execution/v1alpha1"
SOURCE = '''package inventory

import (
	"errors"
	"math"
)

var ErrOverflow = errors.New("available stock would overflow")

// Release adds released stock to available inventory without wrapping.
func Release(available, amount int64) (int64, error) {
	if amount < 0 {
		return available, errors.New("release amount cannot be negative")
	}
	if amount > 0 && available > math.MaxInt64-amount {
		return available, ErrOverflow
	}
	return available + amount, nil
}
'''
TEST = '''package inventory

import (
	"errors"
	"math"
	"testing"
)

func TestReleaseRejectsOverflow(t *testing.T) {
	if _, err := Release(math.MaxInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Release(MaxInt64, 1) error = %v, want ErrOverflow", err)
	}
}

func TestReleaseAddsWithinRange(t *testing.T) {
	got, err := Release(8, 3)
	if err != nil || got != 11 {
		t.Fatalf("Release(8, 3) = (%d, %v), want (11, nil)", got, err)
	}
}
'''


def _artifacts(request):
    result = {}
    for item in request.get("artifacts", []):
        try:
            content = base64.b64decode(item["content"], validate=True)
        except (KeyError, ValueError, TypeError) as exc:
            raise ValueError("invalid agentexec artifact bytes") from exc
        result[item.get("path")] = content
    return result


def _candidate_check(request):
    files = _artifacts(request)
    source = files.get("inventory/reservation.go", b"").decode("utf-8", "strict")
    test = files.get("inventory/reservation_test.go", b"").decode("utf-8", "strict")
    checks = [
        ("implementation is supplied", bool(source)),
        ("MaxInt64 addition boundary is guarded", "available > math.MaxInt64-amount" in source),
        ("negative release is rejected", "amount < 0" in source),
        ("overflow test calls Release(MaxInt64, 1)", "Release(math.MaxInt64, 1)" in test),
        ("overflow test requires ErrOverflow", "errors.Is(err, ErrOverflow)" in test),
        ("ordinary addition is preserved", "Release(8, 3)" in test),
    ]
    detail = "; ".join(("passed: " if ok else "failed: ") + label for label, ok in checks)
    return all(ok for _, ok in checks), detail


def response(invocation: dict, slot_phase: str) -> dict:
    if not isinstance(invocation, dict) or set(invocation) != {"apiVersion", "runId", "nonce", "inputDigest", "request"}:
        raise ValueError("agentexec Invocation envelope malformed")
    request = invocation["request"]
    if invocation["apiVersion"] != API or not isinstance(request, dict):
        raise ValueError("agentexec Invocation API malformed")
    role = request.get("role")
    if role not in {"executor", "verifier"}:
        raise ValueError("unsupported agentexec role")
    projection = request.get("projectionId", "").split("/")
    if len(projection) != 3 or projection[0] != "government" or projection[1] != slot_phase or not projection[2]:
        raise ValueError("Invocation projection does not match the configured native slot")
    refs = sorted(set(request.get("scopeIds", [])) |
                  set(request.get("policyIds", [])) |
                  {item.get("path") for item in request.get("artifacts", []) if isinstance(item, dict)})
    out = {
        "apiVersion": API,
        "runId": invocation["runId"],
        "nonce": invocation["nonce"],
        "role": role,
        "inputDigest": invocation["inputDigest"],
        "outcome": "incomplete",
        "candidateFiles": [],
        "candidateJson": None,
        "evidenceRefs": refs,
        "verifierObservations": [],
        "uncertainty": [],
    }
    if slot_phase == "execute" and role == "executor":
        out["outcome"] = "proposed"
        out["candidateFiles"] = [
            {"path": "inventory/reservation.go", "mode": "0644", "content": SOURCE},
            {"path": "inventory/reservation_test.go", "mode": "0644", "content": TEST},
        ]
        return out
    if slot_phase == "review" and role == "verifier":
        passed, detail = _candidate_check(request)
        result = "passed" if passed else "failed"
        out["outcome"] = result
        out["verifierObservations"] = [
            {"subject": subject, "outcome": result, "detail": detail}
            for subject in sorted(set(request.get("scopeIds", [])) | set(request.get("policyIds", [])))
        ]
        if not out["verifierObservations"]:
            out["outcome"] = "incomplete"
            out["uncertainty"] = ["No released review subjects were supplied."]
        return out
    if slot_phase == "vote" and role == "verifier":
        context = request.get("context")
        if isinstance(context, str):
            context = json.loads(context)
        candidate = context.get("candidate", {}) if isinstance(context, dict) else {}
        evidence = context.get("evidence", {}) if isinstance(context, dict) else {}
        candidate_id = candidate.get("id") if isinstance(candidate, dict) else None
        evidence_id = evidence.get("id") if isinstance(evidence, dict) else None
        round_number = evidence.get("round") if isinstance(evidence, dict) else None
        if not (isinstance(candidate_id, str) and candidate_id and isinstance(evidence_id, str) and evidence_id and
                type(round_number) is int and round_number > 0):
            out["uncertainty"] = ["Vote context lacks candidate, evidence, or positive round identity."]
            return out
        passed, check_detail = _candidate_check(request)
        if not passed:
            out["outcome"] = "failed"
            out["verifierObservations"] = [{
                "subject": "government-vote", "outcome": "failed",
                "detail": json.dumps({
                    "outcome": "objection",
                    "reason": check_detail,
                    "materialCandidateId": candidate_id,
                    "evidenceId": evidence_id,
                    "round": round_number,
                }, sort_keys=True, separators=(",", ":")),
            }]
            return out
        out["outcome"] = "passed"
        out["verifierObservations"] = [{
            "subject": "government-vote", "outcome": "passed",
            "detail": json.dumps({
                "outcome": "assent",
                "reason": "Deterministic fixture checked the bound overflow implementation and focused test; this is process-mechanics evidence only.",
                "materialCandidateId": candidate_id,
                "evidenceId": evidence_id,
                "round": round_number,
            }, sort_keys=True, separators=(",", ":")),
        }]
        return out
    raise ValueError("configured role/phase does not match the agentexec Invocation")


def main(argv=None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--phase", required=True, choices=("execute", "review", "vote"))
    args = parser.parse_args(argv)
    invocation = json.load(sys.stdin)
    result = response(invocation, args.phase)
    sys.stdout.write(json.dumps(result, sort_keys=True, separators=(",", ":")) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

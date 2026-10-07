#!/usr/bin/env python3
"""Deterministic protocol test double for the Classic Commerce example.

This program is not a model, semantic verifier, or provider adapter. It returns
fixed candidate bytes and synthetic protocol-coverage observations only.
"""
from __future__ import annotations

import argparse
import json
import sys
from typing import Any

API_VERSION = "markitect.example.org/agent-execution/v1alpha1"
DOTNET_PROJECTION = '["markitect.foundation/v1","Projection","commerce","application-dotnet"]'


def candidate_files(mode: str) -> list[dict[str, str]]:
    handler_body = (
        "using System;\n\n"
        "namespace Commerce.Application;\n\n"
        "public sealed class CreateOrderHandler\n"
        "{\n"
        "    public decimal Handle(int quantity, decimal unitPrice)\n"
        "    {\n"
        "        if (quantity <= 0) throw new ArgumentOutOfRangeException(nameof(quantity));\n"
        "        if (unitPrice < 0m) throw new ArgumentOutOfRangeException(nameof(unitPrice));\n"
        + ("        return quantity + unitPrice;\n" if mode == "bad-business" else "        return quantity * unitPrice;\n")
        + "    }\n"
        "}\n"
    )
    return [
        {
            "path": "src/Commerce/Commerce.csproj",
            "mode": "0644",
            "content": (
                '<Project Sdk="Microsoft.NET.Sdk">\n'
                "  <PropertyGroup>\n"
                "    <TargetFramework>net8.0</TargetFramework>\n"
                "    <ImplicitUsings>enable</ImplicitUsings>\n"
                "    <Nullable>enable</Nullable>\n"
                "  </PropertyGroup>\n"
                "</Project>\n"
            ),
        },
        {"path": "src/Commerce/CreateOrderHandler.cs", "mode": "0644", "content": handler_body},
        {
            "path": "src/Commerce/EffectAxis.cs",
            "mode": "0644",
            "content": "namespace Commerce.Domain;\n\npublic sealed record EffectAxis(string Boundary = \"application\");\n",
        },
    ]


def required_subjects(context: Any) -> list[str]:
    if not isinstance(context, dict):
        raise ValueError("invocation request context must be a JSON object")
    subjects = context.get("requiredObservationSubjects")
    if not isinstance(subjects, list) or not subjects:
        raise ValueError("verifier invocation has no required observation subjects")
    result: list[str] = []
    for item in subjects:
        if not isinstance(item, dict) or not isinstance(item.get("subject"), str) or not item["subject"]:
            raise ValueError("required observation subject has an unsupported shape")
        result.append(item["subject"])
    if len(set(result)) != len(result):
        raise ValueError("required observation subjects are not unique")
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("positive", "bad-business"), required=True)
    args = parser.parse_args()
    try:
        invocation = json.load(sys.stdin)
        request = invocation["request"]
        role = request["role"]
        if invocation.get("apiVersion") != API_VERSION or role not in ("executor", "verifier"):
            raise ValueError("unsupported invocation version or role")
        response: dict[str, Any] = {
            "apiVersion": invocation["apiVersion"],
            "runId": invocation["runId"],
            "nonce": invocation["nonce"],
            "role": role,
            "inputDigest": invocation["inputDigest"],
            "outcome": "proposed" if role == "executor" else "passed",
            "candidateFiles": candidate_files(args.mode) if role == "executor" and request.get("projectionId") == DOTNET_PROJECTION else [],
            "evidenceRefs": [],
            "verifierObservations": [],
            "uncertainty": [],
        }
        if role == "executor" and request.get("projectionId") != DOTNET_PROJECTION:
            raise ValueError("this fixed Executor supports only the declared .NET Projection")
        if role == "verifier":
            evidence = set(request.get("scopeIds", [])) | set(request.get("policyIds", []))
            artifacts = request.get("artifacts", [])
            if not isinstance(artifacts, list):
                raise ValueError("verifier artifact list has an unsupported shape")
            evidence.update(item["path"] for item in artifacts)
            response["evidenceRefs"] = sorted(evidence)
            context = request.get("context")
            response["verifierObservations"] = [
                {
                    "subject": subject,
                    "outcome": "passed",
                    "detail": "Protocol test double echoed the required observation identity; no semantic judgment was made.",
                }
                for subject in required_subjects(context)
            ]
        json.dump(response, sys.stdout, separators=(",", ":"), ensure_ascii=False)
        sys.stdout.write("\n")
        return 0
    except Exception as exc:
        print(f"protocol test double refused invocation: {type(exc).__name__}: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
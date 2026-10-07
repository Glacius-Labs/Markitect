"""Actual CLI pins and prospective common actor profile. Provider gate stays closed."""
import hashlib
import json
from pathlib import Path

EXPECTED_SHA256 = "280cb1c4e3375d94dbdcba1a191f4f6adbf73c293be1e4f16c74b006662b9c54"
EXPECTED_VERSION = "codex-cli 0.130.0"
MODEL = "gpt-6.1-sol"
REASONING = "high"
CONFIG = {
    "model_reasoning_effort": REASONING,
    "project_doc_max_bytes": 0,
    "features.memories": False,
    "features.multi_agent": False,
    "features.plugins": False,
    "features.hooks": False,
    "features.goals": False,
    "model_providers.openai.request_max_retries": 0,
    "model_providers.openai.stream_max_retries": 0,
    "web_search": "disabled",
}
GAPS = [
    "Account support and resolved identity for gpt-6.1-sol/high have not been established by a provider receipt",
    "No documented pre-dispatch provider-call cap found in installed exec interface or generated app-server protocol; six-call probe limit cannot yet be enforced",
    "exec turn.completed is an agent-turn completion event; provider-turn count inside it remains unknown",
    "Provider token usage arrives retrospectively; no hard aggregate token ceiling is established",
    "Effective instruction/memory/config/tool and filesystem access boundaries have not been probed by an actual Actor",
]


def inspect(executable):
    path = Path(executable).resolve(strict=True)
    digest = hashlib.sha256(path.read_bytes()).hexdigest()
    if digest != EXPECTED_SHA256:
        raise ValueError("Codex binary differs from readiness pin")
    return {"path": str(path), "sha256": digest, "version": EXPECTED_VERSION,
            "requestedModel": MODEL, "requestedReasoning": REASONING, "actualModel": None,
            "modelSupport": "unverified", "providerProbeSessions": 0,
            "prospectiveConfig": CONFIG, "configSha256": hashlib.sha256(
                json.dumps(CONFIG, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
            "gaps": GAPS}


def prospective_argv(executable, actor_root):
    """Describe equal configuration across arms; never launch this from readiness."""
    pin = inspect(executable)
    root = Path(actor_root).resolve(strict=True)
    argv = [pin["path"], "exec", "--ignore-user-config", "--ignore-rules", "--model", MODEL,
            "--sandbox", "workspace-write", "--ephemeral", "--json", "--skip-git-repo-check"]
    for key, value in sorted(CONFIG.items()):
        argv += ["--config", key + "=" + json.dumps(value)]
    return [*argv, "--cd", str(root), "-"]


def require_provider_ready():
    raise ValueError("Provider launch gate closed: " + "; ".join(GAPS))

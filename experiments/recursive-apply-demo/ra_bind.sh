#!/usr/bin/env bash
# Bind one demonstration attempt to a finished build: Runtime profiles (one Claude
# model for every Agent role, Sonnet unless MARKITECT_AGENT_MODEL names
# another), check profiles, the private runtime token file and the frozen
# scripts. MARKITECT_STORE labels the attempt's containers (default
# ra-library), so parallel sessions can clean up only their own. Never prints a
# credential.
# Usage: ra_bind.sh BUILD_DIR ATTEMPT_DIR
set -euo pipefail
BUILD="$1"
ATTEMPT="$2"
DEMO="$(cd "$(dirname "$0")" && pwd)"
MODEL="${MARKITECT_AGENT_MODEL:-claude-sonnet-5-5}"
STORE="${MARKITECT_STORE:-ra-library}"
test ! -e "$ATTEMPT"
mkdir -p "$ATTEMPT/host" "$ATTEMPT/secret" "$ATTEMPT/frozen"
chmod 700 "$ATTEMPT/secret"
SHA=$(grep '^sha=' "$BUILD/build.txt" | cut -d= -f2)
TAG="${SHA:0:12}"
GOIMG=$(docker image inspect --format '{{.Id}}' "markitect-runtime-go:ra-$TAG")
RUNTIME=$(docker image inspect --format '{{.Id}}' "markitect-runtime:ra-$TAG")
AGENTD=$(grep '^agentd=' "$BUILD/runtime.env" | cut -d= -f2)
( umask 077; set -a; . "${MARKITECT_RUNTIME_ENV:-$HOME/.markitect/runtime.env}"; set +a; printf '%s' "$CLAUDE_CODE_OAUTH_TOKEN" > "$ATTEMPT/secret/claude-token" )
test -s "$ATTEMPT/secret/claude-token"
cat > "$ATTEMPT/host/host.yaml" <<YAML
apiVersion: markitect.runtime/v1
agentProfiles:
  library-agent:
    claudeCode:
      profileVersion: 1
      engineExecutable: /usr/bin/docker
      imageID: $GOIMG
      agentdVersion: $AGENTD
      instructions: []
      credentialRef: ra-claude-token
      maxWireBytes: 1048576
      session:
        version: 1
        model: $MODEL
        effort: high
        systemPrompt: "You are a Markitect Manager agent in a team that applies a repository's canonical model. Follow the turn input and report only through the offered Orchestrator tools. Never print or copy credentials. Use only the Go standard library for the realization."
        builtinTools: [Read, Write, Edit, Glob, Grep, Bash]
        store: $STORE
        cpus: 2
        memoryBytes: 4294967296
        pids: 1024
        tmpBytes: 1073741824
        network: ""
checkProfiles:
  library-go:
    container:
      profileVersion: 1
      engineExecutable: /usr/bin/docker
      imageID: $GOIMG
      executable: /usr/local/go/bin/go
      argumentPrefix: []
      env: ["GOTOOLCHAIN=local", "GOCACHE=/home/agent/.cache/go-build", "GOPATH=/home/agent/go", "GOTMPDIR=/home/agent", "GOFLAGS=-buildvcs=false"]
      limits:
        milliCPUs: 2000
        memoryBytes: 4294967296
        processes: 1024
      maxOutputBytes: 1048576
YAML
cp "$DEMO"/ra_project.sh "$DEMO"/ra_demo.sh "$DEMO"/ra_inner.sh "$DEMO"/ra_cleanup.sh "$DEMO"/ra_assess.sh "$DEMO"/ra_extract_project.sh "$DEMO"/ra_extract.sh "$DEMO"/rx_rebuild_project.sh "$DEMO"/extract-execution.yaml "$DEMO"/ra_runconfig.py "$DEMO"/library-execution.yaml "$ATTEMPT/frozen/"
{
  echo "product_sha=$SHA"
  echo "cli_sha256=$(sha256sum "$BUILD/markitect" | cut -d' ' -f1)"
  echo "runtime_image=$RUNTIME"
  echo "agent_image=$GOIMG"
  echo "agentd=$AGENTD"
  echo "model=$MODEL effort=high provider=claude-code (all Execute and Verify turns)"
  echo "host_profile_sha256=$(sha256sum "$ATTEMPT/host/host.yaml" | cut -d' ' -f1)"
  echo "credential=dedicated Claude runtime token from MARKITECT_RUNTIME_ENV (not printed)"
  (cd "$ATTEMPT/frozen" && sha256sum *)
} > "$ATTEMPT/binding.txt"
cp "$BUILD/markitect" "$ATTEMPT/markitect"
cat "$ATTEMPT/binding.txt"

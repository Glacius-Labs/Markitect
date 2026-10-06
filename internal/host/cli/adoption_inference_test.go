package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const validBrownfieldRuntime = `{"apiVersion":"markitect.brownfield/inference-runtime/v1alpha1","agent":{"command":"provider","args":["--stdio"],"model":"model-id","modelOptions":{"temperature":0},"providerVersion":"provider/1","timeoutSeconds":30,"maxStdoutBytes":4096,"maxStderrBytes":1024,"runtimeFiles":[{"path":"provider","mode":"0755","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]},"tempParent":"C:/private/temp","privateLogs":"C:/private/logs"}`

func TestDecodeBrownfieldInferenceRuntimeIsClosedAndBounded(t *testing.T) {
	config, err := decodeBrownfieldInferenceRuntime([]byte(validBrownfieldRuntime))
	if err != nil {
		t.Fatal(err)
	}
	if config.Agent.Command != "provider" || len(config.Agent.Args) != 1 || config.Agent.Model != "model-id" || config.Agent.ProviderVersion != "provider/1" || config.Agent.TimeoutSeconds != 30 || config.TempParent != "C:/private/temp" || config.PrivateLogs != "C:/private/logs" {
		t.Fatalf("runtime fields were not preserved literally: %+v", config)
	}

	for name, bad := range map[string]string{
		"unknown top-level key":    strings.Replace(validBrownfieldRuntime, `"privateLogs"`, `"extra":true,"privateLogs"`, 1),
		"unknown agent key":        strings.Replace(validBrownfieldRuntime, `"command"`, `"extra":true,"command"`, 1),
		"unknown runtime file key": strings.Replace(validBrownfieldRuntime, `"mode"`, `"extra":true,"mode"`, 1),
		"trailing JSON value":      validBrownfieldRuntime + `{}`,
		"wrong version":            strings.Replace(validBrownfieldRuntime, "markitect.brownfield/inference-runtime/v1alpha1", "other/v1", 1),
		"unbounded timeout":        strings.Replace(validBrownfieldRuntime, `"timeoutSeconds":30`, `"timeoutSeconds":601`, 1),
		"missing model options":    strings.Replace(validBrownfieldRuntime, `"modelOptions":{"temperature":0},`, "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeBrownfieldInferenceRuntime([]byte(bad)); err == nil {
				t.Fatal("runtime configuration was accepted")
			}
		})
	}
}

func TestCopyMeInferRequiresExplicitInputsAndRejectsDecisions(t *testing.T) {
	for _, args := range [][]string{
		{"copy-me", "--action", "infer"},
		{"copy-me", "--action", "infer", "--workspace", "handoff", "--queue", "queue", "--runtime", "runtime", "--decision="},
		{"copy-me", "--action", "validate", "--workspace", "handoff", "--queue", "queue"},
		{"copy-me", "--workspace", "handoff", "--queue", "queue", "--runtime", "runtime"},
	} {
		var out, errout bytes.Buffer
		if code := Run(args, &out, &errout); code != 2 || errout.Len() == 0 {
			t.Fatalf("must refuse %v: exit %d, stdout=%q stderr=%q", args, code, out.String(), errout.String())
		}
	}
}

func TestCopyMeInferEmitsFlatJSONFailureAndDefaultValidationRemainsYAML(t *testing.T) {
	var inferred, inferErr bytes.Buffer
	if code := Run([]string{"copy-me", "--action", "infer", "--workspace", "missing-workspace", "--queue", "missing-queue", "--runtime", "missing-runtime"}, &inferred, &inferErr); code != 1 {
		t.Fatalf("infer failure exit = %d, stderr=%q", code, inferErr.String())
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(inferred.Bytes(), &value); err != nil {
		t.Fatalf("infer output must be JSON: %v (%q)", err, inferred.String())
	}
	for _, key := range []string{"status", "adopted", "error"} {
		if _, ok := value[key]; !ok {
			t.Fatalf("flat infer result omitted %q: %s", key, inferred.String())
		}
	}
	if _, nested := value["result"]; nested {
		t.Fatalf("infer output unexpectedly nested its result: %s", inferred.String())
	}

	var validated, validationErr bytes.Buffer
	if code := Run([]string{"copy-me", "--workspace", "missing-workspace", "--queue", "missing-queue"}, &validated, &validationErr); code != 2 || !strings.Contains(validationErr.String(), "workspace") {
		t.Fatalf("default validation behavior changed: exit=%d stdout=%q stderr=%q", code, validated.String(), validationErr.String())
	}
}

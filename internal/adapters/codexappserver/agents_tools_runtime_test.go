package codexappserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// TestRuntimeNativeAgentTools uses the installed CLI, agentd MCP and an HTTP
// recorder inside one network-none container. The recorder returns a synthetic
// 400 before inference. No credential, external provider or generated work is used.
func TestRuntimeNativeAgentTools(t *testing.T) {
	enginePath, image := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_RUNTIME_IMAGE")
	if enginePath == "" || image == "" {
		t.Skip("set MARKITECT_TEST_ENGINE and MARKITECT_TEST_RUNTIME_IMAGE for the offline CLI probe")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	image, err = engine.ImageID(ctx, image)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("offline tool inventory image=%s", image)
	for _, test := range []struct {
		name        string
		overrides   []string
		wantNative  bool
		wantAgents  bool
		wantV2      bool
		omitV2Guard bool
	}{
		{"feature-only-is-insufficient", []string{"-c", "agents.enabled=true"}, true, true, false, false},
		{"explicit-v2-overrides-agent-switch", nil, true, false, true, true},
		{"adapter-overrides-inherited-agent-settings", nil, false, false, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "codex-tools-" + smokeToken(t)
			home := name + "-home"
			labels := map[string]string{"markitect.store": name}
			if _, err := engine.CreateVolumeExclusive(ctx, home, labels); err != nil {
				t.Fatal(err)
			}
			defer func() {
				cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
				defer done()
				if err := engine.RemoveVolume(cleanup, home); err != nil {
					t.Error(err)
				}
			}()
			peer, err := engine.Start(ctx, container.ContainerSpec{Name: name, Image: image, Labels: labels,
				User: "1000:1000", Workdir: "/work", Network: "none", CPUs: 1, MemoryBytes: 2 << 30,
				Pids: 192, TmpBytes: 256 << 20, Mounts: []container.Mount{{Volume: home, Target: "/home/agent"}}})
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := peer.Close(); err != nil {
					t.Error(err)
				}
			}()
			command := providerArgv()
			if test.omitV2Guard {
				for i := 0; i+1 < len(command); i++ {
					if command[i] == "--disable" && command[i+1] == "multi_agent_v2" {
						command = append(command[:i], command[i+2:]...)
						break
					}
				}
			}
			argv, err := json.Marshal(append(command, test.overrides...))
			if err != nil {
				t.Fatal(err)
			}
			_, err = peer.StartProvider(ctx, frame.Start{Argv: []string{"/usr/bin/python3", "-c", offlineToolsProbe, string(argv)},
				Dir: "/work", Tools: toolSpecs()})
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Version  string   `json:"version"`
				Model    string   `json:"model"`
				Agents   bool     `json:"agents"`
				V1       bool     `json:"v1"`
				V2       bool     `json:"v2"`
				Names    []string `json:"names"`
				Shell    bool     `json:"shell"`
				Files    bool     `json:"files"`
				Host     bool     `json:"host"`
				Catalog  bool     `json:"catalog"`
				Deferred bool     `json:"deferred"`
			}
			if err := json.NewDecoder(io.LimitReader(peer.Stdout(), 1<<20)).Decode(&got); err != nil {
				t.Fatalf("offline capture failed: %v; diagnostics=%s", err, peer.Diagnostics())
			}
			select {
			case exit := <-peer.Exited():
				if exit.Code != 0 || exit.Signal != "" {
					t.Fatalf("offline probe exit=%+v", exit)
				}
			case <-ctx.Done():
				t.Fatal("offline probe did not exit", ctx.Err())
			}
			native := slices.Contains(got.Names, "spawn_agent")
			if got.Version != "codex-cli 0.162.0" || got.Model != "gpt-6-luna" ||
				got.Agents != test.wantAgents || got.V1 || got.V2 != test.wantV2 || native != test.wantNative ||
				!got.Shell || !got.Files || !got.Host || !got.Catalog {
				t.Fatalf("unexpected effective config/tool declaration: %+v", got)
			}
			if !test.wantNative {
				for _, name := range []string{"collaboration", "send_input", "resume_agent", "close_agent", "send_message", "followup_task", "interrupt_agent", "list_agents", "wait_agent"} {
					if slices.Contains(got.Names, name) {
						t.Fatalf("native delegation tool remains: %s", name)
					}
				}
			}
			t.Logf("version=%s model=%s agents.enabled=%t multi_agent=%t multi_agent_v2=%t tools=%v shell=%t files=%t Markitect_Run_tools_registered=%t ALL_TOOLS_catalog=%t MCP_deferred=%t", got.Version, got.Model, got.Agents, got.V1, got.V2, got.Names, got.Shell, got.Files, got.Host, got.Catalog, got.Deferred)
		})
	}
}

const offlineToolsProbe = `
import http.server, json, os, queue, subprocess, sys, threading, time
captured = queue.Queue()
class Capture(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        captured.put((self.path, body))
        self.send_response(400)
        self.end_headers()
        self.wfile.write(b'{"error":{"message":"offline inventory captured; no inference"}}')
    def log_message(self, *args):
        pass
os.makedirs('/home/agent/.codex', exist_ok=True)
with open('/home/agent/.codex/config.toml', 'w') as f:
    f.write('[agents]\nenabled=true\n[features]\nmulti_agent=true\nmulti_agent_v2=true\n')
server = http.server.HTTPServer(('127.0.0.1', 0), Capture)
threading.Thread(target=server.serve_forever, daemon=True).start()
args = json.loads(sys.argv[1]) + ['--disable', 'enable_request_compression', '-c', 'model_provider="offline"', '-c', 'model_providers.offline={name="offline",base_url="http://127.0.0.1:%d/v1",wire_api="responses",requires_openai_auth=false,supports_websockets=false}' % server.server_port]
proc = subprocess.Popen(args, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=sys.stderr, text=True)
lines = queue.Queue()
def pump():
    for line in proc.stdout:
        lines.put(json.loads(line))
threading.Thread(target=pump, daemon=True).start()
def request(i, method, params):
    proc.stdin.write(json.dumps({'id':i,'method':method,'params':params})+'\n')
    proc.stdin.flush()
    deadline = time.monotonic()+15
    while time.monotonic() < deadline:
        result = lines.get(timeout=max(0.1, deadline-time.monotonic()))
        if result.get('id') == i:
            if 'error' in result:
                raise RuntimeError(json.dumps(result['error']))
            return result['result']
    raise RuntimeError('response deadline')
try:
    request(1, 'initialize', {'clientInfo':{'name':'markitect_offline_probe','version':'1'},'capabilities':{'experimentalApi':True}})
    proc.stdin.write('{"method":"initialized"}\n')
    proc.stdin.flush()
    config = request(2, 'config/read', {'includeLayers':False})['config']
    mcp = request(5, 'mcpServerStatus/list', {})
    thread = request(3, 'thread/start', {'model':'gpt-6-luna','cwd':'/work','approvalPolicy':'never','sandbox':'danger-full-access','baseInstructions':'Offline inventory probe.','developerInstructions':''})
    request(4, 'turn/start', {'threadId':thread['thread']['id'],'input':[{'type':'text','text':'Inventory probe.'}],'effort':'high'})
    path, body = captured.get(timeout=20)
    assert path == '/v1/responses'
    def names(tools):
        result = []
        for tool in tools:
            result.append(tool.get('name', tool.get('function', {}).get('name', '')))
            result += names(tool.get('tools', []))
        return result
    tools = list(body.get('tools', []))
    for item in body.get('input', []):
        if item.get('type') == 'additional_tools':
            tools += item.get('tools', [])
    declaration = json.dumps(tools)
    host = any(s['name'] == 'markitect' and s.get('toolsError') is None and all(name in s.get('tools', {}) for name in ['delegate','await','review','ask','reply','read_notes','append_note','judge','check']) for s in mcp['data'])
    print(json.dumps({'version':subprocess.check_output(['/usr/local/bin/codex','--version'],text=True).strip(), 'model':body['model'], 'agents':config['agents']['enabled'], 'v1':config['features']['multi_agent'], 'v2':config['features']['multi_agent_v2'], 'names':names(tools), 'shell':'exec_command' in declaration, 'files':'apply_patch' in declaration, 'host':host, 'catalog':'ALL_TOOLS' in declaration, 'deferred':'mcp__markitect__delegate' not in declaration}), flush=True)
finally:
    proc.kill()
    proc.wait(timeout=5)
    server.shutdown()
`

func smokeToken(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b[:])
}

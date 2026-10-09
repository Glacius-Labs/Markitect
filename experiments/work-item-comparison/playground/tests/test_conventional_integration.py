"""Whole wrapper paths with scripted local executables, never native agents."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import unittest

import test_conventional_service as service_fixtures
from conventional.service import Service
from conventional.mcp import _Protocol


CLI_FIXTURE = r'''
import json, sys
args = sys.argv[1:]
prompt = sys.stdin.read()
session = args[args.index('resume') + 1] if 'resume' in args else 'offline-session'
print(json.dumps({'type':'thread.started','thread_id':session}), flush=True)
print(json.dumps({'type':'fixture.prompt','text':prompt}), flush=True)
print(json.dumps({'type':'turn.completed','usage':{'input_tokens':2}}), flush=True)
'''

APP_SERVER_FIXTURE = r'''
import json, sys
def send(value): print(json.dumps(value), flush=True)
for line in sys.stdin:
    message = json.loads(line)
    method = message.get('method')
    if method == 'initialize': send({'id':message['id'],'result':{}})
    elif method in ('thread/start','thread/resume'):
        session = message['params'].get('threadId','offline-session')
        send({'id':message['id'],'result':{'thread':{'id':session}}})
    elif method == 'turn/start':
        send({'method':'fixture.prompt','params':{'text':message['params']['input'][0]['text']}})
        # Real nested terminal can arrive before the turn/start response.
        send({'method':'turn/completed','params':{'threadId':session,'turn':{'id':'offline-turn','status':'completed'}}})
        send({'id':message['id'],'result':{'turn':{'id':'offline-turn'}}})
        break
'''


class IntegrationTests(unittest.TestCase):
    def setUp(self):
        self.binding = service_fixtures.ServiceTests()
        self.binding.setUp()
        self.addCleanup(self.binding.doCleanups)

    def native_config(self, backend, body):
        fixture = self.binding.repo.parent / "scripted_native.py"
        fixture.write_bytes(body.encode("utf-8"))
        self.binding.config.update(backend=backend, command=[str(Path(sys.executable).resolve()), str(fixture)])
        self.binding.config["filePins"][str(fixture)] = hashlib.sha256(fixture.read_bytes()).hexdigest()
        self.binding.persist()

    def protocol(self, service):
        protocol = _Protocol(service)
        result = protocol.handle({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
            "protocolVersion": "2025-11-25", "capabilities": {}, "clientInfo": {"name": "offline", "version": "1"}}})
        self.assertIn("result", result)
        protocol.handle({"jsonrpc": "2.0", "method": "notifications/initialized"})
        return protocol

    def test_mcp_service_cli_backend_start_and_continuation(self):
        self.exercise_mcp("codex-cli", CLI_FIXTURE)

    def test_old_mcp_version_uses_text_results_and_no_new_annotations(self):
        service = self.binding.service()
        protocol = _Protocol(service)
        protocol.handle({"jsonrpc":"2.0","id":1,"method":"initialize","params":{
            "protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"offline","version":"1"}}})
        protocol.handle({"jsonrpc":"2.0","method":"notifications/initialized"})
        tools = protocol.handle({"jsonrpc":"2.0","id":2,"method":"tools/list"})["result"]["tools"]
        self.assertTrue(all("annotations" not in tool for tool in tools))
        result = protocol.handle({"jsonrpc":"2.0","id":3,"method":"tools/call","params":{
            "name":"conventional_start","arguments":{"prompt":"offline"}}})["result"]
        self.assertNotIn("structuredContent",result)
        handle = json.loads(result["content"][0]["text"])
        self.assertEqual("completed",service.wait(handle["runId"])["state"])

    def test_mcp_service_app_server_backend_start_and_continuation(self):
        self.exercise_mcp("codex-app-server", APP_SERVER_FIXTURE)

    def exercise_mcp(self, backend, body):
        self.native_config(backend, body)
        service = Service(self.binding.path)
        self.addCleanup(service.close)
        protocol = self.protocol(service)
        response = protocol.handle({"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {
            "name": "conventional_start", "arguments": {"prompt": "ordinary backlog request"}}})
        accepted = response["result"]["structuredContent"]
        first = service.wait(accepted["runId"])
        self.assertEqual("completed", first["state"], first)
        response = protocol.handle({"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": {
            "name": "conventional_resume", "arguments": {"run_id": first["runId"], "prompt": "next public wave"}}})
        second = service.wait(response["result"]["structuredContent"]["runId"])
        self.assertEqual("completed", second["state"], second)
        self.assertEqual("offline-session", second["nativeSessionId"])
        self.assertEqual("NOT RUN", second["taskAssessment"])
        self.assertEqual("direct-child-only", second["ownedProcessScope"])
        folder = self.binding.audit / "conventional-execution" / second["runId"]
        events = [json.loads(line) for line in (folder / "events.jsonl").read_text(encoding="utf-8").splitlines()]
        self.assertEqual(list(range(1, len(events)+1)), [event["sequence"] for event in events])
        self.assertTrue(any("next public wave" in event["event"].get("raw", "") for event in events))
        self.assertTrue(any("rawBase64" in event["event"] for event in events))

    def test_cli_facade_waits_for_scripted_native_terminal(self):
        self.native_config("codex-cli", CLI_FIXTURE)
        prompt = self.binding.repo.parent / "prompt.txt"
        prompt.write_bytes("nächste öffentliche Arbeit ✓".encode("utf-8"))
        entry = Path(__file__).resolve().parents[1] / "conventional_wrapper.py"
        result = subprocess.run([sys.executable, "-B", str(entry), "--config", str(self.binding.path),
                                 "start", "--prompt-file", str(prompt)], capture_output=True, timeout=30)
        self.assertEqual(0, result.returncode, result.stderr.decode("utf-8"))
        results = [json.loads(line) for line in result.stdout.splitlines()]
        self.assertEqual(2, len(results))
        self.assertEqual(results[0]["runId"], results[1]["runId"])
        self.assertEqual("completed", results[1]["state"])
        self.assertEqual("NOT RUN", results[1]["taskAssessment"])
        self.assertEqual("nächste öffentliche Arbeit ✓", results[1]["prompt"])

    def test_disabled_mcp_stdio_never_launches_scripted_or_real_native_process(self):
        self.binding.config["execution_authorized"] = False
        self.binding.persist()
        entry = Path(__file__).resolve().parents[1] / "conventional_wrapper.py"
        messages = [
            {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25",
             "capabilities":{},"clientInfo":{"name":"offline","version":"1"}}},
            {"jsonrpc":"2.0","method":"notifications/initialized"},
            {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"conventional_start","arguments":{"prompt":"work"}}}]
        wire = ("\n".join(json.dumps(message) for message in messages)+"\n").encode("utf-8")
        result = subprocess.run([sys.executable,"-B",str(entry),"--config",str(self.binding.path),"mcp"],
                                input=wire,capture_output=True,timeout=30)
        self.assertEqual(0,result.returncode,result.stderr.decode("utf-8"))
        responses = [json.loads(line) for line in result.stdout.splitlines()]
        self.assertEqual(2,len(responses))
        self.assertTrue(responses[1]["result"]["isError"])
        self.assertIn("execution disabled",responses[1]["result"]["content"][0]["text"])
        self.assertFalse((self.binding.audit/"conventional-execution").exists())


if __name__ == "__main__":
    unittest.main()

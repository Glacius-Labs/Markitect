#!/usr/bin/env python3
"""G0 prototype referee: the Run remote and its rule keeper in one process.

Serves the Run's bare repository over Git smart HTTP (git http-backend as CGI)
with one Basic-auth credential per Manager, decides every push through the
repository's pre-receive hook, runs the declared check of every handoff, and
keeps the score. It never assigns work; it only refuses, checks, delivers and
counts. Throwaway prototype code (design section 13), not product code.

Layout inside the referee container:
  /run/config.json   written by the host before start (see load_config)
  /srv/run.git       the Run remote, seeded by the host
  /state/            score.json, events.jsonl, checks/
"""
import base64
import http.server
import json
import os
import shutil
import socketserver
import subprocess
import sys
import tempfile
import threading
import time
import urllib.parse

REPO = "/srv/run.git"
STATE = "/state"
KINDS = {"task", "question", "answer", "handoff", "review", "finding"}
VERDICTS = {"approve", "reject", "blocker"}
WAIT_MAX = 300
DEADLOCK_QUIET = 30


def git(*args, env=None, check=True):
    p = subprocess.run(["git", "-C", REPO, *args], capture_output=True, text=True, env=env)
    if check and p.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)}: {p.stderr.strip()}")
    return p.stdout.strip()


def is_ancestor(a, b, env=None):
    return subprocess.run(["git", "-C", REPO, "merge-base", "--is-ancestor", a, b],
                          capture_output=True, env=env).returncode == 0


class Refusal(Exception):
    pass


class Referee:
    def __init__(self, cfg):
        self.cfg = cfg
        self.phase = cfg["phase"]                  # assessment | implement | verify
        self.base = cfg["base"]
        self.tokens = cfg["credentials"]           # token -> manager
        self.parent = {m["name"]: m.get("parent") for m in cfg["managers"]}
        self.children = {n: [c for c, p in self.parent.items() if p == n] for n in self.parent}
        self.root = next(n for n, p in self.parent.items() if p is None)
        self.budget = cfg["messageBudget"]
        self.lock = threading.RLock()
        self.cond = threading.Condition(self.lock)
        self.inbox = {n: [] for n in self.parent}
        self.waiting = set()
        self.checks_running = 0
        self.quiet_since = None
        self.m = {n: {"tasked": False, "handoffs": [], "requiredBase": None, "stale": False}
                  for n in self.parent}
        self.messages = 0
        self.refusals = []
        self.final = None
        self.started = time.time()
        self.deliver(self.root, {"kind": "task", "from": "host", "text": cfg["rootTask"]})
        self.m[self.root]["tasked"] = True
        self.save()

    # --- bookkeeping -------------------------------------------------------
    def log(self, event):
        event["t"] = round(time.time() - self.started, 1)
        with open(os.path.join(STATE, "events.jsonl"), "a") as f:
            f.write(json.dumps(event) + "\n")

    def save(self):
        score = {"phase": self.phase, "base": self.base, "root": self.root,
                 "messages": self.messages, "messageBudget": self.budget,
                 "refusals": self.refusals, "final": self.final,
                 "elapsed": round(time.time() - self.started, 1),
                 "managers": self.m, "waiting": sorted(self.waiting),
                 "checksRunning": self.checks_running}
        tmp = os.path.join(STATE, "score.json.tmp")
        with open(tmp, "w") as f:
            json.dump(score, f, indent=1)
        os.replace(tmp, os.path.join(STATE, "score.json"))

    def deliver(self, to, event):
        self.inbox[to].append(event)
        self.log({"deliver": to, **{k: v for k, v in event.items() if k != "text"}})
        self.cond.notify_all()

    def end(self, outcome, **extra):
        if self.final:
            return
        self.final = {"outcome": outcome, **extra}
        self.log({"end": self.final})
        for n in self.parent:
            self.inbox[n].append({"kind": "run_end", "outcome": outcome})
        self.save()
        self.cond.notify_all()

    def latest(self, name):
        h = self.m[name]["handoffs"]
        return h[-1] if h else None

    # --- the push rules ----------------------------------------------------
    def decide(self, user, updates, env):
        """Called from the pre-receive hook. Raises Refusal or applies state."""
        with self.lock:
            if self.final:
                raise Refusal(f"the Run has ended ({self.final['outcome']})")
            own = f"refs/heads/mk/{user}/"
            talk_updates = []
            work_new = None
            for old, new, ref in updates:
                if new == "0" * 40:
                    raise Refusal(f"{ref}: deleting refs is not allowed")
                if ref == "refs/heads/mk/notes":
                    self.check_notes(old, new, env)
                    continue
                if not ref.startswith(own) or ref[len(own):] not in ("work", "talk", "wip"):
                    raise Refusal(f"{ref}: {user} may write only mk/{user}/work, mk/{user}/talk, mk/{user}/wip and mk/notes")
                kind = ref[len(own):]
                if old != "0" * 40 and not is_ancestor(old, new, env):
                    raise Refusal(f"{ref}: only forward (fast-forward) updates are allowed")
                if kind == "work":
                    if not is_ancestor(self.base, new, env):
                        raise Refusal(f"{ref}: must descend from mk/base {self.base[:12]}")
                    if self.phase != "implement" and new != self.base:
                        raise Refusal(f"{ref}: in {self.phase} the work branch stays at mk/base; hand off a judgment without code")
                    work_new = new
                elif kind == "talk":
                    talk_updates.append((old, new))
            for old, new in talk_updates:
                rng = new if old == "0" * 40 else f"{old}..{new}"
                for c in git("rev-list", "--reverse", rng, env=env).split():
                    self.message(user, c, work_new, env)
            self.save()

    def check_notes(self, old, new, env):
        rng = new if old == "0" * 40 else f"{old}..{new}"
        for c in git("rev-list", rng, env=env).split():
            st = git("diff-tree", "--no-commit-id", "-r", "--root", "--name-status", c, env=env)
            for line in st.splitlines():
                if not line.startswith("A\t") or not line.split("\t", 1)[1].startswith("notes/"):
                    raise Refusal("mk/notes: each commit may only add new files under notes/")

    def message(self, user, commit, work_new, env):
        st = git("diff-tree", "--no-commit-id", "-r", "--root", "--name-status", commit, env=env).splitlines()
        if len(st) != 1 or not st[0].startswith("A\tmessages/"):
            raise Refusal(f"talk commit {commit[:12]}: must add exactly one new file under messages/ and change nothing else")
        path = st[0].split("\t", 1)[1]
        try:
            msg = json.loads(git("show", f"{commit}:{path}", env=env))
        except ValueError as e:
            raise Refusal(f"{path}: not valid JSON ({e})")
        kind, to = msg.get("kind"), msg.get("to")
        if kind not in KINDS:
            raise Refusal(f"{path}: kind must be one of {sorted(KINDS)}")
        if self.messages >= self.budget:
            self.end("incomplete", reason="message budget exhausted")
            raise Refusal("message budget exhausted; the Run ends")
        if kind == "handoff":
            self.handoff(user, msg, work_new, env)
        else:
            allowed = set(self.children[user]) | ({self.parent[user]} - {None})
            if to not in allowed:
                raise Refusal(f"{path}: {user} may write only to its parent or direct children: {sorted(allowed)}")
            if kind == "review":
                self.review(user, to, msg)
            elif kind == "task":
                if to not in self.children[user]:
                    raise Refusal(f"{path}: tasks go to direct children only")
                self.m[to]["tasked"] = True
            self.deliver(to, {"kind": kind, "from": user, "text": msg.get("text", ""),
                              **{k: msg[k] for k in ("commit", "accept", "findings") if k in msg}})
        self.messages += 1
        self.log({"message": kind, "from": user, "to": to or self.parent[user], "commit": commit})

    def handoff(self, user, msg, work_new, env):
        commit = msg.get("commit", "")
        verdict = msg.get("verdict")
        tip = work_new or git("rev-parse", f"refs/heads/mk/{user}/work", env=env)
        if commit != tip:
            raise Refusal(f"handoff must name your pushed work tip {tip[:12]}, not {commit[:12] or 'nothing'}")
        if self.phase != "implement":
            if verdict not in VERDICTS:
                raise Refusal(f"in {self.phase} a handoff is a judgment: verdict must be one of {sorted(VERDICTS)}")
        elif verdict == "blocker":
            pass
        elif verdict not in (None, "submit"):
            raise Refusal("in implement a handoff submits code; verdict may only be 'blocker'")
        req = self.m[user]["requiredBase"]
        if req and not is_ancestor(req, commit, env):
            raise Refusal(f"your parent rejected your last handoff at its work tip {req[:12]}; merge it first (mk-sync), then hand off again")
        if verdict != "blocker":
            for c in self.children[user]:
                h = self.latest(c)
                if not h or not h.get("accepted"):
                    raise Refusal(f"your child {c} has no accepted latest handoff yet; review its handoff first")
        h = {"commit": commit, "verdict": verdict or "submit", "text": msg.get("text", ""),
             "findings": msg.get("findings", []), "check": "pending", "accepted": False}
        self.m[user]["handoffs"].append(h)
        self.m[user]["stale"] = False
        # a new handoff withdraws every ancestor's handoff built on the old one
        a = self.parent[user]
        while a is not None:
            la = self.latest(a)
            if la and not self.m[a]["stale"] and (la.get("accepted") or a == self.root):
                self.m[a]["stale"] = True
                la["accepted"] = False
                self.deliver(a, {"kind": "reopened", "from": "referee",
                                 "text": f"{user} handed off again; your earlier handoff no longer covers its subtree"})
            a = self.parent[a]
        need_check = self.phase == "implement" and verdict != "blocker"
        if not need_check:
            h["check"] = "not-applicable"
            self.after_check(user, h)
        else:
            self.checks_running += 1
            threading.Thread(target=self.run_check, args=(user, h), daemon=True).start()

    def review(self, user, child, msg):
        h = self.latest(child)
        if not h:
            raise Refusal(f"{child} has not handed off anything to review")
        if msg.get("commit") != h["commit"]:
            raise Refusal(f"a review must name {child}'s latest handoff {h['commit'][:12]}")
        if h["check"] == "pending":
            raise Refusal(f"the check of {child}'s handoff is still running; wait for it")
        if msg.get("accept") is True:
            if h["check"] == "fail":
                raise Refusal(f"{child}'s handoff failed its check; it cannot be accepted")
            h["accepted"] = True
            self.m[child]["requiredBase"] = None
        elif msg.get("accept") is False:
            tip = git("rev-parse", f"refs/heads/mk/{user}/work")
            self.m[child]["requiredBase"] = tip
            h["accepted"] = False
        else:
            raise Refusal("a review needs accept: true or false")

    # --- checks ------------------------------------------------------------
    def run_check(self, user, h):
        commit = h["commit"]
        deadline = time.time() + 60
        while time.time() < deadline and subprocess.run(
                ["git", "-C", REPO, "cat-file", "-e", commit + "^{commit}"]).returncode != 0:
            time.sleep(0.5)
        out_dir = os.path.join(STATE, "checks")
        os.makedirs(out_dir, exist_ok=True)
        work = tempfile.mkdtemp(prefix="check-")
        log_path = os.path.join(out_dir, f"{user.replace('/', '_')}-{commit[:12]}.log")
        ok = False
        try:
            subprocess.run(f"git -C {REPO} archive {commit} | tar -x -C {work}", shell=True, check=True)
            with open(log_path, "w") as log:
                ok = True
                for cmd in self.cfg["checkCommands"]:
                    log.write(f"$ {cmd}\n")
                    log.flush()
                    try:
                        p = subprocess.run(cmd, shell=True, cwd=work, stdout=log, stderr=subprocess.STDOUT,
                                           timeout=self.cfg.get("checkTimeout", 300))
                        rc = p.returncode
                    except subprocess.TimeoutExpired:
                        rc = "timeout"
                    log.write(f"exit {rc}\n")
                    if rc != 0:
                        ok = False
                        break
        except Exception as e:
            with open(log_path, "a") as log:
                log.write(f"check infrastructure error: {e}\n")
        finally:
            shutil.rmtree(work, ignore_errors=True)
        with open(log_path) as f:
            tail = f.read()[-3000:]
        with self.lock:
            self.checks_running -= 1
            h["check"] = "pass" if ok else "fail"
            h["checkOutput"] = tail
            self.log({"check": h["check"], "manager": user, "commit": commit})
            self.after_check(user, h)
            self.save()

    def after_check(self, user, h):
        """Route a handoff once its check result is known."""
        if user == self.root:
            self.root_final(h)
            return
        self.deliver(self.parent[user], {"kind": "handoff", "from": user, "commit": h["commit"],
                                         "verdict": h["verdict"], "check": h["check"],
                                         "checkOutput": h.get("checkOutput", ""),
                                         "findings": h["findings"], "text": h["text"]})
        if h["check"] == "fail":
            self.deliver(user, {"kind": "check", "from": "referee", "commit": h["commit"], "check": "fail",
                                "checkOutput": h.get("checkOutput", ""),
                                "text": "your handoff failed its check; your parent sees the result too"})

    def root_final(self, h):
        if h["verdict"] == "blocker":
            self.end("model_blocker", judgment="blocker", text=h["text"], findings=h["findings"])
            return
        missing = [n for n in self.parent if n != self.root and not (self.latest(n) or {}).get("accepted")]
        if missing:
            self.deliver(self.root, {"kind": "refused", "from": "referee",
                                     "text": f"final judgment not valid: no accepted latest handoff from {missing}"})
            return
        if h["check"] == "fail":
            self.deliver(self.root, {"kind": "check", "from": "referee", "commit": h["commit"], "check": "fail",
                                     "checkOutput": h.get("checkOutput", ""),
                                     "text": "your final handoff failed the complete check"})
            return
        h["accepted"] = True
        self.end("completed", judgment=h["verdict"], commit=h["commit"], text=h["text"], findings=h["findings"])

    # --- waiting and deadlock ----------------------------------------------
    def wait(self, user, timeout):
        with self.lock:
            self.waiting.add(user)
            self.save()
            end = time.time() + min(timeout, WAIT_MAX)
            while not self.inbox[user] and time.time() < end and not self.final:
                self.cond.wait(end - time.time())
            events, self.inbox[user] = self.inbox[user], []
            self.waiting.discard(user)
            self.save()
            return events

    def is_open(self, user):
        """Whether a Manager owes work: undelivered events or an owed handoff."""
        with self.lock:
            st = self.m[user]
            h = self.latest(user)
            owes = st["tasked"] and (h is None or st["stale"] or
                                     (h["check"] != "pending" and not h.get("accepted") and st["requiredBase"]) or
                                     h["check"] == "fail")
            # a parent whose children are still working is waiting, not idle: no nudge
            children_done = all((self.latest(c) or {}).get("accepted") for c in self.children[user])
            return {"open": bool(self.inbox[user]) or bool(owes and children_done), "queued": len(self.inbox[user]),
                    "owesHandoff": bool(owes), "ended": bool(self.final)}

    def watchdog(self):
        while not self.final:
            time.sleep(5)
            with self.lock:
                if time.time() - self.started > self.cfg["deadlineSeconds"]:
                    self.end("incomplete", reason="deadline passed")
                    break
                idle = (self.waiting == set(self.parent) and self.checks_running == 0
                        and not any(self.inbox.values()))
                if not idle:
                    self.quiet_since = None
                    continue
                self.quiet_since = self.quiet_since or time.time()
                if time.time() - self.quiet_since > DEADLOCK_QUIET:
                    self.end("incomplete", reason="deadlock: every Manager waits and nobody owes an answer")
                    break


REF = None


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        pass

    def user(self):
        auth = self.headers.get("Authorization", "")
        if auth.startswith("Basic "):
            name, _, token = base64.b64decode(auth[6:]).decode().partition(":")
            if REF.tokens.get(token) == name:
                return name
        return None

    def reply(self, code, obj=None, body=None, ctype="application/json", extra=None):
        data = body if body is not None else json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        for k, v in (extra or {}).items():
            self.send_header(k, v)
        self.end_headers()
        self.wfile.write(data)

    def body(self):
        if self.headers.get("Transfer-Encoding", "").lower() == "chunked":
            out = bytearray()
            while True:
                size = int(self.rfile.readline().strip().split(b";")[0], 16)
                if size == 0:
                    self.rfile.readline()
                    return bytes(out)
                out += self.rfile.read(size)
                self.rfile.readline()
        n = int(self.headers.get("Content-Length") or 0)
        return self.rfile.read(n) if n else b""

    def do_GET(self):
        self.route()

    def do_POST(self):
        self.route()

    def route(self):
        url = urllib.parse.urlsplit(self.path)
        if url.path == "/hook":                                   # local hook only
            if self.client_address[0] != "127.0.0.1":
                return self.reply(403, {"error": "local only"})
            req = json.loads(self.body())
            try:
                REF.decide(req["user"], req["updates"], {**os.environ, **req["env"]})
                return self.reply(200, {"ok": True})
            except Refusal as e:
                with REF.lock:
                    REF.refusals.append({"user": req["user"], "reason": str(e)})
                    REF.log({"refused": req["user"], "reason": str(e)})
                    REF.save()
                return self.reply(200, {"ok": False, "reason": str(e)})
        user = self.user()
        if not user:
            return self.reply(401, {"error": "credentials required"},
                              extra={"WWW-Authenticate": 'Basic realm="run"'})
        q = dict(urllib.parse.parse_qsl(url.query))
        if url.path.startswith("/git/"):
            return self.cgi(user, url)
        if url.path == "/wait":
            return self.reply(200, REF.wait(user, int(q.get("timeout", 60))))
        if url.path == "/open":
            return self.reply(200, REF.is_open(user))
        if url.path == "/handoff":                                 # a child's latest handoff, for parents
            child = q.get("child", "")
            if REF.parent.get(child) != user:
                return self.reply(403, {"error": f"{child} is not your direct child"})
            with REF.lock:
                return self.reply(200, REF.latest(child) or {})
        if url.path == "/me":
            with REF.lock:
                return self.reply(200, {"name": user, "parent": REF.parent[user], "children": REF.children[user],
                                        "phase": REF.phase, "base": REF.base, "state": REF.m[user]})
        self.reply(404, {"error": "unknown path"})

    def cgi(self, user, url):
        body = self.body()
        env = {"PATH": os.environ["PATH"], "GIT_PROJECT_ROOT": "/srv", "GIT_HTTP_EXPORT_ALL": "1",
               "REMOTE_USER": user, "REMOTE_ADDR": self.client_address[0],
               "REQUEST_METHOD": self.command, "PATH_INFO": url.path[len("/git"):],
               "QUERY_STRING": url.query, "CONTENT_TYPE": self.headers.get("Content-Type", ""),
               "CONTENT_LENGTH": str(len(body)),
               "HTTP_CONTENT_ENCODING": self.headers.get("Content-Encoding", ""),
               "GIT_PROTOCOL": self.headers.get("Git-Protocol", "")}
        p = subprocess.run(["git", "http-backend"], input=body, capture_output=True, env=env)
        head, _, payload = p.stdout.partition(b"\r\n\r\n")
        status, headers = 200, {}
        for line in head.decode(errors="replace").split("\r\n"):
            k, _, v = line.partition(":")
            if k.lower() == "status":
                status = int(v.strip().split()[0])
            elif k:
                headers[k.strip()] = v.strip()
        ctype = headers.pop("Content-Type", "application/octet-stream")
        self.reply(status, body=payload, ctype=ctype, extra=headers)


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True
    allow_reuse_address = True


def seed(cfg):
    """Create the Run remote from the host's start bundle: mk/base and every work ref at the base."""
    subprocess.run(["git", "init", "-q", "--bare", "-b", "mk/base", REPO], check=True)
    git("config", "http.receivepack", "true")
    git("config", "receive.denyDeletes", "true")
    git("config", "uploadpack.allowAnySHA1InWant", "true")
    git("fetch", "-q", "/run/start.bundle", f"+{cfg['baseRef']}:refs/heads/mk/base")
    if git("rev-parse", "refs/heads/mk/base") != cfg["base"]:
        raise RuntimeError("start bundle does not hold the configured base")
    for m in cfg["managers"]:
        git("update-ref", f"refs/heads/mk/{m['name']}/work", cfg["base"])
    git("update-ref", "refs/heads/mk/notes", cfg["base"])
    hook = os.path.join(REPO, "hooks", "pre-receive")
    shutil.copy("/mk/bin/pre-receive", hook)
    os.chmod(hook, 0o755)


def main():
    global REF
    with open("/run/config.json") as f:
        cfg = json.load(f)
    os.makedirs(STATE, exist_ok=True)
    seed(cfg)
    REF = Referee(cfg)
    threading.Thread(target=REF.watchdog, daemon=True).start()
    print("referee listening on :8080", flush=True)
    Server(("0.0.0.0", 8080), Handler).serve_forever()


if __name__ == "__main__":
    sys.exit(main())

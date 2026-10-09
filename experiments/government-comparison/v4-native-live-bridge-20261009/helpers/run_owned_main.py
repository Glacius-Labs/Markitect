"""Bound one own Main subprocess; never starts a native Actor or retries."""
import argparse,hashlib,json,subprocess,time
from pathlib import Path

def main():
    p=argparse.ArgumentParser();p.add_argument('spec');a=p.parse_args()
    spec=json.loads(Path(a.spec).read_text(encoding='utf-8-sig'))
    root=Path(spec['state']);root.mkdir(exist_ok=True)
    start=time.time();end=min(start+spec['maxSeconds'],spec['notAfterEpoch'])
    receipt={'command':spec['command'],'args':spec['args'],'cwd':spec['cwd'],
             'reservedEpoch':start,'endEpoch':end,'specSha256':hashlib.sha256(Path(a.spec).read_bytes()).hexdigest()}
    child=None
    with (root/'main.stdout.json').open('xb') as out,(root/'main.stderr.log').open('xb') as err:
        try:
            child=subprocess.Popen([spec['command'],*spec['args']],cwd=spec['cwd'],stdout=out,stderr=err)
            receipt.update(pid=child.pid,startedEpoch=time.time())
            (root/'process-receipt.json').write_text(json.dumps(receipt))
            child.wait(timeout=max(.01,end-time.time()))
        finally:
            if child is not None:
                if child.poll() is None:
                    child.terminate()
                    try:child.wait(timeout=5)
                    except subprocess.TimeoutExpired:child.kill();child.wait(timeout=5)
                receipt.update(exitCode=child.returncode,terminalEpoch=time.time(),ownHandleOnly=True)
                (root/'process-receipt.json').write_text(json.dumps(receipt))

if __name__=='__main__':main()

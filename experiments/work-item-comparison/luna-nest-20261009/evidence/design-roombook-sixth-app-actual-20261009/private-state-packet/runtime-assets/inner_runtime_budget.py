"""External file-only budget gate around a separately pinned genuine adapter.

No prompts, model API, response decoding or product-role implementation. Disabled
until Root supplies an immutable binding and exact enabled cell grant.
"""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time
from datetime import timedelta
import app_ledger
import nest

OLD_ADAPTER = '0ceb15e6cfa28de4256d346d61de64a207c307405be4a3549b29bb4f380fc4ac'
MAX_INPUT = 32 * 1024 * 1024


class OwnedTree:
    """Only the process/group/job created by this invocation; no inventory."""
    def __init__(self, process):
        self.process = process
        self.job = None
        if os.name == 'nt':
            from ctypes import wintypes as w
            self.api = ctypes.WinDLL('kernel32', use_last_error=True)
            for name, args, result in [
                ('CreateJobObjectW', [ctypes.c_void_p, w.LPCWSTR], ctypes.c_void_p),
                ('SetInformationJobObject', [ctypes.c_void_p, ctypes.c_int, ctypes.c_void_p, w.DWORD], w.BOOL),
                ('AssignProcessToJobObject', [ctypes.c_void_p, ctypes.c_void_p], w.BOOL),
                ('QueryInformationJobObject', [ctypes.c_void_p, ctypes.c_int, ctypes.c_void_p, w.DWORD, ctypes.c_void_p], w.BOOL),
                ('TerminateJobObject', [ctypes.c_void_p, w.UINT], w.BOOL),
                ('CloseHandle', [ctypes.c_void_p], w.BOOL)]:
                fn = getattr(self.api, name); fn.argtypes = args; fn.restype = result
            class Limits(ctypes.Structure):
                _fields_ = [('user',ctypes.c_longlong),('jobuser',ctypes.c_longlong),('flags',w.DWORD),
                            ('minws',ctypes.c_size_t),('maxws',ctypes.c_size_t),('active',w.DWORD),
                            ('affinity',ctypes.c_size_t),('priority',w.DWORD),('scheduling',w.DWORD)]
            class Extended(ctypes.Structure):
                _fields_ = [('basic',Limits),('io',ctypes.c_ulonglong * 6),
                            ('memory',ctypes.c_size_t * 4)]
            class Accounting(ctypes.Structure):
                _fields_ = [('times',ctypes.c_longlong * 4),('faults',w.DWORD),
                            ('total',w.DWORD),('active',w.DWORD),('terminated',w.DWORD)]
            self.Accounting = Accounting
            self.job = self.api.CreateJobObjectW(None, None)
            info = Extended(); info.basic.flags = 0x2000  # KILL_ON_JOB_CLOSE
            if not self.job or not self.api.SetInformationJobObject(self.job,9,ctypes.byref(info),ctypes.sizeof(info)):
                self.close(); raise OSError('Could not establish owned Job Object')
            if not self.api.AssignProcessToJobObject(self.job, int(process._handle)):
                self.close(); raise OSError('Could not attach gated own child to Job Object')

    def active(self):
        if os.name == 'nt':
            info = self.Accounting()
            if not self.api.QueryInformationJobObject(self.job,1,ctypes.byref(info),ctypes.sizeof(info),None):
                raise OSError('Own job query failed')
            return info.active != 0
        try:
            os.killpg(self.process.pid,0); return True
        except ProcessLookupError:
            return False

    def stop(self, seconds=3):
        end = time.monotonic() + seconds
        try:
            if os.name == 'nt':
                if not self.api.TerminateJobObject(self.job,124):
                    return {'localTreeExitConfirmed':False,'error':'own-job-termination-failed'}
            else:
                try: os.killpg(self.process.pid,signal.SIGKILL)
                except ProcessLookupError: pass
            while time.monotonic() < end:
                self.process.poll()
                if not self.active():
                    self.process.wait(timeout=max(.01,end-time.monotonic()))
                    return {'localTreeExitConfirmed':True,'scope':'own Windows Job Object or own POSIX process group'}
                time.sleep(.01)
            return {'localTreeExitConfirmed':False,'error':'own-tree-cleanup-timeout'}
        except (OSError,subprocess.TimeoutExpired):
            return {'localTreeExitConfirmed':False,'error':'own-tree-cleanup-unconfirmed'}

    def close(self):
        if self.job:
            self.api.CloseHandle(self.job); self.job = None


def parent_entry(binding, child=None):
    state = nest.read(Path(binding['stateDir'])/'state.json')
    ledger = nest.read(Path(binding['stateDir'])/'app-ledger.json')
    parent = next(e for e in ledger['entries'] if e['reservation']==binding['parentReservation'])
    kind = 'assessment' if binding['phase']=='assessment' else 'implementation'
    if (parent['status'] not in ('reserved','started') or parent['depth'] != 1 or
        parent['kind'] != kind or parent['stationIndex'] != state['stationIndex'] or
        parent.get('handle') != binding['parentHandle'] or ledger['status']=='closed' or
        any(e['status']=='unknown-stop' for e in ledger['entries'])):
        raise ValueError('Exact active own parent/phase/station required')
    if child and child['stationIndex'] != parent['stationIndex']:
        raise ValueError('Station changed after reservation')
    return parent


def run(binding, raw, command):
    if binding.get('executionAuthorized') is not True:
        raise ValueError('Disabled resource binding; no process or reservation')
    if binding['phase'] not in ('implementation','assessment'):
        raise ValueError('Unknown resource phase')
    if nest.sha(binding['grantPath']) != binding['grantSha256']:
        raise ValueError('Immutable enabled cell grant hash differs')
    parent_entry(binding)
    kind = 'assessment-helper' if binding['phase']=='assessment' else 'helper'
    entry = app_ledger.reserve(binding['stateDir'],binding['grantPath'],kind,
                               depth=2,parent=binding['parentReservation'])
    receipt = {'reservation':entry['reservation'],'parent':binding['parentReservation'],
               'phase':binding['phase'],'stationIndex':entry['stationIndex'],
               'inputSha256':hashlib.sha256(raw).hexdigest(),'requestBytes':len(raw),
               'requestedModel':'gpt-6-luna','reasoning':'high','actualServingModel':None,
               'usage':None,'cost':None,'providerModelStartUtc':None,'argv':command}
    process = tree = None
    handle = None
    status = 'call-failed'
    try:
        parent = parent_entry(binding,entry)
        if len(raw)>MAX_INPUT or command!=binding['command']:
            raise ValueError('Input bound or exact command differs')
        config = json.loads(os.environ.get('MARKITECT_AGENT_CONFIG_JSON','{}'))
        if config.get('model')!='gpt-6-luna' or config.get('modelOptions',{}).get('model_reasoning_effort')!='high':
            raise ValueError('Exact inherited Luna High configuration required')
        if not binding.get('runtimePins'):
            raise ValueError('Exact executable/adapter runtime pins required')
        for pin in binding['runtimePins']:
            if pin['sha256']==OLD_ADAPTER or nest.sha(pin['path'])!=pin['sha256']:
                raise ValueError('Runtime pin mismatch or superseded adapter')
        pinned = {Path(pin['path']).resolve() for pin in binding['runtimePins']}
        executable = Path(command[0]) if command else Path('')
        command_files = {Path(arg).resolve() for arg in command if Path(arg).is_file()}
        if (not executable.is_absolute() or not executable.is_file() or
            executable.resolve() not in pinned or len(command_files)<2 or
            not command_files.issubset(pinned)):
            raise ValueError('Exact executable and adapter/file arguments must be pinned')
        seconds = binding['maxOwnedWallSeconds']
        if not isinstance(seconds,(int,float)) or not 0<seconds<=600:
            raise ValueError('Owned role wall bound must be positive and <=600s')
        end = min(app_ledger.date(entry['deadlineUtc']),app_ledger.date(parent['deadlineUtc']),
                  app_ledger.date(entry['reservedUtc'])+timedelta(seconds=seconds))
        cleanup = binding.get('cleanupReserveSeconds',10)
        # Owned stop may take 3 seconds and pipe drain another second. Keep at
        # least one further second before the sealed deadline for recording.
        if not 5<=cleanup<=15 or (end-app_ledger.now()).total_seconds()<=cleanup:
            raise ValueError('No remaining bounded role/cleanup window')
        receipt['effectiveDeadlineUtc']=end.isoformat()
        # Child blocks on a single resource gate byte before launch of the exact
        # original command. It cannot call the genuine adapter before attachment.
        process = subprocess.Popen([sys.executable,'-B',str(Path(__file__).resolve()),
                                    '--guard-child','--',*command],stdin=subprocess.PIPE,
                                   stdout=subprocess.PIPE,stderr=subprocess.PIPE,
                                   start_new_session=os.name!='nt')
        handle = f'process:{process.pid}:reservation:{entry["reservation"]}'
        receipt.update(ownPid=process.pid,createdUtc=nest.utc(),handle=handle)
        tree = OwnedTree(process)
        parent_entry(binding,entry)
        app_ledger.record(binding['stateDir'],entry['reservation'],'started',handle,receipt)
        timeout = (end-app_ledger.now()).total_seconds()-cleanup
        if timeout<=0: raise subprocess.TimeoutExpired(command,0)
        receipt['adapterReleaseUtc']=nest.utc()
        try:
            stdout,stderr=process.communicate(input=b'\x01'+raw,timeout=timeout)
            code=process.returncode
            stop=tree.stop() if tree.active() else {'localTreeExitConfirmed':True}
            status='returned' if code==0 else 'call-failed'
        except subprocess.TimeoutExpired:
            stop=tree.stop(); code=124
            try:
                stdout,stderr=process.communicate(timeout=1)
            except subprocess.TimeoutExpired as drain:
                stdout,stderr=drain.output or b'',drain.stderr or b''
                stop={'localTreeExitConfirmed':False,'error':'owned-output-drain-unconfirmed',
                      'priorStop':stop}
            status='interrupted'
        if not stop['localTreeExitConfirmed']: status='unknown-stop';code=125
        receipt.update(stop=stop,exitCode=code,stdoutSha256=hashlib.sha256(stdout).hexdigest(),
                       stderrSha256=hashlib.sha256(stderr).hexdigest(),endedUtc=nest.utc())
        return code,stdout,stderr,receipt
    except BaseException as error:
        receipt['errorType']=type(error).__name__
        if tree:
            stop=tree.stop();receipt['stop']=stop
            if not stop['localTreeExitConfirmed']: status='unknown-stop'
        elif process:
            process.kill()
            try: process.wait(timeout=3)
            except subprocess.TimeoutExpired: status='unknown-stop'
        raise
    finally:
        if tree: tree.close()
        receipt['terminalRecordedUtc']=nest.utc()
        app_ledger.record(binding['stateDir'],entry['reservation'],status,handle,receipt)
        nest.write(Path(binding['stateDir'])/'inner-receipts'/f'{entry["reservation"]:03d}.json',receipt)


def main():
    if len(sys.argv)>2 and sys.argv[1:3]==['--guard-child','--']:
        if os.read(0,1)!=b'\x01': return 125
        # Windows execv does not preserve the replacement process exit code.
        # Inherit the already-gated raw streams and unchanged environment/argv;
        # the child inherits this guard's owned Job Object/process group.
        child=subprocess.Popen(sys.argv[3:],stdin=sys.stdin.buffer,
                               stdout=sys.stdout.buffer,stderr=sys.stderr.buffer)
        return child.wait()
    parser=argparse.ArgumentParser()
    parser.add_argument('--binding',required=True)
    parser.add_argument('--binding-sha256',required=True)
    parser.add_argument('command',nargs=argparse.REMAINDER)
    args=parser.parse_args()
    try:
        if nest.sha(args.binding)!=args.binding_sha256: raise ValueError('Binding hash differs')
        command=args.command[1:] if args.command[:1]==['--'] else args.command
        code,out,err,_=run(nest.read(args.binding),sys.stdin.buffer.read(MAX_INPUT+1),command)
        sys.stdout.buffer.write(out);sys.stderr.buffer.write(err)
        return code
    except Exception as error:
        print('Shared resource gate rejected or stopped invocation: '+type(error).__name__,file=sys.stderr)
        return 125


if __name__=='__main__': raise SystemExit(main())

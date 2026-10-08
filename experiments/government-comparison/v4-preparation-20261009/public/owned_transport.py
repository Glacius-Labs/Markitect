"""Own-child dynamic URL transport only; no HTTP assertion semantics."""
import datetime, json, os, pathlib, queue, re, subprocess, threading, time

class Server:
    def __init__(self, dll, database, state, label, deadline_utc):
        self.dll=pathlib.Path(dll).resolve();self.db=pathlib.Path(database).resolve()
        self.state=pathlib.Path(state).resolve();self.label=label;self.deadline=datetime.datetime.fromisoformat(deadline_utc.replace('Z','+00:00')).timestamp();self.child=None
    def __enter__(self):
        assert time.time()<self.deadline
        record=self.state/'owned-server-starts.json'
        starts=json.loads(record.read_text()) if record.exists() else []
        assert len(starts)<12,'server start cap'
        env=os.environ.copy();env['ASPNETCORE_URLS']='http://127.0.0.1:0';env['ORDER_DB_PATH']=str(self.db)
        item={'label':self.label,'reservedUtc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'dll':str(self.dll),'database':str(self.db),'urls':env['ASPNETCORE_URLS']}
        starts.append(item);record.write_text(json.dumps(starts,indent=2)+'\n')
        try:
            self.child=subprocess.Popen(['dotnet',str(self.dll)],cwd=str(self.state),env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,bufsize=1)
            item.update(pid=self.child.pid,startedUtc=datetime.datetime.now(datetime.timezone.utc).isoformat());record.write_text(json.dumps(starts,indent=2)+'\n')
            messages=queue.Queue()
            def read():
                with (self.state/(self.label+'.server.log')).open('w',encoding='utf8') as log:
                    for line in self.child.stdout:log.write(line);log.flush();messages.put(line)
            self.reader=threading.Thread(target=read,daemon=True);self.reader.start()
            limit=min(time.time()+25,self.deadline-5)
            while time.time()<limit:
                try:line=messages.get(timeout=.2)
                except queue.Empty:
                    if self.child.poll() is not None:break
                    continue
                if '10013' in line:raise RuntimeError('own dynamic bind10013; no retry authorized')
                match=re.search(r'Now listening on:\s*(http://127\.0\.0\.1:\d+)',line)
                if match:
                    self.url=match.group(1);item['url']=self.url;record.write_text(json.dumps(starts,indent=2)+'\n');return self
            raise RuntimeError('no own stdout URL; no retry authorized')
        except BaseException:
            self.__exit__(None,None,None);raise
    def __exit__(self,*args):
        if self.child is not None:
            if self.child.poll() is None:
                self.child.terminate()
                try:self.child.wait(timeout=5)
                except subprocess.TimeoutExpired:self.child.kill();self.child.wait(timeout=5)
            if hasattr(self,'reader'):self.reader.join(timeout=2)
            if self.child.stdout:self.child.stdout.close()
            p=self.state/'owned-server-starts.json';a=json.loads(p.read_text());r=next(x for x in reversed(a) if x.get('pid')==self.child.pid and x['label']==self.label)
            r.update(exitCode=self.child.returncode,stoppedUtc=datetime.datetime.now(datetime.timezone.utc).isoformat());p.write_text(json.dumps(a,indent=2)+'\n')
